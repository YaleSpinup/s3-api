package s3api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// BucketCreateHandler creates a new s3 bucket
func (s *server) BucketCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	s3Service, ok := s.s3Services[account]
	if !ok {
		log.Errorf("S3 service not found for account: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	iamService, ok := s.iamServices[account]
	if !ok {
		log.Errorf("IAM service not found for account: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var req s3.CreateBucketInput
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Errorf("cannot decode body into create bucket input: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Checks if a bucket exists in the account heading it.  This is a bit of a hack since in
	// us-east-1 (only) bucket creation will succeed if the bucket already exists in your
	// account.  In all other regions, the API will return s3.ErrCodeBucketAlreadyOwnedByYou 🤷‍♂️
	_, err = s3Service.Service.HeadBucketWithContext(r.Context(), &s3.HeadBucketInput{
		Bucket: req.Bucket,
	})
	if err == nil {
		msg := fmt.Sprintf("%s: Bucket exists and is owned by you.", s3.ErrCodeBucketAlreadyOwnedByYou)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(msg))
		return
	}

	log.Infof("creating bucket: %s", *req.Bucket)
	bucketOutput, err := s3Service.Service.CreateBucketWithContext(r.Context(), &req)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeBucketAlreadyExists:
				msg := fmt.Sprintf("%s: %s", s3.ErrCodeBucketAlreadyExists, aerr.Error())
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(msg))
			case s3.ErrCodeBucketAlreadyOwnedByYou:
				msg := fmt.Sprintf("%s: %s", s3.ErrCodeBucketAlreadyOwnedByYou, aerr.Error())
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(msg))
			default:
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		}
		return
	}

	defaultPolicy, err := iamService.DefaultBucketAdminPolicy(req.Bucket)
	if err != nil {
		// TODO: delete newly created bucket
		log.Errorf("failed creating default IAM policy for bucket %s: %s", *req.Bucket, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	policyName := fmt.Sprintf("%s-BucketAdminPolicy", *req.Bucket)
	log.Infof("creating IAM policy: %s", policyName)

	policyOutput, err := iamService.Service.CreatePolicyWithContext(r.Context(), &iam.CreatePolicyInput{
		Description:    aws.String(fmt.Sprintf("Admin policy for %s bucket", *req.Bucket)),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String(policyName),
	})

	if err != nil {
		// TODO: delete newly created bucket
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeInvalidInputException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeInvalidInputException, aerr.Error())
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(msg))
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeLimitExceededException, aerr.Error())
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(msg))
			case iam.ErrCodeEntityAlreadyExistsException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeEntityAlreadyExistsException, aerr.Error())
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(msg))
			case iam.ErrCodeMalformedPolicyDocumentException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeMalformedPolicyDocumentException, aerr.Error())
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(msg))
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeServiceFailureException, aerr.Error())
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(msg))
			default:
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		}
		return
	}

	groupName := fmt.Sprintf("%s-BucketAdminGroup", *req.Bucket)
	log.Infof("creating IAM group: %s", groupName)

	group, err := iamService.Service.CreateGroupWithContext(r.Context(), &iam.CreateGroupInput{
		GroupName: aws.String(groupName),
	})

	if err != nil {
		// TODO: delete newly created bucket and IAM policy
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeLimitExceededException, aerr.Error())
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(msg))
			case iam.ErrCodeEntityAlreadyExistsException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeEntityAlreadyExistsException, aerr.Error())
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(msg))
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeNoSuchEntityException, aerr.Error())
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(msg))
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeServiceFailureException, aerr.Error())
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(msg))
			default:
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		}
		return
	}

	log.Infof("attaching group %s to policy with arn %s", groupName, *policyOutput.Policy.Arn)
	_, err = iamService.Service.AttachGroupPolicyWithContext(r.Context(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String(groupName),
		PolicyArn: policyOutput.Policy.Arn,
	})
	if err != nil {
		// TODO: delete newly created bucket, IAM policy, and group
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeNoSuchEntityException, aerr.Error())
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(msg))
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeLimitExceededException, aerr.Error())
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(msg))
			case iam.ErrCodeInvalidInputException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeInvalidInputException, aerr.Error())
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(msg))
			case iam.ErrCodePolicyNotAttachableException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodePolicyNotAttachableException, aerr.Error())
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(msg))
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", iam.ErrCodeServiceFailureException, aerr.Error())
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(msg))
			default:
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		}
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

	log.Infof("deleting bucket: %s", bucket)
	output, err := s3Service.Service.DeleteBucketWithContext(r.Context(), &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				log.Errorf("bucket %s not found.", bucket)
				w.WriteHeader(http.StatusNotFound)
				return
			case "BucketNotEmpty":
				log.Errorf("trying to delete bucket %s that is not empty.", bucket)
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte("bucket not empty"))
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
