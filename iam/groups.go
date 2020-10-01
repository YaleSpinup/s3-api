package iam

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// CreateGroup handles creating an IAM group
func (i *IAM) CreateGroup(ctx context.Context, input *iam.CreateGroupInput) (*iam.Group, error) {
	if input == nil || aws.StringValue(input.GroupName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating IAM group: %s", aws.StringValue(input.GroupName))

	output, err := i.Service.CreateGroupWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create iam group", err)
	}

	log.Debugf("returning created group %s", awsutil.Prettify(output.Group))

	return output.Group, nil
}

// GetGroup gets the details of an IAM group
func (i *IAM) GetGroup(ctx context.Context, groupName string) (*iam.Group, error) {
	if groupName == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting details for group %s", groupName)

	out, err := i.Service.GetGroupWithContext(ctx, &iam.GetGroupInput{
		GroupName: aws.String(groupName),
	})

	if err != nil {
		return nil, ErrCode("failed to get details for group", err)
	}

	log.Debugf("returning details about group %s: %s", groupName, awsutil.Prettify(out.Group))

	return out.Group, nil
}

// DeleteGroup handles deleting an IAM group
func (i *IAM) DeleteGroup(ctx context.Context, input *iam.DeleteGroupInput) error {
	if input == nil || aws.StringValue(input.GroupName) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting iam group %s", aws.StringValue(input.GroupName))

	_, err := i.Service.DeleteGroupWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to delete iam group", err)
	}

	return nil
}

// AttachGroupPolicy attaches a policy to a group
func (i *IAM) AttachGroupPolicy(ctx context.Context, input *iam.AttachGroupPolicyInput) error {
	if input == nil || aws.StringValue(input.GroupName) == "" || aws.StringValue(input.PolicyArn) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("attaching policy '%s' to group '%s'", aws.StringValue(input.PolicyArn), aws.StringValue(input.GroupName))

	_, err := i.Service.AttachGroupPolicyWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to attach policy to group", err)
	}

	return nil
}

// DetachGroupPolicy detaches a policy from a group
func (i *IAM) DetachGroupPolicy(ctx context.Context, input *iam.DetachGroupPolicyInput) error {
	if input == nil || aws.StringValue(input.GroupName) == "" || aws.StringValue(input.PolicyArn) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("detatching policy '%s' from group '%s'", aws.StringValue(input.PolicyArn), aws.StringValue(input.GroupName))

	if _, err := i.Service.DetachGroupPolicyWithContext(ctx, input); err != nil {
		return ErrCode("failed to deattach policy from group", err)
	}

	return nil
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
			return policies, ErrCode("failed to list attached group policies", err)
		}
		truncated = aws.BoolValue(output.IsTruncated)
		policies = append(policies, output.AttachedPolicies...)
		input.Marker = output.Marker
	}

	log.Debugf("returning list of policies attached to group %s: %s", aws.StringValue(input.GroupName), awsutil.Prettify(policies))

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
			return users, ErrCode("failed to list users for group", err)
		}

		truncated = aws.BoolValue(output.IsTruncated)
		users = append(users, output.Users...)
		input.Marker = output.Marker
	}

	log.Debugf("returning list of users in group %s: %s", aws.StringValue(input.GroupName), awsutil.Prettify(users))

	return users, nil
}
