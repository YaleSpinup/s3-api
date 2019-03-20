package s3api

import (
	"net/http"
	"os"
	"time"

	"github.com/YaleSpinup/s3-api/common"
	"github.com/YaleSpinup/s3-api/iam"
	"github.com/YaleSpinup/s3-api/s3"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

type server struct {
	s3Services  map[string]s3.S3
	iamServices map[string]iam.IAM
	router      *mux.Router
	version     common.Version
}

// NewServer creates a new server and starts it
func NewServer(config common.Config) error {
	s := server{
		s3Services:  make(map[string]s3.S3),
		iamServices: make(map[string]iam.IAM),
		router:      mux.NewRouter(),
		version:     config.Version,
	}

	// Create a shared S3 session
	for name, c := range config.Accounts {
		log.Debugf("Creating new S3 service for account '%s' with key '%s' in region '%s'", name, c.Akid, c.Region)
		s.s3Services[name] = s3.NewSession(c)
		s.iamServices[name] = iam.NewSession(c)
	}

	publicURLs := map[string]string{
		"/v1/s3/ping":    "public",
		"/v1/s3/version": "public",
		"/v1/s3/metrics": "public",
	}

	// load routes
	s.routes()

	if config.ListenAddress == "" {
		config.ListenAddress = ":8080"
	}
	handler := handlers.RecoveryHandler()(handlers.LoggingHandler(os.Stdout, TokenMiddleware(config.Token, publicURLs, s.router)))
	srv := &http.Server{
		Handler:      handler,
		Addr:         config.ListenAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Infof("Starting listener on %s", config.ListenAddress)
	if err := srv.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

// LogWriter is an http.ResponseWriter
type LogWriter struct {
	http.ResponseWriter
}

// Write log message if http response writer returns an error
func (w LogWriter) Write(p []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(p)
	if err != nil {
		log.Errorf("Write failed: %v", err)
	}
	return
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
