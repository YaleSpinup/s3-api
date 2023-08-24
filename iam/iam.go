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

var (
	BucketAdminPolicy = []string{
		// "s3:CreateAccessPoint",
		// "s3:DeleteAccessPoint",
		// "s3:DeleteAccessPointPolicy",
		// "s3:GetAccessPoint",
		// "s3:GetAccessPointPolicy",
		// "s3:GetAccessPointPolicyStatus",
		// "s3:PutAccessPointPolicy",
		"s3:PutBucketPolicy",
		"s3:DeleteBucketPolicy",
		"s3:PutBucketWebsite",
		"s3:DeleteBucketWebsite",
		"s3:ListAllMyBuckets",
		"s3:PutAccelerateConfiguration",
		"s3:PutBucketAcl",
		"s3:PutBucketCORS",
		"s3:PutBucketNotification",
		"s3:PutBucketObjectLockConfiguration",
		"s3:PutBucketRequestPayment",
		"s3:PutBucketVersioning",
		"s3:PutInventoryConfiguration",
		"s3:PutLifecycleConfiguration",
		"s3:PutReplicationConfiguration",
	}

	ObjectWritePolicy = []string{
		"s3:AbortMultipartUpload",
		"s3:DeleteObject",
		"s3:DeleteObjectVersion",
		"s3:PutObject",
		"s3:PutObjectAcl",
		"s3:PutObjectVersionAcl",
		"s3:PutObjectRetention",
		"s3:ReplicateDelete",
		"s3:ReplicateObject",
		"s3:RestoreObject",
		"s3:PutObjectLegalHold",
	}

	ObjectReadPolicy = []string{
		"s3:GetObject",
		"s3:GetObjectAcl",
		"s3:GetObjectLegalHold",
		"s3:GetObjectRetention",
		"s3:GetObjectTagging",
		"s3:GetObjectVersion",
		"s3:GetObjectVersionAcl",
		"s3:GetObjectVersionForReplication",
		"s3:GetObjectVersionTagging",
	}

	BucketReadPolicy = []string{
		"s3:GetAccelerateConfiguration",
		"s3:GetBucketAcl",
		"s3:GetBucketCORS",
		"s3:GetBucketLocation",
		"s3:GetBucketLogging",
		"s3:GetBucketNotification",
		"s3:GetBucketObjectLockConfiguration",
		"s3:GetBucketPolicy",
		"s3:GetBucketPolicyStatus",
		"s3:GetBucketPublicAccessBlock",
		"s3:GetBucketRequestPayment",
		"s3:GetBucketTagging",
		"s3:GetBucketVersioning",
		"s3:GetBucketWebsite",
		"s3:GetEncryptionConfiguration",
		"s3:GetInventoryConfiguration",
		"s3:GetLifecycleConfiguration",
		"s3:GetReplicationConfiguration",
		"s3:GetMetricsConfiguration",
		"s3:GetReplicationConfiguration",
		"s3:ListAccessPoints",
		"s3:ListAllMyBuckets",
		"s3:ListBucket",
		"s3:ListBucketMultipartUploads",
		"s3:ListBucketVersions",
		"s3:ListMultipartUploadParts",
	}
)

// PolicyStatement is an individual IAM Policy statement
type PolicyStatement struct {
	Effect    string
	Principal string `json:",omitempty"`
	Action    []string
	Resource  []string
	Condition map[string]PolicyCondition `json:",omitempty"`
}

type PolicyCondition map[string]string

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
func NewSession(sess *session.Session, account common.Account) IAM {
	if sess == nil {
		config := aws.Config{
			Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
			Region:      aws.String(account.Region),
		}

		if account.Endpoint != "" {
			config.Endpoint = aws.String(account.Endpoint)
		}
		log.Infof("creating new aws session for IAM with key id %s in region %s", account.Akid, account.Region)
		sess = session.Must(session.NewSession(&config))
	}

	i := IAM{}
	i.Service = iam.New(sess)
	i.DefaultS3BucketActions = account.DefaultS3BucketActions
	i.DefaultS3ObjectActions = account.DefaultS3ObjectActions
	i.DefaultCloudfrontDistributionActions = account.DefaultCloudfrontDistributionActions

	return i
}

// ReadOnlyBucketPolicy generates the read-only bucket policy
func (i *IAM) ReadOnlyBucketPolicy(bucket string) ([]byte, error) {

	log.Infof("generating read-only bucket policy document for %s", bucket)

	policyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect:   "Allow",
				Action:   BucketReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", bucket)},
			},
			{
				Effect:   "Allow",
				Action:   ObjectReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate read-only bucket policy for %s: %s", bucket, err)
		return []byte{}, err
	}

	log.Debugf("generated policy document %s", string(policyDoc))

	return policyDoc, nil
}

