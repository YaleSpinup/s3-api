package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/YaleSpinup/s3-api/common"
	iamapi "github.com/YaleSpinup/s3-api/iam"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// groupSuffixToInline maps legacy group suffixes to inline policy suffixes
var groupSuffixToInline = map[string]string{
	"BktAdmGrp": "BktAdmPlc",
	"BktRWGrp":  "BktRWPlc",
	"BktROGrp":  "BktROPlc",
	"WebAdmGrp": "WebAdmPlc",
}

// knownSuffixes lists all group suffixes we want to migrate
var knownSuffixes = []string{"-BktAdmGrp", "-BktRWGrp", "-BktROGrp", "-WebAdmGrp"}

func main() {
	configFile := flag.String("config", "config/config.json", "Path to s3-api configuration file")
	dryRun := flag.Bool("dry-run", false, "Preview changes without executing them")
	targetAccount := flag.String("account", "", "Specific account name to migrate (default: all accounts)")
	verbose := flag.Bool("verbose", false, "Enable debug logging")
	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	f, err := os.Open(*configFile)
	if err != nil {
		log.Fatalf("Unable to open config file: %s", err)
	}
	defer f.Close()

	config, err := common.ReadConfig(bufio.NewReader(f))
	if err != nil {
		log.Fatalf("Unable to read config: %s", err)
	}

	if *dryRun {
		log.Info("=== DRY RUN MODE — no changes will be made ===")
	}

	ctx := context.Background()

	// Create the base AWS session with the config credentials
	baseSess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(config.Account.Akid, config.Account.Secret, ""),
		Region:      aws.String(config.Account.Region),
	}))

	for accountName, accountId := range config.AccountsMap {
		if *targetAccount != "" && accountName != *targetAccount {
			continue
		}

		log.Infof("=== Migrating account: %s (ID: %s) ===", accountName, accountId)

		// Assume the configured role into the target account (same pattern as cloudfront/server)
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, config.Account.Role)
		log.Infof("Assuming role: %s", roleArn)

		assumedSess := session.Must(session.NewSession(&aws.Config{
			Credentials: stscreds.NewCredentials(baseSess, roleArn, func(p *stscreds.AssumeRoleProvider) {
				if config.Account.ExternalId != "" {
					p.ExternalID = aws.String(config.Account.ExternalId)
				}
			}),
			Region: aws.String(config.Account.Region),
		}))

		iamService := iamapi.NewSession(assumedSess, config.Account)

		if err := migrateAccount(ctx, iamService, accountName, *dryRun); err != nil {
			log.Errorf("Error migrating account %s: %s", accountName, err)
		}
	}

	log.Info("=== Migration complete ===")
}

func migrateAccount(ctx context.Context, iamService iamapi.IAM, accountName string, dryRun bool) error {
	// List all groups in the account
	allGroups, err := iamService.ListGroups(ctx, &iam.ListGroupsInput{MaxItems: aws.Int64(1000)}, "")
	if err != nil {
		return fmt.Errorf("failed to list groups: %w", err)
	}

	// Filter to only groups matching our known suffixes
	var matchingGroups []*iam.Group
	for _, g := range allGroups {
		groupName := aws.StringValue(g.GroupName)
		if matchesKnownSuffix(groupName) {
			matchingGroups = append(matchingGroups, g)
		}
	}

	log.Infof("Found %d matching legacy groups out of %d total groups in account %s",
		len(matchingGroups), len(allGroups), accountName)

	if len(matchingGroups) == 0 {
		log.Info("Nothing to migrate.")
		return nil
	}

	var migrated, skipped, errored int
	for _, group := range matchingGroups {
		result, err := migrateGroup(ctx, iamService, group, dryRun)
		if err != nil {
			log.Errorf("Failed to migrate group %s: %s", aws.StringValue(group.GroupName), err)
			errored++
			continue
		}
		switch result {
		case "migrated":
			migrated++
		case "skipped":
			skipped++
		}
	}

	log.Infof("Account %s summary: migrated=%d, skipped=%d, errors=%d", accountName, migrated, skipped, errored)
	return nil
}

