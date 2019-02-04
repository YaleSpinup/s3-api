package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// BucketCreateHandler creates a new s3 bucket
func BucketCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	s3Service, ok := S3Services[account]
	if !ok {
		log.Errorf("account not found: %s", account)
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

	// Checks if a bucket exists in the account by getting the location.  This is
	// a hack since in us-east-1 (only) bucket creation will succeed if the bucket already exists in your
	// account.  In all other regions, the API will return s3.ErrCodeBucketAlreadyOwnedByYou 🤷‍♂️
	_, err = s3Service.Service.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: req.Bucket})
	if err == nil {
		msg := fmt.Sprintf("%s: Bucket exists and is owned by you.", s3.ErrCodeBucketAlreadyOwnedByYou)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(msg))
		return
	}

	bucket, err := s3Service.Service.CreateBucketWithContext(r.Context(), &req)
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

	j, err := json.Marshal(bucket)
	if err != nil {
		log.Errorf("cannot marshal reasponse(%v) into JSON: %s", bucket, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(j)
}

// BucketListHandler gets a list of buckets
func BucketListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	s3Service, ok := S3Services[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

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

// BucketShowHandler gets a bucket and returns the details
func BucketShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	bucket := vars["bucket"]
	s3Service, ok := S3Services[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

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
