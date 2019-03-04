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

// BucketExists checks if a bucket exists
func (s *S3) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if _, err := s.Service.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "NotFound":
				return false, nil
			case "Forbidden":
				msg := fmt.Sprintf("forbidden to access requested bucket %s: %s", bucketName, aerr.Error())
				return true, apierror.New(apierror.ErrForbidden, msg, err)
			default:
				return false, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}
		return false, apierror.New(apierror.ErrInternalError, "unexpected error checking for bucket", err)
	}

	// looks like the bucket exists and you have access to it
	return true, nil
}

// CreateBucket handles checking if a bucket exists and creating it
func (s *S3) CreateBucket(ctx context.Context, input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	if input == nil || aws.StringValue(input.Bucket) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating bucket: %s", aws.StringValue(input.Bucket))

	// Checks if a bucket exists in the account.  This is necessary since in
	// us-east-1 (only) bucket creation will succeed if the bucket already exists in your
	// account.  In all other regions, the API will return s3.ErrCodeBucketAlreadyOwnedByYou 🤷‍♂️
	if exists, err := s.BucketExists(ctx, aws.StringValue(input.Bucket)); exists {
		return nil, apierror.New(apierror.ErrConflict, "bucket exists", nil)
	} else if err != nil {
		return nil, apierror.New(apierror.ErrInternalError, "internal error", nil)
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

// ListBuckets handles getting a list of buckets in an account
func (s *S3) ListBuckets(ctx context.Context, input *s3.ListBucketsInput) ([]*s3.Bucket, error) {
	log.Info("listing buckets")
	output, err := s.Service.ListBucketsWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return nil, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return nil, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}
	return output.Buckets, err
}

// GetBucketTags handles getting the tags for a bucket
func (s *S3) GetBucketTags(ctx context.Context, bucket string) ([]*s3.Tag, error) {
	if bucket == "" {
		return []*s3.Tag{}, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}
	log.Infof("getting tags for bucket %s", bucket)
	output, err := s.Service.GetBucketTaggingWithContext(ctx, &s3.GetBucketTaggingInput{Bucket: aws.String(bucket)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return []*s3.Tag{}, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return []*s3.Tag{}, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return output.TagSet, err
}

// TagBucket adds tags to a bucket
func (s *S3) TagBucket(ctx context.Context, bucket string, tags []*s3.Tag) error {
	if len(tags) == 0 {
		return nil
	}

	log.Infof("tagging bucket %s with tags %+v", bucket, tags)

	if _, err := s.Service.PutBucketTaggingWithContext(ctx, &s3.PutBucketTaggingInput{
		Bucket:  aws.String(bucket),
		Tagging: &s3.Tagging{TagSet: tags},
	}); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return nil
}

// UpdateWebsiteConfig sets the configuration for an s3 website, defaults index suffix to index.html
func (s *S3) UpdateWebsiteConfig(ctx context.Context, input *s3.PutBucketWebsiteInput) error {
	if input == nil || aws.StringValue(input.Bucket) == "" || input.WebsiteConfiguration == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("updating website configuration for bucket %s", aws.StringValue(input.Bucket))

	// set the default index document to index.html
	if input.WebsiteConfiguration.IndexDocument == nil || aws.StringValue(input.WebsiteConfiguration.IndexDocument.Suffix) == "" {
		log.Debugf("Index document not set for %s, setting to index.html", aws.StringValue(input.Bucket))
		input.WebsiteConfiguration.IndexDocument = &s3.IndexDocument{Suffix: aws.String("index.html")}
	}

	_, err := s.Service.PutBucketWebsiteWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}
	return err
}

// UpdateBucketPolicy sets a bucket access policy
func (s *S3) UpdateBucketPolicy(ctx context.Context, input *s3.PutBucketPolicyInput) error {
	if input == nil || aws.StringValue(input.Bucket) == "" || aws.StringValue(input.Policy) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("applying bucket policy to %s", aws.StringValue(input.Bucket))

	_, err := s.Service.PutBucketPolicyWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}
	return err
}
