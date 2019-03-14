package s3

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
)

var testBucket1 = s3.Bucket{
	CreationDate: &testTime,
	Name:         aws.String("testbucket1"),
}

var testBucket2 = s3.Bucket{
	CreationDate: &testTime,
	Name:         aws.String("testbucket2"),
}

var testBucket3 = s3.Bucket{
	CreationDate: &testTime,
	Name:         aws.String("testbucket3"),
}

var testBuckets1 = []*s3.Bucket{&testBucket1, &testBucket2, &testBucket3}

var testTags1 = []*s3.Tag{
	&s3.Tag{Key: aws.String("foo"), Value: aws.String("bar")},
	&s3.Tag{Key: aws.String("fuz"), Value: aws.String("buz")},
	&s3.Tag{Key: aws.String("fiz"), Value: aws.String("biz")},
}

func (m *mockS3Client) HeadBucketWithContext(ctx context.Context, input *s3.HeadBucketInput, opts ...request.Option) (*s3.HeadBucketOutput, error) {
	if aws.StringValue(input.Bucket) == "testbucket" {
		return nil, awserr.New(s3.ErrCodeNoSuchBucket, "Not Found", nil)
	}

	if strings.HasSuffix(aws.StringValue(input.Bucket), "-exists") {
		return &s3.HeadBucketOutput{}, nil
	}

	if strings.HasSuffix(aws.StringValue(input.Bucket), "-missing") {
		return nil, awserr.New(s3.ErrCodeNoSuchBucket, "Not Found", nil)
	}

	return &s3.HeadBucketOutput{}, nil
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

func (m *mockS3Client) ListBucketsWithContext(ctx context.Context, input *s3.ListBucketsInput, opts ...request.Option) (*s3.ListBucketsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &s3.ListBucketsOutput{Buckets: testBuckets1}, nil
}

func (m *mockS3Client) GetBucketTaggingWithContext(ctx context.Context, input *s3.GetBucketTaggingInput, opts ...request.Option) (*s3.GetBucketTaggingOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &s3.GetBucketTaggingOutput{TagSet: testTags1}, nil
}

func (m *mockS3Client) PutBucketTaggingWithContext(ctx context.Context, input *s3.PutBucketTaggingInput, opts ...request.Option) (*s3.PutBucketTaggingOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &s3.PutBucketTaggingOutput{}, nil
}

func (m *mockS3Client) PutBucketWebsiteWithContext(ctx context.Context, input *s3.PutBucketWebsiteInput, opts ...request.Option) (*s3.PutBucketWebsiteOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	// check that IndexDocument is set
	if input.WebsiteConfiguration.IndexDocument == nil {
		return nil, errors.New("expected index.html to be set for IndexDocument")
	}

	return &s3.PutBucketWebsiteOutput{}, nil
}

func (m *mockS3Client) PutBucketPolicyWithContext(ctx context.Context, input *s3.PutBucketPolicyInput, opts ...request.Option) (*s3.PutBucketPolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &s3.PutBucketPolicyOutput{}, nil
}

func (m *mockS3Client) ListObjectsV2WithContext(ctx context.Context, input *s3.ListObjectsV2Input, opts ...request.Option) (*s3.ListObjectsV2Output, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.Bucket) == "testBucketNotEmpty" {
		return &s3.ListObjectsV2Output{KeyCount: aws.Int64(int64(1))}, nil
	}

	return &s3.ListObjectsV2Output{KeyCount: aws.Int64(int64(0))}, nil
}

