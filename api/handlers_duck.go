package api

import (
	"fmt"
	"net/http"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/s3-api/duck"
	s3api "github.com/YaleSpinup/s3-api/s3"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// BucketDuck returns a cyberduck bookmark file for the bucket or website
func (s *server) BucketDuck(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	bucket := vars["bucket"]
	website := vars["website"]
	path := r.URL.Query().Get("path")

	log.Debugf("bucket duck path: %s", path)
	if path == "" {
		path = "/"
	}

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("s3:ListBucket")
	if err != nil {
		log.Errorf("cannot generate policy: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
	)
	if err != nil {
		log.Errorf("failed to assume role in account: %s", accountId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = s3api.NewSession(session.Session, s.account, s.mapToAccountName(accountId))

	if bucket == "" {
		bucket = website
	}

	doc, err := duck.DefaultDuck(bucket, path).Generate()
	if err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "duck! failed to generate", err))
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+bucket+".duck")
	w.WriteHeader(http.StatusOK)
	w.Write(doc)
}
