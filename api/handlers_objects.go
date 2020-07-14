package api

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ObjectCountHandler returns the count of objects as a header
func (s *server) ObjectCountHandler(w http.ResponseWriter, r *http.Request) {
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
	_, err := s3Service.Service.HeadBucketWithContext(r.Context(), &s3.HeadBucketInput{
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
