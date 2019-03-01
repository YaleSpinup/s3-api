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
	if input == nil || aws.StringValue(input.GroupName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

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

// AttachGroupPolicy attaches a policy to a group
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

// DetachGroupPolicy detaches a policy from a group
func (i *IAM) DetachGroupPolicy(ctx context.Context, input *iam.DetachGroupPolicyInput) (*iam.DetachGroupPolicyOutput, error) {
	if input == nil || aws.StringValue(input.GroupName) == "" || aws.StringValue(input.PolicyArn) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	output, err := i.Service.DetachGroupPolicyWithContext(ctx, input)
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
			case iam.ErrCodeInvalidInputException:
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

// ListGroupPolicies lists the policies attached to a group
func (i *IAM) ListGroupPolicies(ctx context.Context, input *iam.ListAttachedGroupPoliciesInput) ([]*iam.AttachedPolicy, error) {
	policies := []*iam.AttachedPolicy{}

	if input == nil || aws.StringValue(input.GroupName) == "" {
		return policies, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing policies attached to group %s", aws.StringValue(input.GroupName))

	truncated := true
	for truncated {
		output, err := i.Service.ListAttachedGroupPoliciesWithContext(ctx, input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				// * ErrCodeNoSuchEntityException "NoSuchEntity"
				// The request was rejected because it referenced a resource entity that does
				// not exist. The error message describes the resource.
				case iam.ErrCodeNoSuchEntityException:
					msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
					return policies, apierror.New(apierror.ErrNotFound, msg, err)
				// * ErrCodeInvalidInputException "InvalidInput"
				// The request was rejected because an invalid or out-of-range value was supplied
				// for an input parameter.
				case iam.ErrCodeInvalidInputException:
					msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
					return policies, apierror.New(apierror.ErrBadRequest, msg, err)
				// * ErrCodeServiceFailureException "ServiceFailure"
				// The request processing has failed because of an unknown error, exception
				// or failure.
				case iam.ErrCodeServiceFailureException:
					msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
					return policies, apierror.New(apierror.ErrServiceUnavailable, msg, err)
				default:
					return policies, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
				}
			}
			return policies, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
		}
		truncated = aws.BoolValue(output.IsTruncated)
		policies = append(policies, output.AttachedPolicies...)
		input.Marker = output.Marker
	}

	return policies, nil
}

// ListGroupUsers lists the users that belong to a group
func (i *IAM) ListGroupUsers(ctx context.Context, input *iam.GetGroupInput) ([]*iam.User, error) {
	users := []*iam.User{}
	if input == nil || aws.StringValue(input.GroupName) == "" {
		return users, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing iam users for group %s", aws.StringValue(input.GroupName))

	truncated := true
	for truncated {
		output, err := i.Service.GetGroupWithContext(ctx, input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				// * ErrCodeNoSuchEntityException "NoSuchEntity"
				// The request was rejected because it referenced a resource entity that does
				// not exist. The error message describes the resource.
				case iam.ErrCodeNoSuchEntityException:
					msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
					return users, apierror.New(apierror.ErrNotFound, msg, err)
				// * ErrCodeServiceFailureException "ServiceFailure"
				// The request processing has failed because of an unknown error, exception
				// or failure.
				case iam.ErrCodeServiceFailureException:
					msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
					return users, apierror.New(apierror.ErrServiceUnavailable, msg, err)
				default:
					return users, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
				}
			}
			return users, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
		}
		truncated = aws.BoolValue(output.IsTruncated)
		users = append(users, output.Users...)
		input.Marker = output.Marker
	}

	return users, nil
}
