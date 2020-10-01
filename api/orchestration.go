package api

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// CreateBucketGroupPolicy expects an acount, bucket name and the group name (without the bucket prefix).  It verifies the group
// is one of our supported types and then generates a policy doc for the group and bucket.  Finally, it creates the group
// and attaches the policy.  It returns a rollback function and will rollback itself if it encounters an error.
func (s *server) CreateBucketGroupPolicy(ctx context.Context, account, bucket, group string) ([]rollbackFunc, error) {
	var err error
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	i, ok := s.iamServices[account]
	if !ok {
		return rollBackTasks, fmt.Errorf("IAM service not found for account: %s", account)
	}

	var policyName, policyDescription string
	var policyDocument []byte
	// TODO: add website groups
	switch group {
	case "BktAdmGrp":
		policyName = fmt.Sprintf("%s-BktAdmPlc", bucket)
		policyDescription = fmt.Sprintf("Admin policy for %s bucket", bucket)
		if policyDocument, err = i.AdminBucketPolicy(bucket); err != nil {
			return rollBackTasks, err
		}
	case "BktRWGrp":
		policyName = fmt.Sprintf("%s-BktRWPlc", bucket)
		policyDescription = fmt.Sprintf("Read-Write policy for %s bucket", bucket)
		if policyDocument, err = i.ReadWriteBucketPolicy(bucket); err != nil {
			return rollBackTasks, err
		}
	case "BktROGrp":
		policyName = fmt.Sprintf("%s-BktROPlc", bucket)
		policyDescription = fmt.Sprintf("Read-Only policy for %s bucket", bucket)
		if policyDocument, err = i.ReadOnlyBucketPolicy(bucket); err != nil {
			return rollBackTasks, err
		}
	default:
		return rollBackTasks, fmt.Errorf("invalid group name: %s", group)
	}

	var policyOutput *iam.Policy
	if policyOutput, err = i.CreatePolicy(ctx, &iam.CreatePolicyInput{
		Description:    aws.String(policyDescription),
		PolicyDocument: aws.String(string(policyDocument)),
		PolicyName:     aws.String(policyName),
	}); err != nil {
		return rollBackTasks, fmt.Errorf("failed to create iam policy for bucket %s: %s", bucket, err)
	}

	// append policy delete to rollback tasks
	rbfunc := func(ctx context.Context) error {
		if err := i.DeletePolicy(ctx, &iam.DeletePolicyInput{PolicyArn: policyOutput.Arn}); err != nil {
			return err
		}
		return nil
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	groupName := fmt.Sprintf("%s-%s", bucket, group)

	if _, err = i.CreateGroup(ctx, &iam.CreateGroupInput{
		GroupName: aws.String(groupName),
	}); err != nil {
		return rollBackTasks, fmt.Errorf("failed to create group %s: %s", groupName, err)
	}

	// append group delete to rollback tasks
	rbfunc = func(ctx context.Context) error {
		if err := i.DeleteGroup(ctx, &iam.DeleteGroupInput{GroupName: aws.String(groupName)}); err != nil {
			return err
		}
		return nil
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	if err = i.AttachGroupPolicy(ctx, &iam.AttachGroupPolicyInput{
		GroupName: aws.String(groupName),
		PolicyArn: policyOutput.Arn,
	}); err != nil {
		return rollBackTasks, fmt.Errorf("failed to attach policy %s to group %s", aws.StringValue(policyOutput.Arn), groupName)
	}

	return rollBackTasks, nil
}
