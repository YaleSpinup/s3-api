package s3

import (
	"context"
	"errors"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

// CreateObject creates an object in S3
func (s *S3) CreateObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	if input.Body == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("missing object body"))
	}

	if aws.StringValue(input.Bucket) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("missing bucket name"))
	}

	log.Infof("creating object in bucket: %s", aws.StringValue(input.Bucket))

	out, err := s.Service.PutObjectWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create object in bucket", err)
	}

	return out, nil
}
