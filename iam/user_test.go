package iam

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/iam"
)

var testUser = iam.User{
	Arn:                 aws.String("arn:aws:iam::12345678910:user/testuser"),
	CreateDate:          &testTime,
	PasswordLastUsed:    nil,
	Path:                aws.String("/"),
	PermissionsBoundary: nil,
	Tags:                []*iam.Tag{},
	UserId:              aws.String("TESTUSERID123"),
	UserName:            aws.String("testuser"),
}

var testAccessKey = iam.AccessKey{
	CreateDate:      &testTime,
	AccessKeyId:     aws.String("SOMEACCESSKEYID"),
	SecretAccessKey: aws.String("SSSSSHHHHHHHHHHHHHHDONTTELLANYONEYOUSAWTHISSSSSS"),
	Status:          aws.String("Active"),
	UserName:        aws.String("testuser"),
}

var testAccessKeyMetadata1 = iam.AccessKeyMetadata{
	CreateDate:  &testTime,
	AccessKeyId: aws.String("SOMEACCESSKEYID1"),
	Status:      aws.String("Active"),
	UserName:    aws.String("testuser"),
}

var testAccessKeyMetadata2 = iam.AccessKeyMetadata{
	CreateDate:  &testTime,
	AccessKeyId: aws.String("SOMEACCESSKEYID2"),
	Status:      aws.String("Active"),
	UserName:    aws.String("testuser"),
}

var testAccessKeyMetadata3 = iam.AccessKeyMetadata{
	CreateDate:  &testTime,
	AccessKeyId: aws.String("SOMEACCESSKEYID3"),
	Status:      aws.String("Active"),
	UserName:    aws.String("testuser"),
}

var testAccessKeysMetadata1 = []*iam.AccessKeyMetadata{&testAccessKeyMetadata1, &testAccessKeyMetadata2, &testAccessKeyMetadata3}

var testUserGroup1 = iam.Group{
	Arn:        aws.String("arn:aws:iam::12345678910:group/testgroup1"),
	CreateDate: &testTime,
	GroupId:    aws.String("TESTGROUPID123"),
	GroupName:  aws.String("testgroup1"),
	Path:       aws.String("/"),
}

var testUserGroup2 = iam.Group{
	Arn:        aws.String("arn:aws:iam::12345678910:group/testgroup2"),
	CreateDate: &testTime,
	GroupId:    aws.String("TESTGROUPID223"),
	GroupName:  aws.String("testgroup2"),
	Path:       aws.String("/"),
}

var testUserGroup3 = iam.Group{
	Arn:        aws.String("arn:aws:iam::12345678910:group/testgroup3"),
	CreateDate: &testTime,
	GroupId:    aws.String("TESTGROUPID323"),
	GroupName:  aws.String("testgroup3"),
	Path:       aws.String("/"),
}

var testGroups1 = []*iam.Group{&testUserGroup1, &testUserGroup2, &testUserGroup3}

func (m *mockIAMClient) CreateUserWithContext(ctx context.Context, input *iam.CreateUserInput, opts ...request.Option) (*iam.CreateUserOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.CreateUserOutput{User: &testUser}, nil
}

func (m *mockIAMClient) DeleteUserWithContext(ctx context.Context, input *iam.DeleteUserInput, opts ...request.Option) (*iam.DeleteUserOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.DeleteUserOutput{}, nil
}

func (m *mockIAMClient) CreateAccessKeyWithContext(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...request.Option) (*iam.CreateAccessKeyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.CreateAccessKeyOutput{AccessKey: &testAccessKey}, nil
}

func (m *mockIAMClient) DeleteAccessKeyWithContext(ctx context.Context, input *iam.DeleteAccessKeyInput, opts ...request.Option) (*iam.DeleteAccessKeyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.DeleteAccessKeyOutput{}, nil
}

func (m *mockIAMClient) ListAccessKeysWithContext(ctx context.Context, input *iam.ListAccessKeysInput, opts ...request.Option) (*iam.ListAccessKeysOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &iam.ListAccessKeysOutput{AccessKeyMetadata: testAccessKeysMetadata1}, nil
}

