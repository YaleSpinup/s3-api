package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/YaleSpinup/s3-api/common"
	"github.com/YaleSpinup/s3-api/iam"
	"github.com/YaleSpinup/s3-api/s3"
	"github.com/YaleSpinup/s3-api/s3api"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Version is the main version number
	Version = s3api.Version

	// VersionPrerelease is a prerelease marker
	VersionPrerelease = s3api.VersionPrerelease

	// buildstamp is the timestamp the binary was built, it should be set at buildtime with ldflags
	buildstamp = s3api.BuildStamp

	// githash is the git sha of the built binary, it should be set at buildtime with ldflags
	githash = s3api.GitHash

	configFileName = flag.String("config", "config/config.json", "Configuration file.")
	version        = flag.Bool("version", false, "Display version information and exit.")
)

// AppConfig holds the configuration information for the app
var AppConfig common.Config

// S3Services is a global map of S3 services
var S3Services = make(map[string]s3.S3)

// IAMServices is a global map of IAM services
var IAMServices = make(map[string]iam.IAM)

func main() {
	flag.Parse()
	if *version {
		vers()
	}

	log.Infof("Starting S3-API version %s%s", Version, VersionPrerelease)

	configFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalln("Unable to open config file", err)
	}

	r := bufio.NewReader(configFile)
	config, err := common.ReadConfig(r)
	if err != nil {
		log.Fatalf("Unable to read configuration from %s.  %+v", *configFileName, err)
	}
	AppConfig = config

	// Set the loglevel, info if it's unset
	switch AppConfig.LogLevel {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	if AppConfig.LogLevel == "debug" {
		log.Debug("Starting profiler on 127.0.0.1:6080")
		go http.ListenAndServe("127.0.0.1:6080", nil)
	}

	log.Debugf("Read config: %+v", AppConfig)

	// Create a shared S3 session
	for name, c := range AppConfig.Accounts {
		log.Debugf("Creating new S3 service for account '%s' with key '%s' in region '%s'", name, c.Akid, c.Region)
		S3Services[name] = s3.NewSession(c)
		IAMServices[name] = iam.NewSession(c)
	}

	publicURLs := map[string]string{
		"/v1/s3/ping":    "public",
		"/v1/s3/version": "public",
		"/v1/s3/metrics": "public",
	}

	router := mux.NewRouter()
	api := router.PathPrefix("/v1/s3").Subrouter()
	api.HandleFunc("/ping", PingHandler)
	api.HandleFunc("/version", VersionHandler)
	api.Handle("/metrics", promhttp.Handler())

	// Buckets handlers
	api.HandleFunc("/{account}/buckets", BucketListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/buckets", BucketCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/buckets/{bucket}", BucketHeadHandler).Methods(http.MethodHead)
	// api.HandleFunc("/{account}/buckets/{bucket}", BucketShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/buckets/{bucket}", BucketDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/buckets/{bucket}/objects", ObjectCountHandler).Methods(http.MethodHead)

	if AppConfig.ListenAddress == "" {
		AppConfig.ListenAddress = ":8080"
	}
	handler := handlers.LoggingHandler(os.Stdout, TokenMiddleware(publicURLs, router))
	srv := &http.Server{
		Handler:      handler,
		Addr:         AppConfig.ListenAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Infof("Starting listener on %s", AppConfig.ListenAddress)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// LogWriter is an http.ResponseWriter
type LogWriter struct {
	http.ResponseWriter
}

// Write log message if http response writer returns and error
func (w LogWriter) Write(p []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(p)
	if err != nil {
		log.Errorf("Write failed: %v", err)
	}
	return
}

func vers() {
	fmt.Printf("S3-API Version: %s%s\n", Version, VersionPrerelease)
	os.Exit(0)
}
