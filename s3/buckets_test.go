package s3

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
)

func (m *mockS3Client) HeadBucketWithContext(ctx context.Context, input *s3.HeadBucketInput, opts ...request.Option) (*s3.HeadBucketOutput, error) {
	return nil, awserr.New(s3.ErrCodeNoSuchBucket, "not found", nil)
}

func (m *mockS3Client) CreateBucketWithContext(ctx context.Context, input *s3.CreateBucketInput, opts ...request.Option) (*s3.CreateBucketOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &s3.CreateBucketOutput{Location: aws.String("/testbucket")}, nil
}

func (m *mockS3Client) DeleteBucketWithContext(ctx context.Context, input *s3.DeleteBucketInput, opts ...request.Option) (*s3.DeleteBucketOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &s3.DeleteBucketOutput{}, nil
}

func TestCreateBucket(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	// test success
	expected := &s3.CreateBucketOutput{Location: aws.String("/testbucket")}
	out, err := s.CreateBucket(context.TODO(), &s3.CreateBucketInput{Bucket: aws.String("testbucket")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = s.CreateBucket(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty bucket name
	_, err = s.CreateBucket(context.TODO(), &s3.CreateBucketInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeBucketAlreadyExists
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeBucketAlreadyExists, "already exists", nil)
	_, err = s.CreateBucket(context.TODO(), &s3.CreateBucketInput{Bucket: aws.String("testbucket")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeBucketAlreadyOwnedByYou
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeBucketAlreadyOwnedByYou, "already exists and is owned by you", nil)
	_, err = s.CreateBucket(context.TODO(), &s3.CreateBucketInput{Bucket: aws.String("testbucket")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchKey, "no such key", nil)
	_, err = s.CreateBucket(context.TODO(), &s3.CreateBucketInput{Bucket: aws.String("testbucket")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	_, err = s.CreateBucket(context.TODO(), &s3.CreateBucketInput{Bucket: aws.String("testbucket")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteBucket(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	// test success
	expected := &s3.DeleteBucketOutput{}
	out, err := s.DeleteEmptyBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: aws.String("testbucket")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = s.DeleteEmptyBucket(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty bucket name
	_, err = s.DeleteEmptyBucket(context.TODO(), &s3.DeleteBucketInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchBucket
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchBucket, "bucket not found", nil)
	_, err = s.DeleteEmptyBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: aws.String("testbucket")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test NotFound
	s.Service.(*mockS3Client).err = awserr.New("NotFound", "bucket not found", nil)
	_, err = s.DeleteEmptyBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: aws.String("testbucket")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test BucketNotEmpty
	s.Service.(*mockS3Client).err = awserr.New("BucketNotEmpty", "bucket not empty", nil)
	_, err = s.DeleteEmptyBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: aws.String("testbucket")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test Forbidden
	s.Service.(*mockS3Client).err = awserr.New("Forbidden", "bucket not empty", nil)
	_, err = s.DeleteEmptyBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: aws.String("testbucket")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrForbidden {
			t.Errorf("expected error code %s, got: %s", apierror.ErrForbidden, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchKey, "no such key", nil)
	_, err = s.DeleteEmptyBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: aws.String("testbucket")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	_, err = s.DeleteEmptyBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: aws.String("testbucket")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
