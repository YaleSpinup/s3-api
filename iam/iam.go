package iam

import (
	"encoding/json"
	"fmt"

	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	log "github.com/sirupsen/logrus"
)

// PolicyStatement is an individual IAM Policy statement
type PolicyStatement struct {
	Effect    string
	Principal string `json:",omitempty"`
	Action    []string
	Resource  []string
}

// PolicyDoc collects the policy statements
type PolicyDoc struct {
	Version   string
	Statement []PolicyStatement
}

// IAM is a wrapper around the aws IAM service with some default config info
type IAM struct {
	Service                              iamiface.IAMAPI
	DefaultS3BucketActions               []string
	DefaultS3ObjectActions               []string
	DefaultCloudfrontDistributionActions []string
}

// NewSession creates a new IAM session
func NewSession(account common.Account) IAM {
	i := IAM{}
	log.Infof("creating new aws session for IAM with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))

	i.Service = iam.New(sess)
	i.DefaultS3BucketActions = account.DefaultS3BucketActions
	i.DefaultS3ObjectActions = account.DefaultS3ObjectActions
	i.DefaultCloudfrontDistributionActions = account.DefaultCloudfrontDistributionActions

	return i
}

// DefaultBucketAdminPolicy generates the default policy statement for s3 buckets
func (i *IAM) DefaultBucketAdminPolicy(bucket *string) ([]byte, error) {
	b := aws.StringValue(bucket)
	log.Debugf("generating default bucket admin policy for %s", b)
	policyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect:   "Allow",
				Action:   i.DefaultS3BucketActions,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", b)},
			},
			{
				Effect:   "Allow",
				Action:   i.DefaultS3ObjectActions,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/*", b)},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate default bucket admin policy for %s: %s", b, err)
		return []byte{}, err
	}
	log.Debugf("creating policy with document %s", string(policyDoc))

	return policyDoc, nil
}

// DefaultWebAdminPolicy generates the default policy statement for website admin
func (i *IAM) DefaultWebAdminPolicy(distributionArn *string) ([]byte, error) {
	d := aws.StringValue(distributionArn)
	log.Debugf("generating default web admin policy for %s", d)
	policyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect:   "Allow",
				Action:   i.DefaultCloudfrontDistributionActions,
				Resource: []string{aws.StringValue(distributionArn)},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate default web admin policy for %s: %s", d, err)
		return []byte{}, err
	}
	log.Debugf("creating policy with document %s", string(policyDoc))

	return policyDoc, nil
}

// DefaultWebsiteAccessPolicy generated the default website access policy statement for s3 websites
//   {
//     "Version":"2012-10-17",
//     "Statement":[{
// 	     "Sid":"PublicReadGetObject",
// 		 "Effect":"Allow",
// 	     "Principal": "*",
// 	     "Action":["s3:GetObject"],
// 	     "Resource":["arn:aws:s3:::example-bucket/*"]
//     }]
//   }
func (i *IAM) DefaultWebsiteAccessPolicy(bucket *string) ([]byte, error) {
	b := aws.StringValue(bucket)
	log.Debugf("generating default bucket website policy for %s", b)
	policyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect:    "Allow",
				Principal: "*",
				Action:    []string{"s3:GetObject"},
				Resource:  []string{fmt.Sprintf("arn:aws:s3:::%s/*", b)},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate default bucket website policy for %s: %s", b, err)
		return []byte{}, err
	}
	log.Debugf("creating policy with document %s", string(policyDoc))

	return policyDoc, nil
}
