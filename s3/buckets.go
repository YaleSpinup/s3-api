package s3

import (
	"context"
	"fmt"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

// CreateBucket handles checking if a bucket exists and creating it
func (s *S3) CreateBucket(ctx context.Context, input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	if input == nil || aws.StringValue(input.Bucket) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating bucket: %s", aws.StringValue(input.Bucket))

	// Checks if a bucket exists in the account heading it.  This is a bit of a hack since in
	// us-east-1 (only) bucket creation will succeed if the bucket already exists in your
	// account.  In all other regions, the API will return s3.ErrCodeBucketAlreadyOwnedByYou 🤷‍♂️
	if _, err := s.Service.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: input.Bucket,
	}); err == nil {
		return nil, apierror.New(apierror.ErrConflict, "bucket exists and is owned by you", nil)
	}

	output, err := s.Service.CreateBucketWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeBucketAlreadyExists, s3.ErrCodeBucketAlreadyOwnedByYou:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrConflict, msg, err)
			default:
				return nil, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return nil, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return output, nil
}

// DeleteEmptyBucket handles deleting an empty bucket
func (s *S3) DeleteEmptyBucket(ctx context.Context, input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	if input == nil || aws.StringValue(input.Bucket) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting bucket: %s", aws.StringValue(input.Bucket))

	output, err := s.Service.DeleteBucketWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "NotFound":
				msg := fmt.Sprintf("bucket %s not found: %s", aws.StringValue(input.Bucket), aerr.Error())
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
			case "BucketNotEmpty":
				msg := fmt.Sprintf("trying to delete bucket %s that is not empty: %s", aws.StringValue(input.Bucket), aerr.Error())
				return nil, apierror.New(apierror.ErrConflict, msg, err)
			case "Forbidden":
				msg := fmt.Sprintf("forbidden to access requested bucket %s: %s", aws.StringValue(input.Bucket), aerr.Error())
				return nil, apierror.New(apierror.ErrForbidden, msg, err)
			default:
				return nil, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return nil, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return output, err
}
