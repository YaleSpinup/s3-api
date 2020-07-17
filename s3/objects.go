package s3

import (
	"context"
	"errors"
	"strings"

	"github.com/YaleSpinup/apierror"
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

// GetObjectTagging gets the tagging data from an object is S3
func (s *S3) GetObjectTagging(ctx context.Context, input *s3.GetObjectTaggingInput) ([]*s3.Tag, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	if aws.StringValue(input.Bucket) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("missing bucket name"))
	}

	path := aws.StringValue(input.Key)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	log.Infof("getting object tagging for s3:%s%s", aws.StringValue(input.Bucket), path)

	out, err := s.Service.GetObjectTaggingWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to get tagging for object "+path, err)
	}

	return out.TagSet, nil
}

// DeleteObject deletes an object from S3
func (s *S3) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	if aws.StringValue(input.Bucket) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("missing bucket name"))
	}

	if aws.StringValue(input.Key) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("missing key name"))
	}

	path := aws.StringValue(input.Key)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	log.Infof("deleting object s3:%s%s", aws.StringValue(input.Bucket), path)

	out, err := s.Service.DeleteObjectWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to delete object "+path, err)
	}

	return out, nil
}
