package s3api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func executeRollBack(t *[]func() error) {
	if t != nil {
		tasks := *t
		log.Errorf("executing rollback of %d tasks", len(tasks))
		for i := len(tasks) - 1; i >= 0; i-- {
			f := tasks[i]
			if funcerr := f(); funcerr != nil {
				log.Errorf("rollback task error: %s, continuing rollback", funcerr)
			}
		}
	}
}

// BucketCreateHandler creates a new s3 bucket
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

	_, err = iamService.AttachGroupPolicy(r.Context(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String(groupName),
		PolicyArn: policyOutput.Policy.Arn,
	})
	if err != nil {
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
	w.WriteHeader(http.StatusAccepted)
	w.Write(j)
}

// BucketListHandler gets a list of buckets
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

	log.Infof("listing buckets")
	input := s3.ListBucketsInput{}
	output, err := s3Service.Service.ListBucketsWithContext(r.Context(), &input)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	buckets := []string{}
	for _, b := range output.Buckets {
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
	output, err := s3Service.Service.HeadBucketWithContext(r.Context(), &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				log.Errorf("bucket %s not found.", bucket)
				w.WriteHeader(http.StatusNotFound)
				return
			case "NotFound":
				log.Errorf("bucket %s not found.", bucket)
				w.WriteHeader(http.StatusNotFound)
				return
			case "Forbidden":
				log.Errorf("forbidden to access requested bucket %s", bucket)
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}
		log.Errorf("error checking for bucket %s: %s", bucket, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
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

// BucketDeleteHandler deletes an empty bucket
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

	output, err := s3Service.DeleteEmptyBucket(r.Context(), &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		handleError(w, err)
		return
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

func handleError(w http.ResponseWriter, err error) {
	log.Error(err.Error())
	if aerr, ok := err.(apierror.Error); ok {
		switch aerr.Code {
		case apierror.ErrForbidden:
			w.WriteHeader(http.StatusForbidden)
		case apierror.ErrNotFound:
			w.WriteHeader(http.StatusNotFound)
		case apierror.ErrConflict:
			w.WriteHeader(http.StatusConflict)
		case apierror.ErrBadRequest:
			w.WriteHeader(http.StatusBadRequest)
		case apierror.ErrLimitExceeded:
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(aerr.Message))
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}