// ReadOnlyBucketPolicyWithPath generates the read-only bucket policy
func (i *IAM) ReadOnlyBucketPolicyWithPath(bucket string, path string) ([]byte, error) {
	log.Infof("generating read-only bucket policy document for %s", bucket)

	path = RemoveCappingSlashes(path)

	policyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect:   "Allow",
				Action:   BucketReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", bucket)},
				Condition: map[string]PolicyCondition{
					"StringLike": {
						"s3:prefix": fmt.Sprintf("%s/*", path),
					},
				},
			},
			{
				Effect:   "Allow",
				Action:   ObjectReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/%s/*", bucket, path)},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate read-only bucket policy for %s: %s", bucket, err)
		return []byte{}, err
	}

	log.Debugf("generated policy document %s", string(policyDoc))

	return policyDoc, nil
}

// ReadWriteBucketPolicy generates the read-write bucket policy
func (i *IAM) ReadWriteBucketPolicy(bucket string) ([]byte, error) {

	log.Infof("generating read-write bucket policy document for %s", bucket)

	policyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect:   "Allow",
				Action:   BucketReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", bucket)},
			},
			{
				Effect:   "Allow",
				Action:   ObjectReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
			},
			{
				Effect:   "Allow",
				Action:   ObjectWritePolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate read-write bucket policy for %s: %s", bucket, err)
		return []byte{}, err
	}

	log.Debugf("generated policy document %s", string(policyDoc))

	return policyDoc, nil
}

// ReadWriteBucketPolicyWithPath generates the read-write bucket policy
func (i *IAM) ReadWriteBucketPolicyWithPath(bucket string, path string) ([]byte, error) {
	log.Infof("generating read-write bucket policy document for %s", bucket)

	path = RemoveCappingSlashes(path)

	policyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect:   "Allow",
				Action:   BucketReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", bucket)},
				Condition: map[string]PolicyCondition{
					"StringLike": {
						"s3:prefix": fmt.Sprintf("%s/*", path),
					},
				},
			},
			{
				Effect:   "Allow",
				Action:   ObjectReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/%s/*", bucket, path)},
			},
			{
				Effect:   "Allow",
				Action:   ObjectWritePolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/%s/*", bucket, path)},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate read-write bucket policy for %s: %s", bucket, err)
		return []byte{}, err
	}

	log.Debugf("generated policy document %s", string(policyDoc))

	return policyDoc, nil
}

// AdminBucketPolicy generates the administrative bucket policy
func (i *IAM) AdminBucketPolicy(bucket string) ([]byte, error) {

	log.Infof("generating administrative bucket policy document for %s", bucket)

	policyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect:   "Allow",
				Action:   BucketAdminPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", bucket)},
			},
			{
				Effect:   "Allow",
				Action:   BucketReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", bucket)},
			},
			{
				Effect:   "Allow",
				Action:   ObjectReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
			},
			{
				Effect:   "Allow",
				Action:   ObjectWritePolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate read-write bucket policy for %s: %s", bucket, err)
		return []byte{}, err
	}

	log.Debugf("generated policy document %s", string(policyDoc))

	return policyDoc, nil
}

// AdminBucketPolicyWithPath generates the administrative bucket policy
func (i *IAM) AdminBucketPolicyWithPath(bucket string, path string) ([]byte, error) {
	log.Infof("generating administrative bucket policy document for %s", bucket)

	path = RemoveCappingSlashes(path)

	policyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect:   "Allow",
				Action:   BucketAdminPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", bucket)},
				Condition: map[string]PolicyCondition{
					"StringLike": {
						"s3:prefix": fmt.Sprintf("%s/*", path),
					},
				},
			},
			{
				Effect:   "Allow",
				Action:   BucketReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", bucket)},
				Condition: map[string]PolicyCondition{
					"StringLike": {
						"s3:prefix": fmt.Sprintf("%s/*", path),
					},
				},
			},
			{
				Effect:   "Allow",
				Action:   ObjectReadPolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/%s/*", bucket, path)},
			},
			{
				Effect:   "Allow",
				Action:   ObjectWritePolicy,
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/%s/*", bucket, path)},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate read-write bucket policy for %s: %s", bucket, err)
		return []byte{}, err
	}

	log.Debugf("generated policy document %s", string(policyDoc))

	return policyDoc, nil
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
//
//	  {
//	    "Version":"2012-10-17",
//	    "Statement":[{
//		     "Sid":"PublicReadGetObject",
//			 "Effect":"Allow",
//		     "Principal": "*",
//		     "Action":["s3:GetObject"],
//		     "Resource":["arn:aws:s3:::example-bucket/*"]
//	    }]
//	  }
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
