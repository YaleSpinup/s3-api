package common

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Config is representation of the configuration data
type Config struct {
	ListenAddress string
	Account       Account
	AccountsMap   map[string]string
	Token         string
	LogLevel      string
	Version       Version
	Org           string
}

// Account is the configuration for an individual account
type Account struct {
	Endpoint                             string
	Region                               string
	Akid                                 string
	Secret                               string
	ExternalID                           string
	Role                                 string
	DefaultS3BucketActions               []string
	DefaultS3ObjectActions               []string
	DefaultCloudfrontDistributionActions []string
	AccessLog                            AccessLog
	Domains                              map[string]*Domain
	Cleaner                              *Cleaner
}

// AccessLog is the configuration for a bucket's access log
type AccessLog struct {
	Bucket string
	Prefix string
}

// GetBucket gets the bucket name given an account id
func (a *AccessLog) GetBucket(id string) string {
	bucket := strings.Replace(a.Bucket, "{account_id}", id, 1)

	return bucket
}

// Domain is the domain configuration for an S3 site
type Domain struct {
	CertArn      string
	HostedZoneID string
}

// Cleaner is the configuration for the periodic cleaner task
type Cleaner struct {
	Interval string
	MaxSplay string
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
