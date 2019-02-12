package iam

import (
	"encoding/json"
	"fmt"

	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// PolicyStatement is an individual IAM Policy statement
type PolicyStatement struct {
	Effect   string
	Action   []string
	Resource []string
}

// PolicyDoc collects the policy statements
type PolicyDoc struct {
	Version   string
	Statement []PolicyStatement
}

// IAM is a wrapper around the aws IAM service with some default config info
type IAM struct {
	Service                *iam.IAM
	DefaultS3BucketActions []string
	DefaultS3ObjectActions []string
}

// NewSession creates a new IAM session
func NewSession(account common.Account) IAM {
	i := IAM{}
	log.Infof("creating new session with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))

	i.Service = iam.New(sess)
	i.DefaultS3BucketActions = account.DefaultS3BucketActions
	i.DefaultS3ObjectActions = account.DefaultS3ObjectActions

	return i
}

// DefaultBucketAdminPolicy generates the default policy statement for s3 buckets
func (i *IAM) DefaultBucketAdminPolicy(bucket *string) ([]byte, error) {
	b := aws.StringValue(bucket)
	log.Debugf("generating default bucket admin policy for bucket: %s", b)
	policyDoc, err := json.MarshalIndent(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			PolicyStatement{
				Effect:   "Allow",
				Action:   i.DefaultS3BucketActions,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", b)},
			},
			PolicyStatement{
				Effect:   "Allow",
				Action:   i.DefaultS3ObjectActions,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/*", b)},
			},
		},
	}, "", "  ")

	if err != nil {
		log.Errorf("failed to generate default bucket admin policy for bucket: %s: %s", b, err)
		return []byte{}, err
	}
	log.Debugf("creating policy with document %s", string(policyDoc))

	return policyDoc, nil
}
