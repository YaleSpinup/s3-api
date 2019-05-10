package common

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Config is representation of the configuration data
type Config struct {
	ListenAddress string
	Accounts      map[string]Account
	Token         string
	LogLevel      string
	Version       Version
}

// Account is the configuration for an individual account
type Account struct {
	Region                 string
	Akid                   string
	Secret                 string
	DefaultS3BucketActions []string
	DefaultS3ObjectActions []string
	AccessLog              AccessLog
	Domains                map[string]Domain
}

// AccessLog is the configuration for a bucket's access log
type AccessLog struct {
	Bucket string
	Prefix string
}

// Domain is the domain configuration for an S3 site
type Domain struct {
	CertArn      string
	HostedZoneID string
}

// Version carries around the API version information
type Version struct {
	Version           string
	VersionPrerelease string
	BuildStamp        string
	GitHash           string
}

// ReadConfig decodes the configuration from an io Reader
func ReadConfig(r io.Reader) (Config, error) {
	var c Config
	log.Infoln("Reading configuration")
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return c, errors.Wrap(err, "unable to decode JSON message")
	}
	return c, nil
}
