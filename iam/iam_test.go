package iam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

var testTime = time.Now()

// mockIAMClient is a fake IAM client
type mockIAMClient struct {
	iamiface.IAMAPI
	t   *testing.T
	err error
}

func newMockIAMClient(t *testing.T, err error) iamiface.IAMAPI {
	return &mockIAMClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(nil, common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "iam.IAM" {
		t.Errorf("expected type to be 'iam.IAM', got %s", to)
	}
}

var bucket = "vehicles"

var i = &IAM{
	DefaultS3BucketActions:               []string{"f150", "focus", "edge", "ranger", "fusion", "mustang", "gt"},
	DefaultS3ObjectActions:               []string{"silverado", "cruze", "traverse", "colorodo", "malibu", "camaro", "corvette"},
	DefaultCloudfrontDistributionActions: []string{"sl1", "sl2"},
}

var defaultPolicyDoc = PolicyDoc{
	Version: "2012-10-17",
	Statement: []PolicyStatement{
		{
			Effect:   "Allow",
			Action:   i.DefaultS3BucketActions,
			Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", bucket)},
		},
		{
			Effect:   "Allow",
			Action:   i.DefaultS3ObjectActions,
			Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
		},
	},
}

var distributionARN = "arn:aws:cloudfront::012345678910:distribution/ET123456ABCDE"
var defaultWebPolicyDoc = PolicyDoc{
	Version: "2012-10-17",
	Statement: []PolicyStatement{
		{
			Effect:   "Allow",
			Action:   i.DefaultCloudfrontDistributionActions,
			Resource: []string{distributionARN},
		},
	},
}

var defaultWebsitePolicyDoc = PolicyDoc{
	Version: "2012-10-17",
	Statement: []PolicyStatement{
		{
			Effect:    "Allow",
			Principal: "*",
			Action:    []string{"s3:GetObject"},
			Resource:  []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
		},
	},
}

func TestReadOnlyBucketPolicyWithoutPath(t *testing.T) {
	expected := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:GetAccelerateConfiguration","s3:GetBucketAcl","s3:GetBucketCORS","s3:GetBucketLocation","s3:GetBucketLogging","s3:GetBucketNotification","s3:GetBucketObjectLockConfiguration","s3:GetBucketPolicy","s3:GetBucketPolicyStatus","s3:GetBucketPublicAccessBlock","s3:GetBucketRequestPayment","s3:GetBucketTagging","s3:GetBucketVersioning","s3:GetBucketWebsite","s3:GetEncryptionConfiguration","s3:GetInventoryConfiguration","s3:GetLifecycleConfiguration","s3:GetReplicationConfiguration","s3:GetMetricsConfiguration","s3:GetReplicationConfiguration","s3:ListAccessPoints","s3:ListAllMyBuckets","s3:ListBucket","s3:ListBucketMultipartUploads","s3:ListBucketVersions","s3:ListMultipartUploadParts"],"Resource":["arn:aws:s3:::vehicles"],"Condition":null},{"Effect":"Allow","Action":["s3:GetObject","s3:GetObjectAcl","s3:GetObjectLegalHold","s3:GetObjectRetention","s3:GetObjectTagging","s3:GetObjectVersion","s3:GetObjectVersionAcl","s3:GetObjectVersionForReplication","s3:GetObjectVersionTagging"],"Resource":["arn:aws:s3:::vehicles/*"],"Condition":null}]}`

	policyBytes, err := i.ReadOnlyBucketPolicy(bucket, "")
	if err != nil {
		t.Errorf("expected ReadOnlyBucketPolicy to return nil error, got %s", err)
	}

	if !bytes.Equal(policyBytes, []byte(expected)) {
		t.Errorf("expected: %s\ngot: %s", expected, policyBytes)
	}
}

func TestReadOnlyBucketPolicyWithPath(t *testing.T) {
	expected := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:GetAccelerateConfiguration","s3:GetBucketAcl","s3:GetBucketCORS","s3:GetBucketLocation","s3:GetBucketLogging","s3:GetBucketNotification","s3:GetBucketObjectLockConfiguration","s3:GetBucketPolicy","s3:GetBucketPolicyStatus","s3:GetBucketPublicAccessBlock","s3:GetBucketRequestPayment","s3:GetBucketTagging","s3:GetBucketVersioning","s3:GetBucketWebsite","s3:GetEncryptionConfiguration","s3:GetInventoryConfiguration","s3:GetLifecycleConfiguration","s3:GetReplicationConfiguration","s3:GetMetricsConfiguration","s3:GetReplicationConfiguration","s3:ListAccessPoints","s3:ListAllMyBuckets","s3:ListBucket","s3:ListBucketMultipartUploads","s3:ListBucketVersions","s3:ListMultipartUploadParts"],"Resource":["arn:aws:s3:::vehicles"],"Condition":null},{"Effect":"Allow","Action":["s3:GetObject","s3:GetObjectAcl","s3:GetObjectLegalHold","s3:GetObjectRetention","s3:GetObjectTagging","s3:GetObjectVersion","s3:GetObjectVersionAcl","s3:GetObjectVersionForReplication","s3:GetObjectVersionTagging"],"Resource":["arn:aws:s3:::vehicles/testpath/*"],"Condition":null}]}`

	policyBytes, err := i.ReadOnlyBucketPolicy(bucket, "testpath")
	if err != nil {
		t.Errorf("expected ReadOnlyBucketPolicy to return nil error, got %s", err)
	}

	if !bytes.Equal(policyBytes, []byte(expected)) {
		t.Errorf("expected: %s\ngot: %s", expected, policyBytes)
	}
}

func TestReadWriteBucketPolicy(t *testing.T) {
	expected := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:GetAccelerateConfiguration","s3:GetBucketAcl","s3:GetBucketCORS","s3:GetBucketLocation","s3:GetBucketLogging","s3:GetBucketNotification","s3:GetBucketObjectLockConfiguration","s3:GetBucketPolicy","s3:GetBucketPolicyStatus","s3:GetBucketPublicAccessBlock","s3:GetBucketRequestPayment","s3:GetBucketTagging","s3:GetBucketVersioning","s3:GetBucketWebsite","s3:GetEncryptionConfiguration","s3:GetInventoryConfiguration","s3:GetLifecycleConfiguration","s3:GetReplicationConfiguration","s3:GetMetricsConfiguration","s3:GetReplicationConfiguration","s3:ListAccessPoints","s3:ListAllMyBuckets","s3:ListBucket","s3:ListBucketMultipartUploads","s3:ListBucketVersions","s3:ListMultipartUploadParts"],"Resource":["arn:aws:s3:::vehicles"],"Condition":null},{"Effect":"Allow","Action":["s3:GetObject","s3:GetObjectAcl","s3:GetObjectLegalHold","s3:GetObjectRetention","s3:GetObjectTagging","s3:GetObjectVersion","s3:GetObjectVersionAcl","s3:GetObjectVersionForReplication","s3:GetObjectVersionTagging"],"Resource":["arn:aws:s3:::vehicles/*"],"Condition":null},{"Effect":"Allow","Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:DeleteObjectVersion","s3:PutObject","s3:PutObjectAcl","s3:PutObjectVersionAcl","s3:PutObjectRetention","s3:ReplicateDelete","s3:ReplicateObject","s3:RestoreObject","s3:PutObjectLegalHold"],"Resource":["arn:aws:s3:::vehicles/*"],"Condition":null}]}`

	policyBytes, err := i.ReadWriteBucketPolicy(bucket, "")
	if err != nil {
		t.Errorf("expected ReadWriteBucketPolicy to return nil error, got %s", err)
	}

	if !bytes.Equal(policyBytes, []byte(expected)) {
		t.Errorf("expected: %s\ngot: %s", expected, policyBytes)
	}
}

func TestAdminBucketPolicy(t *testing.T) {
	expected := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:PutBucketPolicy","s3:DeleteBucketPolicy","s3:PutBucketWebsite","s3:DeleteBucketWebsite","s3:ListAllMyBuckets","s3:PutAccelerateConfiguration","s3:PutBucketAcl","s3:PutBucketCORS","s3:PutBucketNotification","s3:PutBucketObjectLockConfiguration","s3:PutBucketRequestPayment","s3:PutBucketVersioning","s3:PutInventoryConfiguration","s3:PutLifecycleConfiguration","s3:PutReplicationConfiguration"],"Resource":["arn:aws:s3:::vehicles"],"Condition":null},{"Effect":"Allow","Action":["s3:GetAccelerateConfiguration","s3:GetBucketAcl","s3:GetBucketCORS","s3:GetBucketLocation","s3:GetBucketLogging","s3:GetBucketNotification","s3:GetBucketObjectLockConfiguration","s3:GetBucketPolicy","s3:GetBucketPolicyStatus","s3:GetBucketPublicAccessBlock","s3:GetBucketRequestPayment","s3:GetBucketTagging","s3:GetBucketVersioning","s3:GetBucketWebsite","s3:GetEncryptionConfiguration","s3:GetInventoryConfiguration","s3:GetLifecycleConfiguration","s3:GetReplicationConfiguration","s3:GetMetricsConfiguration","s3:GetReplicationConfiguration","s3:ListAccessPoints","s3:ListAllMyBuckets","s3:ListBucket","s3:ListBucketMultipartUploads","s3:ListBucketVersions","s3:ListMultipartUploadParts"],"Resource":["arn:aws:s3:::vehicles"],"Condition":null},{"Effect":"Allow","Action":["s3:GetObject","s3:GetObjectAcl","s3:GetObjectLegalHold","s3:GetObjectRetention","s3:GetObjectTagging","s3:GetObjectVersion","s3:GetObjectVersionAcl","s3:GetObjectVersionForReplication","s3:GetObjectVersionTagging"],"Resource":["arn:aws:s3:::vehicles/*"],"Condition":null},{"Effect":"Allow","Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:DeleteObjectVersion","s3:PutObject","s3:PutObjectAcl","s3:PutObjectVersionAcl","s3:PutObjectRetention","s3:ReplicateDelete","s3:ReplicateObject","s3:RestoreObject","s3:PutObjectLegalHold"],"Resource":["arn:aws:s3:::vehicles/*"],"Condition":null}]}`
	policyBytes, err := i.AdminBucketPolicy(bucket, "")
	if err != nil {
		t.Errorf("expected AdminBucketPolicy to return nil error, got %s", err)
	}

	if !bytes.Equal(policyBytes, []byte(expected)) {
		t.Errorf("expected: %s\ngot: %s", expected, policyBytes)
	}
}

func TestDefaultBucketAdminPolicy(t *testing.T) {
	p, err := json.Marshal(defaultPolicyDoc)
	if err != nil {
		t.Errorf("expected to marshall defaultPolicyDoc with nil error, got %s", err)
	}

	policyBytes, err := i.DefaultBucketAdminPolicy(&bucket)
	if err != nil {
		t.Errorf("expected DefaultBucketAdminPolicy to return nil error, got %s", err)
	}

	if !bytes.Equal(policyBytes, p) {
		t.Errorf("expected: %s\ngot: %s", defaultPolicyDoc, policyBytes)
	}
}

func TestDefaultWebAdminPolicy(t *testing.T) {
	p, err := json.Marshal(defaultWebPolicyDoc)
	if err != nil {
		t.Errorf("expected to marshall defaultWebPolicyDoc with nil error, got %s", err)
	}

	policyBytes, err := i.DefaultWebAdminPolicy(&distributionARN)
	if err != nil {
		t.Errorf("expected DefaultWebAdminPolicy to return nil error, got %s", err)
	}

	if !bytes.Equal(policyBytes, p) {
		t.Errorf("expected: %+v\ngot: %s", defaultWebPolicyDoc, policyBytes)
	}
}

func TestDefaultWebsiteAccessPolicy(t *testing.T) {
	p, err := json.Marshal(defaultWebsitePolicyDoc)
	if err != nil {
		t.Errorf("expected to marshall defaultWebsitePolicyDoc with nil error, got %s", err)
	}

	policyBytes, err := i.DefaultWebsiteAccessPolicy(&bucket)
	if err != nil {
		t.Errorf("expected DefaultWebsiteAccessPolicy to return nil error, got %s", err)
	}

	if !bytes.Equal(policyBytes, p) {
		t.Errorf("expected: %+v\ngot: %s", defaultWebsitePolicyDoc, policyBytes)
	}
}