func (m *mockIAMClient) AddUserToGroupWithContext(ctx context.Context, input *iam.AddUserToGroupInput, opts ...request.Option) (*iam.AddUserToGroupOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.AddUserToGroupOutput{}, nil
}

func (m *mockIAMClient) RemoveUserFromGroupWithContext(ctx context.Context, input *iam.RemoveUserFromGroupInput, opts ...request.Option) (*iam.RemoveUserFromGroupOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.RemoveUserFromGroupOutput{}, nil
}

func (m *mockIAMClient) ListGroupsForUserWithContext(ctx context.Context, input *iam.ListGroupsForUserInput, opts ...request.Option) (*iam.ListGroupsForUserOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.ListGroupsForUserOutput{Groups: testGroups1}, nil
}

func TestCreateUser(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.CreateUserOutput{User: &testUser}
	out, err := i.CreateUser(context.TODO(), &iam.CreateUserInput{UserName: aws.String("testuser")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.CreateUser(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty user name
	_, err = i.CreateUser(context.TODO(), &iam.CreateUserInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	_, err = i.CreateUser(context.TODO(), &iam.CreateUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeEntityAlreadyExistsException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeEntityAlreadyExistsException, "already exists", nil)
	_, err = i.CreateUser(context.TODO(), &iam.CreateUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	_, err = i.CreateUser(context.TODO(), &iam.CreateUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeInvalidInputException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeInvalidInputException, "invalid input", nil)
	_, err = i.CreateUser(context.TODO(), &iam.CreateUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeConcurrentModificationException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeConcurrentModificationException, "service failed", nil)
	_, err = i.CreateUser(context.TODO(), &iam.CreateUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failed", nil)
	_, err = i.CreateUser(context.TODO(), &iam.CreateUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	_, err = i.CreateUser(context.TODO(), &iam.CreateUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.CreateUser(context.TODO(), &iam.CreateUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteUser(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.DeleteUserOutput{}
	out, err := i.DeleteUser(context.TODO(), &iam.DeleteUserInput{UserName: aws.String("testuser")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.DeleteUser(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty user name
	_, err = i.DeleteUser(context.TODO(), &iam.DeleteUserInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	_, err = i.DeleteUser(context.TODO(), &iam.DeleteUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	_, err = i.DeleteUser(context.TODO(), &iam.DeleteUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeDeleteConflictException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "invalid input", nil)
	_, err = i.DeleteUser(context.TODO(), &iam.DeleteUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeConcurrentModificationException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeConcurrentModificationException, "service failed", nil)
	_, err = i.DeleteUser(context.TODO(), &iam.DeleteUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failed", nil)
	_, err = i.DeleteUser(context.TODO(), &iam.DeleteUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeEntityAlreadyExistsException, "delete conflict", nil)
	_, err = i.DeleteUser(context.TODO(), &iam.DeleteUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.DeleteUser(context.TODO(), &iam.DeleteUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestCreateAccessKey(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.CreateAccessKeyOutput{AccessKey: &testAccessKey}
	out, err := i.CreateAccessKey(context.TODO(), &iam.CreateAccessKeyInput{UserName: aws.String("testuser")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.CreateAccessKey(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty user name
	_, err = i.CreateAccessKey(context.TODO(), &iam.CreateAccessKeyInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	_, err = i.CreateAccessKey(context.TODO(), &iam.CreateAccessKeyInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	_, err = i.CreateAccessKey(context.TODO(), &iam.CreateAccessKeyInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failed", nil)
	_, err = i.CreateAccessKey(context.TODO(), &iam.CreateAccessKeyInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	_, err = i.CreateAccessKey(context.TODO(), &iam.CreateAccessKeyInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.CreateAccessKey(context.TODO(), &iam.CreateAccessKeyInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteAccessKeys(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.DeleteAccessKeyOutput{}
	out, err := i.DeleteAccessKey(context.TODO(), &iam.DeleteAccessKeyInput{
		UserName:    aws.String("testuser"),
		AccessKeyId: aws.String("SOMEACCESSKEYID"),
	})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.DeleteAccessKey(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty user name
	_, err = i.DeleteAccessKey(context.TODO(), &iam.DeleteAccessKeyInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "not found", nil)
	_, err = i.DeleteAccessKey(context.TODO(), &iam.DeleteAccessKeyInput{
		UserName:    aws.String("testuser"),
		AccessKeyId: aws.String("SOMEACCESSKEYID"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	_, err = i.DeleteAccessKey(context.TODO(), &iam.DeleteAccessKeyInput{
		UserName:    aws.String("testuser"),
		AccessKeyId: aws.String("SOMEACCESSKEYID"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failure", nil)
	_, err = i.DeleteAccessKey(context.TODO(), &iam.DeleteAccessKeyInput{
		UserName:    aws.String("testuser"),
		AccessKeyId: aws.String("SOMEACCESSKEYID"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	_, err = i.DeleteAccessKey(context.TODO(), &iam.DeleteAccessKeyInput{
		UserName:    aws.String("testuser"),
		AccessKeyId: aws.String("SOMEACCESSKEYID"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.DeleteAccessKey(context.TODO(), &iam.DeleteAccessKeyInput{
		UserName:    aws.String("testuser"),
		AccessKeyId: aws.String("SOMEACCESSKEYID"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestListAccessKeys(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := testAccessKeysMetadata1
	out, err := i.ListAccessKeys(context.TODO(), &iam.ListAccessKeysInput{UserName: aws.String("testuser")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.ListAccessKeys(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty user name
	_, err = i.ListAccessKeys(context.TODO(), &iam.ListAccessKeysInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "not found", nil)
	_, err = i.ListAccessKeys(context.TODO(), &iam.ListAccessKeysInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failure", nil)
	_, err = i.ListAccessKeys(context.TODO(), &iam.ListAccessKeysInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	_, err = i.ListAccessKeys(context.TODO(), &iam.ListAccessKeysInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.ListAccessKeys(context.TODO(), &iam.ListAccessKeysInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestAddUserToGroup(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.AddUserToGroupOutput{}
	out, err := i.AddUserToGroup(context.TODO(), &iam.AddUserToGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.AddUserToGroup(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty user name and group name
	_, err = i.AddUserToGroup(context.TODO(), &iam.AddUserToGroupInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "not found", nil)
	_, err = i.AddUserToGroup(context.TODO(), &iam.AddUserToGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	_, err = i.AddUserToGroup(context.TODO(), &iam.AddUserToGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failure", nil)
	_, err = i.AddUserToGroup(context.TODO(), &iam.AddUserToGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	_, err = i.AddUserToGroup(context.TODO(), &iam.AddUserToGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.AddUserToGroup(context.TODO(), &iam.AddUserToGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestRemoveUserFromGroup(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.RemoveUserFromGroupOutput{}
	out, err := i.RemoveUserFromGroup(context.TODO(), &iam.RemoveUserFromGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.RemoveUserFromGroup(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty user name and group name
	_, err = i.RemoveUserFromGroup(context.TODO(), &iam.RemoveUserFromGroupInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "not found", nil)
	_, err = i.RemoveUserFromGroup(context.TODO(), &iam.RemoveUserFromGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	_, err = i.RemoveUserFromGroup(context.TODO(), &iam.RemoveUserFromGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failure", nil)
	_, err = i.RemoveUserFromGroup(context.TODO(), &iam.RemoveUserFromGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	_, err = i.RemoveUserFromGroup(context.TODO(), &iam.RemoveUserFromGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.RemoveUserFromGroup(context.TODO(), &iam.RemoveUserFromGroupInput{UserName: aws.String("testuser"), GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestListUserGroups(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := testGroups1
	out, err := i.ListUserGroups(context.TODO(), &iam.ListGroupsForUserInput{UserName: aws.String("testuser")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.ListUserGroups(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty user name
	_, err = i.ListUserGroups(context.TODO(), &iam.ListGroupsForUserInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "not found", nil)
	_, err = i.ListUserGroups(context.TODO(), &iam.ListGroupsForUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	_, err = i.ListUserGroups(context.TODO(), &iam.ListGroupsForUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.ListUserGroups(context.TODO(), &iam.ListGroupsForUserInput{UserName: aws.String("testuser")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}