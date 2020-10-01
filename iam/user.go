package iam

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
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
		return nil, ErrCode("failed to create iam user", err)
	}

	log.Debugf("output creating user: %s", awsutil.Prettify(output))

	return output, nil
}

func (i *IAM) DeleteUser(ctx context.Context, input *iam.DeleteUserInput) error {
	if input == nil || aws.StringValue(input.UserName) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting iam user %s", aws.StringValue(input.UserName))

	_, err := i.Service.DeleteUserWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to delete iam user", err)
	}

	return nil
}

// GetUser gets the details for an IAM user
func (i *IAM) GetUser(ctx context.Context, input *iam.GetUserInput) (*iam.GetUserOutput, error) {
	if input == nil || aws.StringValue(input.UserName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting information about iam user %s", aws.StringValue(input.UserName))

	output, err := i.Service.GetUser(input)
	if err != nil {
		return nil, ErrCode("failed to get iam user", err)
	}

	log.Debugf("get user output: %s", awsutil.Prettify(output))

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
		return nil, ErrCode("failed to create iam access key", err)
	}

	return output, nil
}

// DeleteAccessKey deletes a users access key
func (i *IAM) DeleteAccessKey(ctx context.Context, input *iam.DeleteAccessKeyInput) error {
	if input == nil || aws.StringValue(input.UserName) == "" || aws.StringValue(input.AccessKeyId) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting access key id %s for iam user %s", aws.StringValue(input.AccessKeyId), aws.StringValue(input.UserName))

	_, err := i.Service.DeleteAccessKeyWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to delete iam access key", err)
	}

	return nil
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
			return nil, ErrCode("failed to list iam access keys", err)
		}
		truncated = aws.BoolValue(output.IsTruncated)
		keys = append(keys, output.AccessKeyMetadata...)
		input.Marker = output.Marker
	}

	log.Debugf("list access keys output: %s", awsutil.Prettify(keys))

	return keys, nil
}

// AddUserToGroup adds the existing user to an existing group
func (i *IAM) AddUserToGroup(ctx context.Context, input *iam.AddUserToGroupInput) error {
	if input == nil || aws.StringValue(input.UserName) == "" || aws.StringValue(input.GroupName) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("adding user %s to group %s", aws.StringValue(input.UserName), aws.StringValue(input.GroupName))

	_, err := i.Service.AddUserToGroupWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to add user to group", err)
	}

	return nil
}

// RemoveUserFromGroup removes an existing user from a group
func (i *IAM) RemoveUserFromGroup(ctx context.Context, input *iam.RemoveUserFromGroupInput) error {
	if input == nil || aws.StringValue(input.UserName) == "" || aws.StringValue(input.GroupName) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("removing user %s from group %s", aws.StringValue(input.UserName), aws.StringValue(input.GroupName))

	_, err := i.Service.RemoveUserFromGroupWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to remove user from group", err)
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
			return nil, ErrCode("failed to list groups for user", err)
		}
		truncated = aws.BoolValue(output.IsTruncated)
		groups = append(groups, output.Groups...)
		input.Marker = output.Marker
	}

	log.Debugf("got list of groups for user %s: %s", aws.StringValue(input.UserName), awsutil.Prettify(groups))

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
			return nil, ErrCode("failed to list policies for user", err)
		}
		truncated = aws.BoolValue(output.IsTruncated)
		policies = append(policies, output.AttachedPolicies...)
		input.Marker = output.Marker
	}

	log.Debugf("got list of policies for user %s: %s", aws.StringValue(input.UserName), awsutil.Prettify(policies))

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
		return ErrCode("failed to detach policy for user", err)
	}

	return nil
}
