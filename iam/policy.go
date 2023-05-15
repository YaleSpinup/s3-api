package iam

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// CreatePolicy handles creating IAM policy
func (i *IAM) CreatePolicy(ctx context.Context, input *iam.CreatePolicyInput) (*iam.Policy, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating iam policy: %s", *input.PolicyName)

	output, err := i.Service.CreatePolicyWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create iam policy", err)
	}

	log.Debugf("returning created iam policy %s", awsutil.Prettify(output.Policy))

	return output.Policy, nil
}

// DeletePolicy handles deleting IAM policy
func (i *IAM) DeletePolicy(ctx context.Context, input *iam.DeletePolicyInput) error {
	if input == nil || aws.StringValue(input.PolicyArn) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting iam policy %s", aws.StringValue(input.PolicyArn))

	_, err := i.Service.DeletePolicyWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to create iam policy", err)
	}

	log.Debugf("deleted iam policy %s", aws.StringValue(input.PolicyArn))

	return nil
}

// ListPolicies lists all policies for an account
func (i *IAM) ListPolicies(ctx context.Context, input *iam.ListPoliciesInput) ([]*iam.Policy, error) {
	policies := []*iam.Policy{}
	if input == nil {
		return policies, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Info("listing iam policies")

	truncated := true
	for truncated {
		output, err := i.Service.ListPoliciesWithContext(ctx, input)
		if err != nil {
			return nil, ErrCode("failed to list iam policy", err)
		}
		truncated = aws.BoolValue(output.IsTruncated)
		policies = append(policies, output.Policies...)
		input.Marker = output.Marker
	}

	log.Debugf("returning list of iam policies: %s", awsutil.Prettify(policies))

	return policies, nil
}
