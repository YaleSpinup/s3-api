package s3

import (
	"bytes"
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

func (m *mockS3Client) PutObjectWithContext(ctx aws.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &s3.PutObjectOutput{}, nil
}

func TestCreateObject(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	expected := &s3.PutObjectOutput{}
	input := s3.PutObjectInput{
		Bucket: aws.String("testbucket"),
		Body:   bytes.NewReader([]byte("hi")),
	}
	// test success
	out, err := s.CreateObject(context.TODO(), &input)
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(expected, out) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	if _, err = s.CreateObject(context.TODO(), nil); err == nil {
		t.Error("expected error, got nil")
	}

	// test missing bucket
	if _, err = s.CreateObject(context.TODO(), &s3.PutObjectInput{Body: bytes.NewReader([]byte("hi"))}); err == nil {
		t.Error("expected error, got nil")
	}

	// test missing body
	if _, err = s.CreateObject(context.TODO(), &s3.PutObjectInput{Bucket: aws.String("testbucket")}); err == nil {
		t.Error("expected error, got nil")
	}

	// test AccessDenied
	s.Service.(*mockS3Client).err = awserr.New("AccessDenied", "you are forbidden", nil)
	_, err = s.CreateObject(context.TODO(), &input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrForbidden {
			t.Errorf("expected error code %s, got: %s", apierror.ErrForbidden, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test InvalidBucketState
	s.Service.(*mockS3Client).err = awserr.New("InvalidBucketState", "in progress", nil)
	_, err = s.CreateObject(context.TODO(), &input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchBucket
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchBucket, "not found", nil)
	_, err = s.CreateObject(context.TODO(), &input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test EntityTooSmall
	s.Service.(*mockS3Client).err = awserr.New("EntityTooSmall", "too small", nil)
	_, err = s.CreateObject(context.TODO(), &input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	_, err = s.CreateObject(context.TODO(), &input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
