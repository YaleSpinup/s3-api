package s3

import (
	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

// S3 is a wrapper around the aws S3 service with some default config info
type S3 struct {
	Service *s3.S3
}

// NewSession creates a new ECS session
func NewSession(account common.Account) S3 {
	s := S3{}
	log.Infof("creating new session with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	s.Service = s3.New(sess)
	return s
}

// KeyValuePair maps a key to a value
type KeyValuePair struct {
	Key   string
	Value string
}
