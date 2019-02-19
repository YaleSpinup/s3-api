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

var testGroup = iam.Group{
	Arn:        aws.String("arn:aws:iam::12345678910:group/testgroup"),
	CreateDate: &testTime,
	GroupId:    aws.String("TESTGROUPID123"),
	GroupName:  aws.String("testgroup"),
	Path:       aws.String("/"),
}

func (m *mockIAMClient) CreateGroupWithContext(ctx context.Context, input *iam.CreateGroupInput, opts ...request.Option) (*iam.CreateGroupOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.CreateGroupOutput{Group: &testGroup}, nil
}

func (m *mockIAMClient) DeleteGroupWithContext(ctx context.Context, input *iam.DeleteGroupInput, opts ...request.Option) (*iam.DeleteGroupOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.DeleteGroupOutput{}, nil
}

func (m *mockIAMClient) AttachGroupPolicyWithContext(ctx context.Context, input *iam.AttachGroupPolicyInput, opts ...request.Option) (*iam.AttachGroupPolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.AttachGroupPolicyOutput{}, nil
}

func TestCreateGroup(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.CreateGroupOutput{Group: &testGroup}
	out, err := i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.CreateGroup(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty group name
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeEntityAlreadyExistsException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeEntityAlreadyExistsException, "already exists", nil)
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failed", nil)
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteGroup(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.DeleteGroupOutput{}
	out, err := i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.DeleteGroup(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty group name
	_, err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	_, err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	_, err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeDeleteConflictException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	_, err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failed", nil)
	_, err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeEntityAlreadyExistsException, "entity already exists", nil)
	_, err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestAttachGroupPolicy(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.AttachGroupPolicyOutput{}
	out, err := i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.AttachGroupPolicy(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty group name and empty policy arn
	_, err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	_, err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
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
	_, err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeInvalidInputException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeInvalidInputException, "limit exceeded", nil)
	_, err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodePolicyNotAttachableException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodePolicyNotAttachableException, "limit exceeded", nil)
	_, err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "limit exceeded", nil)
	_, err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "entity already exists", nil)
	_, err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
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
	_, err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
