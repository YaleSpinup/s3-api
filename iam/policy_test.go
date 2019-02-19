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

var testPolicy = iam.Policy{
	Arn:                           aws.String("arn:aws:iam::12345678910:group/testpolicy"),
	AttachmentCount:               aws.Int64(0),
	CreateDate:                    &testTime,
	DefaultVersionId:              aws.String("v1"),
	Description:                   aws.String("policy thang"),
	IsAttachable:                  aws.Bool(true),
	Path:                          aws.String("/"),
	PermissionsBoundaryUsageCount: aws.Int64(0),
	PolicyId:                      aws.String("TESTPOLICYID123"),
	PolicyName:                    aws.String("testpolicy"),
	UpdateDate:                    &testTime,
}

func (m *mockIAMClient) CreatePolicyWithContext(ctx context.Context, input *iam.CreatePolicyInput, opts ...request.Option) (*iam.CreatePolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.CreatePolicyOutput{Policy: &testPolicy}, nil
}

func (m *mockIAMClient) DeletePolicyWithContext(ctx context.Context, input *iam.DeletePolicyInput, opts ...request.Option) (*iam.DeletePolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.DeletePolicyOutput{}, nil
}

func TestCreatePolicy(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.CreatePolicyOutput{Policy: &testPolicy}

	// build the default IAM bucket admin policy (from the config and known inputs)
	defaultPolicy, err := i.DefaultBucketAdminPolicy(aws.String("testBucket"))
	if err != nil {
		t.Errorf("expected nil error creating default policy doc, got %s", err)
	}

	out, err := i.CreatePolicy(context.TODO(), &iam.CreatePolicyInput{
		Description:    aws.String("policy thang"),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String("testpolicy"),
	})

	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.CreatePolicy(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeInvalidInputException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeInvalidInputException, "invalid input", nil)
	_, err = i.CreatePolicy(context.TODO(), &iam.CreatePolicyInput{
		Description:    aws.String("policy thang"),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String("testpolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeMalformedPolicyDocumentException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeMalformedPolicyDocumentException, "malformed policy document", nil)
	_, err = i.CreatePolicy(context.TODO(), &iam.CreatePolicyInput{
		Description:    aws.String("policy thang"),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String("testpolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	_, err = i.CreatePolicy(context.TODO(), &iam.CreatePolicyInput{
		Description:    aws.String("policy thang"),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String("testpolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeEntityAlreadyExistsException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeEntityAlreadyExistsException, "policy already exists", nil)
	_, err = i.CreatePolicy(context.TODO(), &iam.CreatePolicyInput{
		Description:    aws.String("policy thang"),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String("testpolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failure", nil)
	_, err = i.CreatePolicy(context.TODO(), &iam.CreatePolicyInput{
		Description:    aws.String("policy thang"),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String("testpolicy"),
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
	_, err = i.CreatePolicy(context.TODO(), &iam.CreatePolicyInput{
		Description:    aws.String("policy thang"),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String("testpolicy"),
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
	_, err = i.CreatePolicy(context.TODO(), &iam.CreatePolicyInput{
		Description:    aws.String("policy thang"),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String("testpolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeletePolicy(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &iam.DeletePolicyOutput{}
	out, err := i.DeletePolicy(context.TODO(), &iam.DeletePolicyInput{PolicyArn: aws.String("arn:aws:iam::12345678910:group/testpolicy")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.DeletePolicy(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty policy arn
	_, err = i.DeletePolicy(context.TODO(), &iam.DeletePolicyInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "not found", nil)
	_, err = i.DeletePolicy(context.TODO(), &iam.DeletePolicyInput{PolicyArn: aws.String("arn:aws:iam::12345678910:group/testpolicy")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "not found", nil)
	_, err = i.DeletePolicy(context.TODO(), &iam.DeletePolicyInput{PolicyArn: aws.String("arn:aws:iam::12345678910:group/testpolicy")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeInvalidInputException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeInvalidInputException, "not found", nil)
	_, err = i.DeletePolicy(context.TODO(), &iam.DeletePolicyInput{PolicyArn: aws.String("arn:aws:iam::12345678910:group/testpolicy")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeDeleteConflictException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "not found", nil)
	_, err = i.DeletePolicy(context.TODO(), &iam.DeletePolicyInput{PolicyArn: aws.String("arn:aws:iam::12345678910:group/testpolicy")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "not found", nil)
	_, err = i.DeletePolicy(context.TODO(), &iam.DeletePolicyInput{PolicyArn: aws.String("arn:aws:iam::12345678910:group/testpolicy")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeEntityAlreadyExistsException, "policy exists", nil)
	_, err = i.DeletePolicy(context.TODO(), &iam.DeletePolicyInput{PolicyArn: aws.String("arn:aws:iam::12345678910:group/testpolicy")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.DeletePolicy(context.TODO(), &iam.DeletePolicyInput{PolicyArn: aws.String("arn:aws:iam::12345678910:group/testpolicy")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
