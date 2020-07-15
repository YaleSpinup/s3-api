package iam

import (
	"context"
	"fmt"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// CreateUser creates an IAM user
func (i *IAM) CreateUser(ctx context.Context, input *iam.CreateUserInput) (*iam.CreateUserOutput, error) {
	if input == nil || aws.StringValue(input.UserName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating iam user: %s", aws.StringValue(input.UserName))
	output, err := i.Service.CreateUserWithContext(ctx, input)

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
			// * ErrCodeInvalidInputException "InvalidInput"
			// The request was rejected because an invalid or out-of-range value was supplied
			// for an input parameter.
			case iam.ErrCodeInvalidInputException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrBadRequest, msg, err)
			// * ErrCodeConcurrentModificationException "ConcurrentModification"
			// The request was rejected because multiple requests to change this object
			// were submitted simultaneously. Wait a few minutes and submit your request
			// again.
			case iam.ErrCodeConcurrentModificationException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrConflict, msg, err)
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

func (i *IAM) DeleteUser(ctx context.Context, input *iam.DeleteUserInput) (*iam.DeleteUserOutput, error) {
	if input == nil || aws.StringValue(input.UserName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting iam user %s", aws.StringValue(input.UserName))

	output, err := i.Service.DeleteUserWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeNoSuchEntityException "NoSuchEntity"
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
			// * ErrCodeDeleteConflictException "DeleteConflict"
			// The request was rejected because it attempted to delete a resource that has
			// attached subordinate entities. The error message describes these entities.
			case iam.ErrCodeDeleteConflictException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrConflict, msg, err)
			// * ErrCodeConcurrentModificationException "ConcurrentModification"
			// The request was rejected because multiple requests to change this object
			// were submitted simultaneously. Wait a few minutes and submit your request
			// again.
			case iam.ErrCodeConcurrentModificationException:
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

// GetUser gets the details for an IAM user
func (i *IAM) GetUser(ctx context.Context, input *iam.GetUserInput) (*iam.GetUserOutput, error) {
	if input == nil || aws.StringValue(input.UserName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting information about iam user %s", aws.StringValue(input.UserName))

	output, err := i.Service.GetUser(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeNoSuchEntityException "NoSuchEntity"
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
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

// CreateAccessKey creates an access key for an IAM user
func (i *IAM) CreateAccessKey(ctx context.Context, input *iam.CreateAccessKeyInput) (*iam.CreateAccessKeyOutput, error) {
	if input == nil || aws.StringValue(input.UserName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating access key for iam user %s", aws.StringValue(input.UserName))

	output, err := i.Service.CreateAccessKeyWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeNoSuchEntityException "NoSuchEntity"
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
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

// DeleteAccessKey deletes a users access key
func (i *IAM) DeleteAccessKey(ctx context.Context, input *iam.DeleteAccessKeyInput) (*iam.DeleteAccessKeyOutput, error) {
	if input == nil || aws.StringValue(input.UserName) == "" || aws.StringValue(input.AccessKeyId) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting access key id %s for iam user %s", aws.StringValue(input.AccessKeyId), aws.StringValue(input.UserName))

	output, err := i.Service.DeleteAccessKeyWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeNoSuchEntityException "NoSuchEntity"
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
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

// ListAccessKeys lists the access keys for a user
func (i *IAM) ListAccessKeys(ctx context.Context, input *iam.ListAccessKeysInput) ([]*iam.AccessKeyMetadata, error) {
	keys := []*iam.AccessKeyMetadata{}

	if input == nil || aws.StringValue(input.UserName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing access keys for iam user %s", aws.StringValue(input.UserName))

	truncated := true
	for truncated {
		output, err := i.Service.ListAccessKeysWithContext(ctx, input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				// * ErrCodeNoSuchEntityException "NoSuchEntity"
				// The request was rejected because it referenced a resource entity that does
				// not exist. The error message describes the resource.
				case iam.ErrCodeNoSuchEntityException:
					msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
					return nil, apierror.New(apierror.ErrNotFound, msg, err)
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
		truncated = aws.BoolValue(output.IsTruncated)
		keys = append(keys, output.AccessKeyMetadata...)
		input.Marker = output.Marker
	}

	return keys, nil
}

// AddUserToGroup adds the existing user to an existing group
func (i *IAM) AddUserToGroup(ctx context.Context, input *iam.AddUserToGroupInput) (*iam.AddUserToGroupOutput, error) {
	if input == nil || aws.StringValue(input.UserName) == "" || aws.StringValue(input.GroupName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("adding user %s to group %s", aws.StringValue(input.UserName), aws.StringValue(input.GroupName))

	output, err := i.Service.AddUserToGroupWithContext(ctx, input)
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

// RemoveUserFromGroup removes an existing user from a group
func (i *IAM) RemoveUserFromGroup(ctx context.Context, input *iam.RemoveUserFromGroupInput) error {
	if input == nil || aws.StringValue(input.UserName) == "" || aws.StringValue(input.GroupName) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("removing user %s from group %s", aws.StringValue(input.UserName), aws.StringValue(input.GroupName))

	_, err := i.Service.RemoveUserFromGroupWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeNoSuchEntityException "NoSuchEntity"
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return apierror.New(apierror.ErrNotFound, msg, err)
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeServiceFailureException "ServiceFailure"
			// The request processing has failed because of an unknown error, exception
			// or failure.
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return apierror.New(apierror.ErrServiceUnavailable, msg, err)
			default:
				return apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}
		return apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return nil
}

// ListUserGroups returns a list of groups that a user belongs to
func (i *IAM) ListUserGroups(ctx context.Context, input *iam.ListGroupsForUserInput) ([]*iam.Group, error) {
	groups := []*iam.Group{}

	if input == nil || aws.StringValue(input.UserName) == "" {
		return groups, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing groups for user %s", aws.StringValue(input.UserName))

	truncated := true
	for truncated {
		output, err := i.Service.ListGroupsForUserWithContext(ctx, input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				// * ErrCodeNoSuchEntityException "NoSuchEntity"
				// The request was rejected because it referenced a resource entity that does
				// not exist. The error message describes the resource.
				case iam.ErrCodeNoSuchEntityException:
					msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
					return groups, apierror.New(apierror.ErrNotFound, msg, err)
				default:
					return groups, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
				}
			}
			return groups, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
		}
		truncated = aws.BoolValue(output.IsTruncated)
		groups = append(groups, output.Groups...)
		input.Marker = output.Marker
	}

	return groups, nil
}

// ListUserPolicies lists the attached policies for a user
func (i *IAM) ListUserPolicies(ctx context.Context, input *iam.ListAttachedUserPoliciesInput) ([]*iam.AttachedPolicy, error) {
	policies := []*iam.AttachedPolicy{}

	if input == nil || aws.StringValue(input.UserName) == "" {
		return policies, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing policies for user %s", aws.StringValue(input.UserName))

	truncated := true
	for truncated {
		output, err := i.Service.ListAttachedUserPoliciesWithContext(ctx, input)
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

// DetachUserPolicy removes an IAM policy from a user
func (i *IAM) DetachUserPolicy(ctx context.Context, input *iam.DetachUserPolicyInput) error {
	if input == nil || aws.StringValue(input.UserName) == "" || aws.StringValue(input.PolicyArn) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("detaching policy %s for user %s", aws.StringValue(input.PolicyArn), aws.StringValue(input.UserName))

	_, err := i.Service.DetachUserPolicyWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeNoSuchEntityException "NoSuchEntity"
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return apierror.New(apierror.ErrNotFound, msg, err)
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeInvalidInputException "InvalidInput"
			// The request was rejected because an invalid or out-of-range value was supplied
			// for an input parameter.
			case iam.ErrCodeInvalidInputException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return apierror.New(apierror.ErrBadRequest, msg, err)
			// * ErrCodeServiceFailureException "ServiceFailure"
			// The request processing has failed because of an unknown error, exception
			// or failure.
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return apierror.New(apierror.ErrServiceUnavailable, msg, err)
			default:
				return apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}
		return apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return nil
}
