package s3api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// CreateWebsiteHandler orchestrates the creation of a new s3 bucket website with rollback in
// the event of failure.  The operations are:
// 1. create the bucket with the given name
// 2. tag the bucket with given tags
// 3. apply the website configuration to the bucket
// 4. generate the default admin bucket policy
// 5. create the admin bucket policy
// 6. create the bucket admin group, '<bucketName>-BktAdmGrp'
// 7. attach the bucket admin policy to the bucket admin group
// Note: this does _not_ create any users for managing the bucket
func (s *server) CreateWebsiteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	s3Service, ok := s.s3Services[account]
	if !ok {
		msg := fmt.Sprintf("s3 service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	iamService, ok := s.iamServices[account]
	if !ok {
		msg := fmt.Sprintf("IAM service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	var req struct {
		Tags                 []*s3.Tag
		BucketInput          s3.CreateBucketInput
		WebsiteConfiguration s3.WebsiteConfiguration
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create website input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// setup err var, rollback function list and defer execution
	var rollBackTasks []func() error
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	bucketName := aws.StringValue(req.BucketInput.Bucket)
	bucketOutput, err := s3Service.CreateBucket(r.Context(), &req.BucketInput)
	if err != nil {
		msg := fmt.Sprintf("failed to create bucket %s", bucketName)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append bucket delete to rollback tasks
	rbfunc := func() error {
		return func() error {
			if _, err := s3Service.DeleteEmptyBucket(r.Context(), &s3.DeleteBucketInput{Bucket: aws.String(bucketName)}); err != nil {
				return err
			}
			return nil
		}()
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	err = s3Service.TagBucket(r.Context(), bucketName, req.Tags)
	if err != nil {
		msg := fmt.Sprintf("failed to tag bucket %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	err = s3Service.UpdateWebsiteConfig(r.Context(), &s3.PutBucketWebsiteInput{
		Bucket:               aws.String(bucketName),
		WebsiteConfiguration: &req.WebsiteConfiguration,
	})
	if err != nil {
		msg := fmt.Sprintf("failed to configure bucket %s as website: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	defaultWebsitePolicy, err := iamService.DefaultWebsiteAccessPolicy(aws.String(bucketName))
	if err != nil {
		msg := fmt.Sprintf("failed building default website bucket access policy for %s: %s", bucketName, err.Error())
		handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
		return
	}

	err = s3Service.UpdateBucketPolicy(r.Context(), &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucketName),
		Policy: aws.String(string(defaultWebsitePolicy)),
	})

	// build the default IAM bucket admin policy (from the config and known inputs)
	defaultPolicy, err := iamService.DefaultBucketAdminPolicy(aws.String(bucketName))
	if err != nil {
		msg := fmt.Sprintf("failed building default IAM policy for bucket %s: %s", bucketName, err.Error())
		handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
		return
	}

	policyOutput, err := iamService.CreatePolicy(r.Context(), &iam.CreatePolicyInput{
		Description:    aws.String(fmt.Sprintf("Admin policy for %s bucket", bucketName)),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String(fmt.Sprintf("%s-BktAdmPlc", bucketName)),
	})

	if err != nil {
		msg := fmt.Sprintf("failed to create policy: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append policy delete to rollback tasks
	rbfunc = func() error {
		return func() error {
			if _, err := iamService.DeletePolicy(r.Context(), &iam.DeletePolicyInput{PolicyArn: policyOutput.Policy.Arn}); err != nil {
				return err
			}
			return nil
		}()
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	groupName := fmt.Sprintf("%s-BktAdmGrp", bucketName)
	group, err := iamService.CreateGroup(r.Context(), &iam.CreateGroupInput{
		GroupName: aws.String(groupName),
	})

	if err != nil {
		msg := fmt.Sprintf("failed to create group: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append group delete to rollback tasks
	rbfunc = func() error {
		return func() error {
			if _, err := iamService.DeleteGroup(r.Context(), &iam.DeleteGroupInput{GroupName: aws.String(groupName)}); err != nil {
				return err
			}
			return nil
		}()
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	if _, err = iamService.AttachGroupPolicy(r.Context(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String(groupName),
		PolicyArn: policyOutput.Policy.Arn,
	}); err != nil {
		msg := fmt.Sprintf("failed to attach policy %s to group %s: %s", aws.StringValue(policyOutput.Policy.Arn), groupName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	output := struct {
		Bucket *string
		Policy *iam.Policy
		Group  *iam.Group
	}{
		bucketOutput.Location,
		policyOutput.Policy,
		group.Group,
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal reasponse(%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// rollBack executes functions from a stack of rollback functions
func rollBack(t *[]func() error) {
	if t == nil {
		return
	}

	tasks := *t
	log.Errorf("executing rollback of %d tasks", len(tasks))
	for i := len(tasks) - 1; i >= 0; i-- {
		f := tasks[i]
		if funcerr := f(); funcerr != nil {
			log.Errorf("rollback task error: %s, continuing rollback", funcerr)
		}
	}
}
