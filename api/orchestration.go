package api

import (
	"context"
	"fmt"

	iamapi "github.com/YaleSpinup/s3-api/iam"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// CreateBucketGroupPolicy expects an acount, bucket name and the group name (without the bucket prefix).  It verifies the group
// is one of our supported types and then generates a policy doc for the group and bucket.  Finally, it creates the group
// and attaches the policy.  It returns a rollback function and will rollback itself if it encounters an error.
func (s *server) CreateBucketGroupPolicy(ctx context.Context, iamService iamapi.IAM, bucket, group string) ([]rollbackFunc, error) {
	var err error
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	var policyName, policyDescription string
	var policyDocument []byte
	// TODO: add website groups
	switch group {
	case "BktAdmGrp":
		policyName = fmt.Sprintf("%s-BktAdmPlc", bucket)
		policyDescription = fmt.Sprintf("Admin policy for %s bucket", bucket)
		if policyDocument, err = iamService.AdminBucketPolicy(bucket); err != nil {
			return rollBackTasks, err
		}
	case "BktRWGrp":
		policyName = fmt.Sprintf("%s-BktRWPlc", bucket)
		policyDescription = fmt.Sprintf("Read-Write policy for %s bucket", bucket)
		if policyDocument, err = iamService.ReadWriteBucketPolicy(bucket); err != nil {
			return rollBackTasks, err
		}
	case "BktROGrp":
		policyName = fmt.Sprintf("%s-BktROPlc", bucket)
		policyDescription = fmt.Sprintf("Read-Only policy for %s bucket", bucket)
		if policyDocument, err = iamService.ReadOnlyBucketPolicy(bucket); err != nil {
			return rollBackTasks, err
		}
	default:
		return rollBackTasks, fmt.Errorf("invalid group name: %s", group)
	}

	var policyOutput *iam.Policy
	if policyOutput, err = iamService.CreatePolicy(ctx, &iam.CreatePolicyInput{
		Description:    aws.String(policyDescription),
		PolicyDocument: aws.String(string(policyDocument)),
		PolicyName:     aws.String(policyName),
	}); err != nil {
		return rollBackTasks, fmt.Errorf("failed to create iam policy for bucket %s: %s", bucket, err)
	}

	// append policy delete to rollback tasks
	rbfunc := func(ctx context.Context) error {
		if err := iamService.DeletePolicy(ctx, &iam.DeletePolicyInput{PolicyArn: policyOutput.Arn}); err != nil {
			return err
		}
		return nil
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	groupName := fmt.Sprintf("%s-%s", bucket, group)

	if _, err = iamService.CreateGroup(ctx, &iam.CreateGroupInput{
		GroupName: aws.String(groupName),
	}); err != nil {
		return rollBackTasks, fmt.Errorf("failed to create group %s: %s", groupName, err)
	}

	// append group delete to rollback tasks
	rbfunc = func(ctx context.Context) error {
		if err := iamService.DeleteGroup(ctx, &iam.DeleteGroupInput{GroupName: aws.String(groupName)}); err != nil {
			return err
		}
		return nil
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	if err = iamService.AttachGroupPolicy(ctx, &iam.AttachGroupPolicyInput{
		GroupName: aws.String(groupName),
		PolicyArn: policyOutput.Arn,
	}); err != nil {
		return rollBackTasks, fmt.Errorf("failed to attach policy %s to group %s", aws.StringValue(policyOutput.Arn), groupName)
	}

	return rollBackTasks, nil
}

// CreateWebsiteBucketPolicy expects an acount, bucket name and the group name (without the bucket prefix).  It verifies the group
// is one of our supported types and then generates a policy doc for the group and bucket.  Finally, it creates the group
// and attaches the policy.  It returns a rollback function and will rollback itself if it encounters an error.
func (s *server) CreateWebsiteBucketPolicy(ctx context.Context, iamService iamapi.IAM, website, path string, group string) ([]rollbackFunc, error) {
	var err error
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	var policyName, policyDescription string
	var policyDocument []byte
	// TODO: add website groups
	switch group {
	case "BktAdmGrp":
		policyName = iamapi.FormatGroupName(website, path, "BktAdmPlc")
		policyDescription = fmt.Sprintf("Admin policy for %s website", website)
		if path != "/" {
			if policyDocument, err = iamService.AdminBucketPolicyWithPath(website, path); err != nil {
				return rollBackTasks, err
			}
		} else {
			if policyDocument, err = iamService.AdminBucketPolicy(website); err != nil {
				return rollBackTasks, err
			}
		}
	case "BktRWGrp":
		policyName = iamapi.FormatGroupName(website, path, "BktRWPlc")
		policyDescription = fmt.Sprintf("Read-Write policy for %s website", website)
		if path != "/" {
			if policyDocument, err = iamService.ReadWriteBucketPolicyWithPath(website, path); err != nil {
				return rollBackTasks, err
			}
		} else {
			if policyDocument, err = iamService.ReadWriteBucketPolicy(website); err != nil {
				return rollBackTasks, err
			}
		}
	case "BktROGrp":
		policyName = iamapi.FormatGroupName(website, path, "BktROPlc")
		policyDescription = fmt.Sprintf("Read-Only policy for %s website", website)

		if path != "/" {
			if policyDocument, err = iamService.ReadOnlyBucketPolicyWithPath(website, path); err != nil {
				return rollBackTasks, err
			}
		} else {
			if policyDocument, err = iamService.ReadOnlyBucketPolicy(website); err != nil {
				return rollBackTasks, err
			}
		}
	default:
		return rollBackTasks, fmt.Errorf("invalid group name: %s", group)
	}

	var policyOutput *iam.Policy
	if policyOutput, err = iamService.CreatePolicy(ctx, &iam.CreatePolicyInput{
		Description:    aws.String(policyDescription),
		PolicyDocument: aws.String(string(policyDocument)),
		PolicyName:     aws.String(policyName),
		Path:           aws.String(path),
	}); err != nil {
		return rollBackTasks, fmt.Errorf("failed to create iam policy for website %s: %s", website, err)
	}

	// append policy delete to rollback tasks
	rbfunc := func(ctx context.Context) error {
		if err := iamService.DeletePolicy(ctx, &iam.DeletePolicyInput{PolicyArn: policyOutput.Arn}); err != nil {
			return err
		}
		return nil
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	groupName := iamapi.FormatGroupName(website, path, group)

	if _, err = iamService.CreateGroup(ctx, &iam.CreateGroupInput{
		GroupName: aws.String(groupName),
	}); err != nil {
		return rollBackTasks, fmt.Errorf("failed to create group %s: %s", groupName, err)
	}

	// append group delete to rollback tasks
	rbfunc = func(ctx context.Context) error {
		if err := iamService.DeleteGroup(ctx, &iam.DeleteGroupInput{GroupName: aws.String(groupName)}); err != nil {
			return err
		}
		return nil
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	if err = iamService.AttachGroupPolicy(ctx, &iam.AttachGroupPolicyInput{
		GroupName: aws.String(groupName),
		PolicyArn: policyOutput.Arn,
	}); err != nil {
		return rollBackTasks, fmt.Errorf("failed to attach policy %s to group %s", aws.StringValue(policyOutput.Arn), groupName)
	}

	return rollBackTasks, nil
}
