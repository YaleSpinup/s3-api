package iam

import (
	"context"
	"fmt"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// CreateGroup handles creating an IAM group
func (i *IAM) CreateGroup(ctx context.Context, input *iam.CreateGroupInput) (*iam.CreateGroupOutput, error) {
	log.Infof("creating IAM group: %s", aws.StringValue(input.GroupName))
	output, err := i.Service.CreateGroupWithContext(ctx, input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeEntityAlreadyExistsException "EntityAlreadyExists"
			// The request was rejected because it attempted to create a resource that already
			// exists.
			case iam.ErrCodeEntityAlreadyExistsException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrConflict, msg, err)
			// * ErrCodeNoSuchEntityException "NoSuchEntity"
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrBadRequest, msg, err)
			// * ErrCodeServiceFailureException "ServiceFailure"
			// The request processing has failed because of an unknown error, exception
			// or failure
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrServiceUnavailable, msg, err)
			default:
				return nil, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return nil, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return output, nil
}

// DeleteGroup handles deleting an IAM group
func (i *IAM) DeleteGroup(ctx context.Context, input *iam.DeleteGroupInput) (*iam.DeleteGroupOutput, error) {
	if input == nil || aws.StringValue(input.GroupName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting iam group %s", aws.StringValue(input.GroupName))

	output, err := i.Service.DeleteGroupWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeNoSuchEntityException "NoSuchEntity"
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeDeleteConflictException "DeleteConflict"
			// The request was rejected because it attempted to delete a resource that has
			// attached subordinate entities. The error message describes these entities.
			case iam.ErrCodeDeleteConflictException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrConflict, msg, err)
			// * ErrCodeServiceFailureException "ServiceFailure"
			// The request processing has failed because of an unknown error, exception
			// or failure.
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrServiceUnavailable, msg, err)
			default:
				return nil, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}
		return nil, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return output, nil
}

func (i *IAM) AttachGroupPolicy(ctx context.Context, input *iam.AttachGroupPolicyInput) (*iam.AttachGroupPolicyOutput, error) {
	if input == nil || aws.StringValue(input.GroupName) == "" || aws.StringValue(input.PolicyArn) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	output, err := i.Service.AttachGroupPolicyWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeNoSuchEntityException "NoSuchEntity"
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeInvalidInputException "InvalidInput"
			// The request was rejected because an invalid or out-of-range value was supplied
			// for an input parameter.
			// * ErrCodePolicyNotAttachableException "PolicyNotAttachable"
			// The request failed because AWS service role policies can only be attached
			// to the service-linked role for that service.
			case iam.ErrCodeInvalidInputException, iam.ErrCodePolicyNotAttachableException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrBadRequest, msg, err)
			// * ErrCodeServiceFailureException "ServiceFailure"
			// The request processing has failed because of an unknown error, exception
			// or failure.
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrServiceUnavailable, msg, err)
			default:
				return nil, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return nil, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return output, nil
}
