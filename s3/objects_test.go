package s3

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
)

var testObjectTags = []*s3.Tag{
	{
		Key:   aws.String("FirstName"),
		Value: aws.String("Handsome"),
	},
	{
		Key:   aws.String("LastName"),
		Value: aws.String("Dan"),
	},
}

// mock object store
var testHasObjects = []*s3.GetObjectInput{
	{
		Bucket: aws.String("foo.baz.org"),
		Key:    aws.String("/index.html"),
	},
	{
		Bucket: aws.String("foo.bar.org"),
		Key:    aws.String("/spinup/index.html"),
	},
	{
		Bucket: aws.String("foo.bag.org"),
		Key:    aws.String("/foo/index.html"),
	},
}

func hasTestHasObject(input *s3.GetObjectInput) bool {
	for _, obj := range testHasObjects {
		aBkt := aws.StringValue(obj.Bucket)
		bBkt := aws.StringValue(input.Bucket)
		aKey := aws.StringValue(obj.Key)
		bKey := aws.StringValue(input.Key)

		if aBkt == bBkt && aKey == bKey {
			return true
		}
	}

	return false
}

func (m *mockS3Client) GetObjectTaggingWithContext(ctx context.Context, input *s3.GetObjectTaggingInput, opts ...request.Option) (*s3.GetObjectTaggingOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &s3.GetObjectTaggingOutput{
		TagSet: testObjectTags,
	}, nil
}

func (m *mockS3Client) GetObjectWithContext(ctx context.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	successResponse := &s3.GetObjectOutput{}
	errorResponse := errors.New("object not found")

	if hasTestHasObject(input) {
		return successResponse, nil
	} else {
		return &s3.GetObjectOutput{}, errorResponse
	}
}

func (m *mockS3Client) PutObjectWithContext(ctx context.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) DeleteObjectWithContext(ctx context.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.Key) == "notfound.txt" {
		return nil, awserr.New(s3.ErrCodeNoSuchKey, "Not Found", nil)
	}
	return nil, nil
}

func TestHasObjectWithRootKey(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	input := s3.GetObjectInput{
		Bucket: aws.String("foo.baz.org"),
		Key:    aws.String("/index.html"),
	}

	out, err := s.HasObject(context.TODO(), &input)
	if err != nil {
		t.Errorf("unexpected error. item was supposed to be found with no errors: %s", err)
	}

	if !out {
		t.Errorf("expected %+v, got %+v", true, out)
	}
}

func TestHasObjectWithScopedKey(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	input := s3.GetObjectInput{
		Bucket: aws.String("foo.bar.org"),
		Key:    aws.String("/spinup/index.html"),
	}

	out, err := s.HasObject(context.TODO(), &input)
	if err != nil {
		t.Errorf("unexpected error. item was supposed to be found with no errors: %s", err)
	}

	if !out {
		t.Errorf("expected %+v, got %+v", true, out)
	}
}

func TestHasObjectWithMissingItem(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	input := s3.GetObjectInput{
		Bucket: aws.String("foo.bar.org"),
		Key:    aws.String("/foo/index.html"),
	}

	out, err := s.HasObject(context.TODO(), &input)
	if err != nil {
		t.Errorf("unexpected error. item was supposed to be missing with no errors: %s", err)
	}

	if out {
		t.Errorf("expected %+v, got %+v", false, out)
	}
}

func TestHasObjectWithMissingInputs(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	expected := apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))

	_, err := s.HasObject(context.TODO(), nil)
	if !reflect.DeepEqual(expected, err) {
		t.Errorf("expected %+v, got %+v", expected, err)
	}

	input := s3.GetObjectInput{}
	expected = apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("missing bucket name"))

	_, err = s.HasObject(context.TODO(), &input)
	if !reflect.DeepEqual(expected, err) {
		t.Errorf("expected %+v, got %+v", expected, err)
	}

	input = s3.GetObjectInput{Bucket: aws.String("foo.bar.org")}
	expected = apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("missing key name"))

	_, err = s.HasObject(context.TODO(), &input)
	if !reflect.DeepEqual(expected, err) {
		t.Errorf("expected %+v, got %+v", expected, err)
	}
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

func TestGetObjectTagging(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	expected := testObjectTags
	input := s3.GetObjectTaggingInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("index.html"),
	}
	// test success
	out, err := s.GetObjectTagging(context.TODO(), &input)
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(expected, out) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	input = s3.GetObjectTaggingInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("/index.html"),
	}

	// test success
	out, err = s.GetObjectTagging(context.TODO(), &input)
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(expected, out) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test ErrCodeNoSuchBucket
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchBucket, "not found", nil)
	_, err = s.GetObjectTagging(context.TODO(), &input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchKey
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchKey, "not found", nil)
	_, err = s.GetObjectTagging(context.TODO(), &input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	_, err = s.GetObjectTagging(context.TODO(), &input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteObject(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}
	input := &s3.DeleteObjectInput{
		Bucket: aws.String("testBucket"),
		Key:    aws.String("index.html"),
	}

	// test success
	_, err := s.DeleteObject(context.TODO(), input)
	if err != nil {
		t.Errorf("expected nil error for delete, got %s", err)
	}

	input = &s3.DeleteObjectInput{
		Bucket: aws.String("testBucket"),
		Key:    aws.String("/index.html"),
	}

	// test success with / prefix
	_, err = s.DeleteObject(context.TODO(), input)
	if err != nil {
		t.Errorf("expected nil error for delete, got %s", err)
	}

	// test nil input
	_, err = s.DeleteObject(context.TODO(), nil)
	if err == nil {
		t.Error("expected error for delete, got nil")
	}

	// test missing bucket input
	_, err = s.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Key: aws.String("index.html"),
	})
	if err == nil {
		t.Error("expected error for delete, got nil")
	}

	// test missing key input
	_, err = s.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String("testBucket"),
	})
	if err == nil {
		t.Error("expected error for delete, got nil")
	}

	// test not found key input
	_, err = s.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String("testBucket"),
		Key:    aws.String("notfound.txt"),
	})
	if err == nil {
		t.Error("expected error for delete, got nil")
	}

	// test ErrCodeNoSuchBucket
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchBucket, "not found", nil)
	_, err = s.DeleteObject(context.TODO(), input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchKey
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchKey, "not found", nil)
	_, err = s.DeleteObject(context.TODO(), input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	_, err = s.DeleteObject(context.TODO(), input)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
