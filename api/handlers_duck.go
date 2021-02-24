package api

import (
	"fmt"
	"net/http"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/s3-api/duck"
	"github.com/gorilla/mux"
)

// BucketDuck returns a cyberduck bookmark file for the bucket or website
func (s *server) BucketDuck(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	bucket := vars["bucket"]
	website := vars["website"]

	_, ok := s.s3Services[account]
	if !ok {
		msg := fmt.Sprintf("s3 service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	if bucket == "" {
		bucket = website
	}

	doc, err := duck.DefaultDuck(bucket).Generate()
	if err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "duck! failed to generate", err))
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+bucket+".duck")
	w.WriteHeader(http.StatusOK)
	w.Write(doc)
}
