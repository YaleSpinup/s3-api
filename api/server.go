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
	"github.com/YaleSpinup/s3-api/session"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"

	log "github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type server struct {
	account            common.Account
	accountsMap        map[string]string
	s3Services         map[string]s3.S3
	iamServices        map[string]iam.IAM
	cloudFrontServices map[string]cloudfront.CloudFront
	route53Services    map[string]route53.Route53
	router             *mux.Router
	version            common.Version
	context            context.Context
	session            *session.Session
	sessionCache       *cache.Cache
	org                string
}

// if we have an entry for the account name, return the associated account number
func (s *server) mapAccountNumber(name string) string {
	if a, ok := s.accountsMap[name]; ok {
		return a
	}
	return name
}

// if we have an account id that matches an id in the accounts map we need that account name for the logging bucket
func (s *server) mapToAccountName(id string) string {
	for k, v := range s.accountsMap {
		if id == v {
			return k
		}
	}

	return id
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
	if config.Org == "" {
		return errors.New("'org' cannot be empty in the configuration")
	}

	// setup server context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess := session.New(
		session.WithCredentials(config.Account.Akid, config.Account.Secret, ""),
		session.WithRegion(config.Account.Region),
		session.WithExternalID(config.Account.ExternalId),
		session.WithExternalRoleName(config.Account.Role),
	)
	s := server{
		account:            config.Account,
		accountsMap:        config.AccountsMap,
		s3Services:         make(map[string]s3.S3),
		iamServices:        make(map[string]iam.IAM),
		cloudFrontServices: make(map[string]cloudfront.CloudFront),
		route53Services:    make(map[string]route53.Route53),
		router:             mux.NewRouter(),
		version:            config.Version,
		context:            ctx,
		session:            &sess,
		org:                config.Org,
		sessionCache:       cache.New(600*time.Second, 900*time.Second),
	}
	Org = config.Org

	// Create a shared S3 session
	for name, accountId := range config.AccountsMap {
		log.Debugf("Creating new S3 service for account '%s' with key '%s' in region '%s' (org: %s)", name, config.Account.Akid, config.Account.Region, Org)

		s.s3Services[name] = s3.NewSession(nil, config.Account, name)
		s.iamServices[name] = iam.NewSession(nil, config.Account)
		s.cloudFrontServices[name] = cloudfront.NewSession(nil, config.Account, accountId)
		s.route53Services[name] = route53.NewSession(nil, config.Account)

		if config.Account.Cleaner != nil {
			log.Infof("starting cleaner for account %s (org: %s)", name, Org)

			interval, err := cleanerInterval(config.Account.Cleaner.Interval, config.Account.Cleaner.MaxSplay)

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

type rollbackFunc func(ctx context.Context) error

// rollBack executes functions from a stack of rollback functions
func rollBack(t *[]rollbackFunc) {
	if t == nil {
		return
	}

	timeout, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	done := make(chan string, 1)
	go func() {
		tasks := *t
		log.Errorf("executing rollback of %d tasks", len(tasks))
		for i := len(tasks) - 1; i >= 0; i-- {
			f := tasks[i]
			if funcerr := f(timeout); funcerr != nil {
				log.Errorf("rollback task error: %s, continuing rollback", funcerr)
			}
			log.Infof("executed rollback task %d of %d", len(tasks)-i, len(tasks))
		}
		done <- "success"
	}()

	// wait for a done context
	select {
	case <-timeout.Done():
		log.Error("timeout waiting for successful rollback")
	case <-done:
		log.Info("successfully rolled back")
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
