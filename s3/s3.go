package s3

import (
	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	log "github.com/sirupsen/logrus"
)

// S3 is a wrapper around the aws S3 service with some default config info
type S3 struct {
	Service             s3iface.S3API
	LoggingBucket       string
	LoggingBucketPrefix string
}

// NewSession creates a new S3 session
func NewSession(sess *session.Session, account common.Account, accountId string) S3 {
	if sess == nil {
		config := aws.Config{
			Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
			Region:      aws.String(account.Region),
		}

		if account.Endpoint != "" {
			config.Endpoint = aws.String(account.Endpoint)
		}
		log.Infof("creating new aws session for S3 with key id %s in region %s", account.Akid, account.Region)
		sess = session.Must(session.NewSession(&config))
	}

	s := S3{}
	s.Service = s3.New(sess)

	if account.AccessLog != (common.AccessLog{}) {
		s.LoggingBucket = account.AccessLog.GetBucket(accountId)
		s.LoggingBucketPrefix = account.AccessLog.Prefix
	}

	return s
}