func TestBucketExists(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	exists, err := s.BucketExists(context.TODO(), "testbucket-exists")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !exists {
		t.Errorf("expected testbucket-exists to exist (true), got false")
	}

	notexists, err := s.BucketExists(context.TODO(), "testbucket-missing")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if notexists {
		t.Errorf("expected testbucket-missing to not exist (false), got true")
	}
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

func TestListBuckets(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	// test success
	expected := testBuckets1
	out, err := s.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test some unexpected AWS error
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchUpload, "no such upload", nil)
	_, err = s.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	_, err = s.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestGetBucketTags(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	// test success
	expected := testTags1
	out, err := s.GetBucketTags(context.TODO(), "testBucket1")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test empty bucket
	out, err = s.GetBucketTags(context.TODO(), "")
	if err == nil {
		t.Error("expected api error for empty bucket, got nil")
	}

	if len(out) != 0 {
		t.Errorf("expected empty tags list for empty bucket, got %d entries", len(out))
	}

	// test empty tagset
	expected = []*s3.Tag{}
	s.Service.(*mockS3Client).err = awserr.New("NoSuchTagSet", "no such upload", nil)
	out, err = s.GetBucketTags(context.TODO(), "testBucket1")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test some unexpected AWS error
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchUpload, "no such upload", nil)
	_, err = s.GetBucketTags(context.TODO(), "testBucket1")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	_, err = s.GetBucketTags(context.TODO(), "testBucket1")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestTagBucket(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	// test success
	err := s.TagBucket(context.TODO(), "testBucket1", testTags1)
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// test empty tags
	err = s.TagBucket(context.TODO(), "testBucket1", []*s3.Tag{})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// test some unexpected AWS error
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchUpload, "no such upload", nil)
	err = s.TagBucket(context.TODO(), "testBucket1", testTags1)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	err = s.TagBucket(context.TODO(), "testBucket1", testTags1)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestUpdateWebsiteConfig(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	// test success
	err := s.UpdateWebsiteConfig(context.TODO(), &s3.PutBucketWebsiteInput{
		Bucket:               aws.String("testbucket"),
		WebsiteConfiguration: &s3.WebsiteConfiguration{},
	})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// test nil input
	err = s.UpdateWebsiteConfig(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty bucket name and website configuration
	err = s.UpdateWebsiteConfig(context.TODO(), &s3.PutBucketWebsiteInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchKey, "no such key", nil)
	err = s.UpdateWebsiteConfig(context.TODO(), &s3.PutBucketWebsiteInput{
		Bucket:               aws.String("testbucket"),
		WebsiteConfiguration: &s3.WebsiteConfiguration{},
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	err = s.UpdateWebsiteConfig(context.TODO(), &s3.PutBucketWebsiteInput{
		Bucket:               aws.String("testbucket"),
		WebsiteConfiguration: &s3.WebsiteConfiguration{},
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestUpdateBucketPolicy(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	// test success
	err := s.UpdateBucketPolicy(context.TODO(), &s3.PutBucketPolicyInput{
		Bucket: aws.String("testbucket"),
		Policy: aws.String("somepolicy"),
	})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// test nil input
	err = s.UpdateBucketPolicy(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty bucket name and policy
	err = s.UpdateBucketPolicy(context.TODO(), &s3.PutBucketPolicyInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchKey, "no such key", nil)
	err = s.UpdateBucketPolicy(context.TODO(), &s3.PutBucketPolicyInput{
		Bucket: aws.String("testbucket"),
		Policy: aws.String("somepolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	err = s.UpdateBucketPolicy(context.TODO(), &s3.PutBucketPolicyInput{
		Bucket: aws.String("testbucket"),
		Policy: aws.String("somepolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestBucketEmpty(t *testing.T) {
	s := S3{Service: newMockS3Client(t, nil)}

	// test successful empty bucket
	empty, err := s.BucketEmpty(context.TODO(), "testBucket")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !empty {
		t.Error("expected testBucket bucket to be empty")
	}

	// test successful not empty bucket
	empty, err = s.BucketEmpty(context.TODO(), "testBucketNotEmpty")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if empty {
		t.Error("expected testBucketNotEmpty bucket to not be empty")
	}

	// test empty bucket name
	empty, err = s.BucketEmpty(context.TODO(), "")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	s.Service.(*mockS3Client).err = awserr.New(s3.ErrCodeNoSuchKey, "no such key", nil)
	empty, err = s.BucketEmpty(context.TODO(), "testBucket")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	s.Service.(*mockS3Client).err = errors.New("things blowing up!")
	empty, err = s.BucketEmpty(context.TODO(), "testBucket")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