func migrateGroup(ctx context.Context, iamService iamapi.IAM, group *iam.Group, dryRun bool) (string, error) {
	groupName := aws.StringValue(group.GroupName)
	suffix := getGroupSuffix(groupName)
	inlineSuffix := groupSuffixToInline[suffix]

	if inlineSuffix == "" {
		return "skipped", fmt.Errorf("unknown suffix for group %s", groupName)
	}

	log.Infof("Processing group: %s (suffix: %s → %s)", groupName, suffix, inlineSuffix)

	// 1. Get users in the group
	users, err := iamService.ListGroupUsers(ctx, &iam.GetGroupInput{GroupName: aws.String(groupName)})
	if err != nil {
		return "", fmt.Errorf("failed to list users in group %s: %w", groupName, err)
	}

	// 2. Get attached managed policies for this group
	policies, err := iamService.ListGroupPolicies(ctx, &iam.ListAttachedGroupPoliciesInput{GroupName: aws.String(groupName)})
	if err != nil {
		return "", fmt.Errorf("failed to list policies for group %s: %w", groupName, err)
	}

	if len(policies) == 0 {
		log.Warnf("Group %s has no attached policies — skipping", groupName)
		return "skipped", nil
	}

	if len(users) == 0 {
		log.Infof("Group %s has no users — cleaning up empty group", groupName)
	}

	// 3. Get the policy document from the first managed policy
	policyArn := aws.StringValue(policies[0].PolicyArn)
	policyDoc, err := getPolicyDocument(ctx, iamService, policyArn)
	if err != nil {
		return "", fmt.Errorf("failed to get policy document for group %s: %w", groupName, err)
	}

	// Derive inline policy name: replace group suffix with inline suffix
	// e.g., "mybucket-BktAdmGrp" → "mybucket-BktAdmPlc"
	inlinePolicyName := groupName[:len(groupName)-len(suffix)] + inlineSuffix

	log.Debugf("Policy document for %s (%d bytes): %s", groupName, len(policyDoc), policyDoc)

	// 4. For each user in the group, attach the inline policy
	for _, user := range users {
		userName := aws.StringValue(user.UserName)

		if dryRun {
			log.Infof("[DRY RUN] Would attach inline policy '%s' to user '%s' (from group %s)",
				inlinePolicyName, userName, groupName)
			log.Infof("[DRY RUN] Would remove user '%s' from group '%s'", userName, groupName)
			continue
		}

		// Check if inline policy already exists on this user
		existingPolicies, err := iamService.ListUserInlinePolicies(ctx, &iam.ListUserPoliciesInput{
			UserName: aws.String(userName),
		})
		if err != nil {
			log.Warnf("Failed to list inline policies for user %s: %s", userName, err)
		}

		alreadyExists := false
		for _, p := range existingPolicies {
			if aws.StringValue(p) == inlinePolicyName {
				alreadyExists = true
				break
			}
		}

		if alreadyExists {
			log.Infof("Inline policy '%s' already exists on user '%s' — skipping attachment", inlinePolicyName, userName)
		} else {
			// Attach inline policy
			if err := iamService.PutUserPolicy(ctx, &iam.PutUserPolicyInput{
				UserName:       aws.String(userName),
				PolicyName:     aws.String(inlinePolicyName),
				PolicyDocument: aws.String(policyDoc),
			}); err != nil {
				return "", fmt.Errorf("failed to put inline policy on user %s: %w", userName, err)
			}
			log.Infof("Attached inline policy '%s' to user '%s'", inlinePolicyName, userName)
		}

		// Remove user from group
		if err := iamService.RemoveUserFromGroup(ctx, &iam.RemoveUserFromGroupInput{
			UserName:  aws.String(userName),
			GroupName: aws.String(groupName),
		}); err != nil {
			return "", fmt.Errorf("failed to remove user %s from group %s: %w", userName, groupName, err)
		}
		log.Infof("Removed user '%s' from group '%s'", userName, groupName)
	}

	// 5. Detach and delete managed policies from the group, then delete the group
	if dryRun {
		for _, p := range policies {
			log.Infof("[DRY RUN] Would detach policy '%s' from group '%s'", aws.StringValue(p.PolicyArn), groupName)
			log.Infof("[DRY RUN] Would delete managed policy '%s'", aws.StringValue(p.PolicyArn))
		}
		log.Infof("[DRY RUN] Would delete group '%s'", groupName)
		return "migrated", nil
	}

	for _, p := range policies {
		if err := iamService.DetachGroupPolicy(ctx, &iam.DetachGroupPolicyInput{
			GroupName: aws.String(groupName),
			PolicyArn: p.PolicyArn,
		}); err != nil {
			log.Warnf("Failed to detach policy %s from group %s: %s", aws.StringValue(p.PolicyArn), groupName, err)
			continue
		}
		log.Infof("Detached policy '%s' from group '%s'", aws.StringValue(p.PolicyArn), groupName)

		if err := iamService.DeletePolicy(ctx, &iam.DeletePolicyInput{PolicyArn: p.PolicyArn}); err != nil {
			log.Warnf("Failed to delete policy %s: %s", aws.StringValue(p.PolicyArn), err)
		} else {
			log.Infof("Deleted managed policy '%s'", aws.StringValue(p.PolicyArn))
		}
	}

	if err := iamService.DeleteGroup(ctx, &iam.DeleteGroupInput{GroupName: aws.String(groupName)}); err != nil {
		log.Warnf("Failed to delete group %s: %s", groupName, err)
	} else {
		log.Infof("Deleted group '%s'", groupName)
	}

	return "migrated", nil
}

func matchesKnownSuffix(groupName string) bool {
	for _, suffix := range knownSuffixes {
		if strings.HasSuffix(groupName, suffix) {
			return true
		}
	}
	return false
}

func getGroupSuffix(groupName string) string {
	for _, suffix := range knownSuffixes {
		if strings.HasSuffix(groupName, suffix) {
			return strings.TrimPrefix(suffix, "-")
		}
	}
	return ""
}

// getPolicyDocument retrieves the policy document text from a managed policy ARN
func getPolicyDocument(ctx context.Context, iamService iamapi.IAM, policyArn string) (string, error) {
	policyDetail, err := iamService.Service.GetPolicyWithContext(ctx, &iam.GetPolicyInput{
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get policy %s: %w", policyArn, err)
	}

	versionOutput, err := iamService.Service.GetPolicyVersionWithContext(ctx, &iam.GetPolicyVersionInput{
		PolicyArn: aws.String(policyArn),
		VersionId: policyDetail.Policy.DefaultVersionId,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get policy version for %s: %w", policyArn, err)
	}

	// AWS returns URL-encoded policy documents
	doc, err := url.QueryUnescape(aws.StringValue(versionOutput.PolicyVersion.Document))
	if err != nil {
		return "", fmt.Errorf("failed to decode policy document for %s: %w", policyArn, err)
	}

	return doc, nil
}

