package api

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/YaleSpinup/s3-api/cloudfront"
	"github.com/YaleSpinup/s3-api/common"
	"github.com/YaleSpinup/s3-api/iam"
	"github.com/YaleSpinup/s3-api/route53"
	"github.com/YaleSpinup/s3-api/s3"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type server struct {
	s3Services         map[string]s3.S3
	iamServices        map[string]iam.IAM
	cloudFrontServices map[string]cloudfront.CloudFront
	route53Services    map[string]route53.Route53
	router             *mux.Router
	version            common.Version
	context            context.Context
}

// cleaner will do its action once every interval
type cleaner struct {
	account           string
	interval          time.Duration
	s3Service         s3.S3
	iamService        iam.IAM
	cloudFrontService cloudfront.CloudFront
	route53Services   route53.Route53
	context           context.Context
}

// Org will carry throughout the api and get tagged on resources
var Org string

// NewServer creates a new server and starts it
func NewServer(config common.Config) error {
	// setup server context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := server{
		s3Services:         make(map[string]s3.S3),
		iamServices:        make(map[string]iam.IAM),
		cloudFrontServices: make(map[string]cloudfront.CloudFront),
		route53Services:    make(map[string]route53.Route53),
		router:             mux.NewRouter(),
		version:            config.Version,
		context:            ctx,
	}

	if config.Org == "" {
		return errors.New("'org' cannot be empty in the configuration")
	}
	Org = config.Org

	// Create a shared S3 session
	for name, c := range config.Accounts {
		log.Debugf("Creating new S3 service for account '%s' with key '%s' in region '%s' (org: %s)", name, c.Akid, c.Region, Org)
		s.s3Services[name] = s3.NewSession(c)
		s.iamServices[name] = iam.NewSession(c)
		s.cloudFrontServices[name] = cloudfront.NewSession(c)
		s.route53Services[name] = route53.NewSession(c)
		if c.Cleaner != nil {
			log.Infof("starting cleaner for account %s (org: %s)", name, Org)
			interval, err := cleanerInterval(c.Cleaner.Interval, c.Cleaner.MaxSplay)
			if err != nil {
				return err
			}

			acctCleaner := &cleaner{
				account:           name,
				interval:          *interval,
				s3Service:         s.s3Services[name],
				iamService:        s.iamServices[name],
				cloudFrontService: s.cloudFrontServices[name],
				route53Services:   s.route53Services[name],
				context:           ctx,
			}
			log.Debugf("initialized cleaner %+v", acctCleaner)

			acctCleaner.run()
		}
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
	handler := handlers.RecoveryHandler()(handlers.LoggingHandler(os.Stdout, TokenMiddleware([]byte(config.Token), publicURLs, s.router)))
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

type stop struct {
	error
}

// retry is stolen from https://upgear.io/blog/simple-golang-retry-function/
func retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			// Add some randomness to prevent creating a Thundering Herd
			jitter := time.Duration(rand.Int63n(int64(sleep)))
			sleep = sleep + jitter/2

			time.Sleep(sleep)
			return retry(attempts, 2*sleep, f)
		}
		return err
	}

	return nil
}
