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
	e := NewSession(common.Account{})
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
		PolicyStatement{
			Effect:   "Allow",
			Action:   i.DefaultS3BucketActions,
			Resource: []string{fmt.Sprintf("arn:aws:s3:::%s", bucket)},
		},
		PolicyStatement{
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
		PolicyStatement{
			Effect:   "Allow",
			Action:   i.DefaultCloudfrontDistributionActions,
			Resource: []string{distributionARN},
		},
	},
}

var defaultWebsitePolicyDoc = PolicyDoc{
	Version: "2012-10-17",
	Statement: []PolicyStatement{
		PolicyStatement{
			Effect:    "Allow",
			Principal: "*",
			Action:    []string{"s3:GetObject"},
			Resource:  []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
		},
	},
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
