package s3

import (
	"reflect"
	"testing"
	"time"

	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

var testTime = time.Now()

// mockS3Client is a fake S3 client
type mockS3Client struct {
	s3iface.S3API
	t   *testing.T
	err error
}

func newMockS3Client(t *testing.T, err error) s3iface.S3API {
	return &mockS3Client{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(nil, common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "s3.S3" {
		t.Errorf("expected type to be 's3.S3', got %s", to)
	}

	e = NewSession(nil, common.Account{
		AccessLog: &common.AccessLog{
			Bucket: "foologbucket",
			Prefix: "s3",
		},
	})

	if e.LoggingBucket != "foologbucket" {
		t.Errorf("expected logging bucket to be 'foologbucket', got %s", e.LoggingBucket)
	}

	if e.LoggingBucketPrefix != "s3" {
		t.Errorf("expected logging bucket prefix to be 's3', got %s", e.LoggingBucketPrefix)
	}
}
