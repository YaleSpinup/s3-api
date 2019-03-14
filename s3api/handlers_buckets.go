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
// 2. tag the bucket with given tags
// 3. generate the default admin bucket policy
// 4. create the admin bucket policy
// 5. create the bucket admin group, '<bucketName>-BktAdmGrp'
// 6. attach the bucket admin policy to the bucket admin group
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

	var req struct {
		Tags        []*s3.Tag
		BucketInput s3.CreateBucketInput
	}
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

	bucketName := aws.StringValue(req.BucketInput.Bucket)
	bucketOutput, err := s3Service.CreateBucket(r.Context(), &req.BucketInput)
	if err != nil {
		handleError(w, err)
		msg := fmt.Sprintf("failed to create bucket: %s", err)
		panic(msg)
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
		handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
		panic(msg)
	}

	// build the default IAM bucket admin policy (from the config and known inputs)
	defaultPolicy, err := iamService.DefaultBucketAdminPolicy(aws.String(bucketName))
	if err != nil {
		msg := fmt.Sprintf("failed creating default IAM policy for bucket %s: %s", bucketName, err.Error())
		handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
		panic(msg)
	}

	policyOutput, err := iamService.CreatePolicy(r.Context(), &iam.CreatePolicyInput{
		Description:    aws.String(fmt.Sprintf("Admin policy for %s bucket", bucketName)),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String(fmt.Sprintf("%s-BktAdmPlc", bucketName)),
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

	groupName := fmt.Sprintf("%s-BktAdmGrp", bucketName)
	group, err := iamService.CreateGroup(r.Context(), &iam.CreateGroupInput{
		GroupName: aws.String(groupName),
	})

	if err != nil {
		handleError(w, err)
		msg := fmt.Sprintf("failed to create group: %s", err.Error())
		panic(msg)
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

// BucketShowHandler returns information about a bucket
func (s *server) BucketShowHandler(w http.ResponseWriter, r *http.Request) {
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

	tags, err := s3Service.GetBucketTags(r.Context(), bucket)
	if err != nil {
		handleError(w, err)
		return
	}

	empty, err := s3Service.BucketEmpty(r.Context(), bucket)
	if err != nil {
		handleError(w, err)
		return
	}

	// setup output struct
	output := struct {
		Tags  []*s3.Tag
		Empty bool
	}{
		Tags:  tags,
		Empty: empty,
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// BucketUpdateHandler handles updating making changes to a bucket.  Currently supports:
// - Updating the bucket's tags
func (s *server) BucketUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	bucket := vars["bucket"]
	s3Service, ok := s.s3Services[account]
	if !ok {
		msg := fmt.Sprintf("s3 service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	var req struct {
		Tags []*s3.Tag
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into update bucket input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	if len(req.Tags) > 0 {
		err = s3Service.TagBucket(r.Context(), bucket, req.Tags)
		if err != nil {
			msg := fmt.Sprintf("failed to tag bucket %s: %s", bucket, err.Error())
			handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}
