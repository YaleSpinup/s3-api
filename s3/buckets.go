package s3

import (
	"context"
	"fmt"
	"strings"

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
func (s *S3) DeleteEmptyBucket(ctx context.Context, input *s3.DeleteBucketInput) error {
	if input == nil || aws.StringValue(input.Bucket) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting bucket: %s", aws.StringValue(input.Bucket))

	_, err := s.Service.DeleteBucketWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "NotFound":
				msg := fmt.Sprintf("bucket %s not found: %s", aws.StringValue(input.Bucket), aerr.Error())
				return apierror.New(apierror.ErrNotFound, msg, err)
			case "BucketNotEmpty":
				msg := fmt.Sprintf("trying to delete bucket %s that is not empty: %s", aws.StringValue(input.Bucket), aerr.Error())
				return apierror.New(apierror.ErrConflict, msg, err)
			case "Forbidden":
				msg := fmt.Sprintf("forbidden to access requested bucket %s: %s", aws.StringValue(input.Bucket), aerr.Error())
				return apierror.New(apierror.ErrForbidden, msg, err)
			default:
				return apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return err
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
			case s3.ErrCodeNoSuchBucket, "NotFound":
				msg := fmt.Sprintf("bucket %s not found: %s", bucket, aerr.Error())
				return []*s3.Tag{}, apierror.New(apierror.ErrNotFound, msg, err)
			case "NoSuchTagSet":
				return []*s3.Tag{}, nil
			default:
				return []*s3.Tag{}, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return []*s3.Tag{}, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return output.TagSet, nil
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
			case s3.ErrCodeNoSuchBucket, "NotFound":
				msg := fmt.Sprintf("bucket %s not found: %s", bucket, aerr.Error())
				return apierror.New(apierror.ErrNotFound, msg, err)
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
			case s3.ErrCodeNoSuchBucket, "NotFound":
				msg := fmt.Sprintf("bucket %s not found: %s", aws.StringValue(input.Bucket), aerr.Error())
				return apierror.New(apierror.ErrNotFound, msg, err)
			default:
				return apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}
	return nil
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
			case s3.ErrCodeNoSuchBucket, "NotFound":
				msg := fmt.Sprintf("bucket %s not found: %s", aws.StringValue(input.Bucket), aerr.Error())
				return apierror.New(apierror.ErrNotFound, msg, err)
			default:
				return apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}
	return nil
}

// UpdateBucketEncryption sets the bucket encryption
func (s *S3) UpdateBucketEncryption(ctx context.Context, input *s3.PutBucketEncryptionInput) error {
	if input == nil || aws.StringValue(input.Bucket) == "" || input.ServerSideEncryptionConfiguration == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("applying bucket encryption to %s", aws.StringValue(input.Bucket))

	_, err := s.Service.PutBucketEncryptionWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "NotFound":
				msg := fmt.Sprintf("bucket %s not found: %s", aws.StringValue(input.Bucket), aerr.Error())
				return apierror.New(apierror.ErrNotFound, msg, err)
			default:
				return apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}
	return nil
}

// UpdateBucketLogging configures the bucket logging
func (s *S3) UpdateBucketLogging(ctx context.Context, bucket, logBucket, logPrefix string) error {
	if bucket == "" || logBucket == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("applying bucket logging configuration to %s", bucket)

	prefix := bucket
	if logPrefix != "" {
		if strings.HasSuffix(logPrefix, "/") {
			prefix = logPrefix + prefix
		} else {
			prefix = logPrefix + "/" + prefix
		}
	}
	prefix = prefix + "/"

	if _, err := s.Service.PutBucketLoggingWithContext(ctx, &s3.PutBucketLoggingInput{
		Bucket: aws.String(bucket),
		BucketLoggingStatus: &s3.BucketLoggingStatus{
			LoggingEnabled: &s3.LoggingEnabled{
				TargetBucket: aws.String(logBucket),
				TargetPrefix: aws.String(prefix),
			},
		},
	}); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "NotFound":
				msg := fmt.Sprintf("bucket %s not found: %s", bucket, aerr.Error())
				return apierror.New(apierror.ErrNotFound, msg, err)
			default:
				return apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}
	return nil
}

// GetBucketLogging gets a buckets logging configuration
func (s *S3) GetBucketLogging(ctx context.Context, bucket string) (*s3.LoggingEnabled, error) {
	if bucket == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting the logging configuration for bucket %s", bucket)

	out, err := s.Service.GetBucketLoggingWithContext(ctx, &s3.GetBucketLoggingInput{Bucket: aws.String(bucket)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "NotFound":
				msg := fmt.Sprintf("bucket %s not found: %s", bucket, aerr.Error())
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
			default:
				return nil, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return nil, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return out.LoggingEnabled, nil
}

// BucketEmpty lists the objects in a bucket with a max of 1, if there are any objects returned, we return false
func (s *S3) BucketEmpty(ctx context.Context, bucket string) (bool, error) {
	if bucket == "" {
		return false, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("checking if bucket %s is empty", bucket)

	out, err := s.Service.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int64(1),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "NotFound":
				msg := fmt.Sprintf("bucket %s not found: %s", bucket, aerr.Error())
				return false, apierror.New(apierror.ErrNotFound, msg, err)
			default:
				return false, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return false, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	if aws.Int64Value(out.KeyCount) > 0 {
		return false, nil
	}

	return true, nil
}
