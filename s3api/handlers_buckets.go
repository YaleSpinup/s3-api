package s3api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// BucketCreateHandler orchestrates the creation of a new s3 bucket with rollback in the event of
// failure.  The operations are
// 1. create the bucket with the given name
// 2. generate the default admin bucket policy
// 3. create the admin bucket policy
// 4. create the bucket admin group, '<bucketName>-BktAdmGrp'
// 5. attach the bucket admin policy to the bucket admin group
// Note: this does _not_ create any users for managing the bucket
func (s *server) BucketCreateHandler(w http.ResponseWriter, r *http.Request) {
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

	var req s3.CreateBucketInput
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create bucket input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// setup rollback function list and defer recovery and execution
	var rollBackTasks []func() error
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("recovering from panic: %s", err)
			executeRollBack(&rollBackTasks)
		}
	}()

	bucketOutput, err := s3Service.CreateBucket(r.Context(), &req)
	if err != nil {
		handleError(w, err)
		msg := fmt.Sprintf("failed to create bucket: %s", err)
		panic(msg)
	}

	// append bucket delete to rollback tasks
	rbfunc := func() error {
		return func() error {
			if _, err := s3Service.DeleteEmptyBucket(r.Context(), &s3.DeleteBucketInput{Bucket: req.Bucket}); err != nil {
				return err
			}
			return nil
		}()
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	// build the default IAM bucket admin policy (from the config and known inputs)
	defaultPolicy, err := iamService.DefaultBucketAdminPolicy(req.Bucket)
	if err != nil {
		msg := fmt.Sprintf("failed creating default IAM policy for bucket %s: %s", *req.Bucket, err.Error())
		handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
		panic(msg)
	}

	policyOutput, err := iamService.CreatePolicy(r.Context(), &iam.CreatePolicyInput{
		Description:    aws.String(fmt.Sprintf("Admin policy for %s bucket", *req.Bucket)),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String(fmt.Sprintf("%s-BktAdmPlc", *req.Bucket)),
	})

	if err != nil {
		handleError(w, err)
		msg := fmt.Sprintf("failed to create policy: %s", err.Error())
		panic(msg)
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

	groupName := fmt.Sprintf("%s-BktAdmGrp", *req.Bucket)
	group, err := iamService.CreateGroup(r.Context(), &iam.CreateGroupInput{
		GroupName: aws.String(groupName),
	})

	if err != nil {
		handleError(w, err)
		msg := fmt.Sprintf("failed to create group: %s", err.Error())
		panic(msg)
	}

	// append policy delete to rollback tasks
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
		handleError(w, err)
		msg := fmt.Sprintf("failed to create group: %s", err.Error())
		panic(msg)
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

// BucketListHandler gets a list of all buckets in the account
func (s *server) BucketListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	s3Service, ok := s.s3Services[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	output, err := s3Service.ListBuckets(r.Context(), &s3.ListBucketsInput{})
	if err != nil {
		handleError(w, err)
		return
	}

	buckets := []string{}
	for _, b := range output {
		buckets = append(buckets, aws.StringValue(b.Name))
	}

	j, err := json.Marshal(buckets)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", buckets, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// BucketHeadHandler checks if a bucket exists
func (s *server) BucketHeadHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	bucket := vars["bucket"]
	s3Service, ok := s.s3Services[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Infof("checking if bucket exists: %s", bucket)
	exists, err := s3Service.BucketExists(r.Context(), bucket)
	if err != nil {
		handleError(w, err)
		return
	}

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte{})
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}

// BucketDeleteHandler deletes an empty bucket and all of it's dependencies.  The operations are
// 1. the bucket is deleted, this will fail if the bucket is not empty
// 2. a list of policies attached to the bucket admin group (<bucketName>-BktAdmGrp) is gathered
// 3. each of those policies is detached from the group and if it starts with '<bucketName>-', it is deleted
// 4. the bucket admin group is deleted
func (s *server) BucketDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	bucket := vars["bucket"]

	s3Service, ok := s.s3Services[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	iamService, ok := s.iamServices[account]
	if !ok {
		msg := fmt.Sprintf("IAM service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	_, err := s3Service.DeleteEmptyBucket(r.Context(), &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	groupName := fmt.Sprintf("%s-BktAdmGrp", bucket)
	policies, err := iamService.ListGroupPolicies(r.Context(), &iam.ListAttachedGroupPoliciesInput{GroupName: aws.String(groupName)})
	if err != nil {
		j, _ := json.Marshal("failed to list group policies: " + err.Error())
		w.Write(j)
		return
	}

	for _, p := range policies {
		if _, err := iamService.DetachGroupPolicy(r.Context(), &iam.DetachGroupPolicyInput{
			GroupName: aws.String(groupName),
			PolicyArn: p.PolicyArn,
		}); err != nil {
			j, _ := json.Marshal("failed to detatch group policy: " + err.Error())
			w.Write(j)
			return
		}

		if strings.HasPrefix(aws.StringValue(p.PolicyName), bucket+"-") {
			if _, err := iamService.DeletePolicy(r.Context(), &iam.DeletePolicyInput{PolicyArn: p.PolicyArn}); err != nil {
				j, _ := json.Marshal("failed to delete group policy: " + err.Error())
				w.Write(j)
				return
			}
			break
		}
	}

	if _, err := iamService.DeleteGroup(r.Context(), &iam.DeleteGroupInput{GroupName: aws.String(groupName)}); err != nil {
		j, _ := json.Marshal("failed to delete group: " + err.Error())
		w.Write(j)
		return
	}

	w.Write([]byte{})
}
