package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// CreateWebsiteHandler orchestrates the creation of a new s3 bucket website with rollback in
// the event of failure.  The operations are:
// 1. create the bucket with the given name
// 2. tag the bucket with given tags
// 3. apply the website configuration to the bucket
// 4. generate the default admin bucket policy
// 5. create the admin bucket policy
// 6. create the bucket admin group, '<bucketName>-BktAdmGrp'
// 7. attach the bucket admin policy to the bucket admin group
// 8. create cloudfront distribution with s3 website origin (for https)
// 9. create the web admin group, '<bucketName>-WebAdmGrp'
// 10. attach the web admin policy to the web admin group
// 11. create alias record in route53
// Note: this does _not_ create any users for managing the bucket
func (s *server) CreateWebsiteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	s3Service, ok := s.s3Services[account]
	if !ok {
		msg := fmt.Sprintf("s3 service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	iamService, ok := s.iamServices[account]
	if !ok {
		msg := fmt.Sprintf("IAM service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	cloudFrontService, ok := s.cloudFrontServices[account]
	if !ok {
		msg := fmt.Sprintf("CloudFront service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	route53Service, ok := s.route53Services[account]
	if !ok {
		msg := fmt.Sprintf("Route53 service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	var req struct {
		Tags                 []*s3.Tag
		BucketInput          s3.CreateBucketInput
		WebsiteConfiguration s3.WebsiteConfiguration
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		msg := fmt.Sprintf("cannot decode body into create website input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// append org tag that will get applied to all resources that tag
	req.Tags = append(req.Tags, &s3.Tag{
		Key:   aws.String("spinup:org"),
		Value: aws.String(Org),
	})

	// setup err var, rollback function list and defer execution
	var err error
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	bucketName := aws.StringValue(req.BucketInput.Bucket)

	var domain *common.Domain
	if domain, err = cloudFrontService.WebsiteDomain(bucketName); err != nil {
		msg := fmt.Sprintf("failed to validate website domain %s", bucketName)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	var bucketOutput *s3.CreateBucketOutput
	if bucketOutput, err = s3Service.CreateBucket(r.Context(), &req.BucketInput); err != nil {
		msg := fmt.Sprintf("failed to create bucket %s", bucketName)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// Update public access for s3 website bucket
	if _, err = s3Service.SetPublicAccessBlock(r.Context(), &s3.PutPublicAccessBlockInput{
		Bucket:                         aws.String(bucketName),
		PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{BlockPublicPolicy: aws.Bool(false)},
	}); err != nil {
		msg := fmt.Sprintf("failed to set bucket access to public for %s", bucketName)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append bucket delete to rollback tasks
	rbfunc := func(ctx context.Context) error {
		err := s3Service.DeleteEmptyBucket(r.Context(), &s3.DeleteBucketInput{Bucket: aws.String(bucketName)})
		return err
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	// wait for the bucket to exist
	if err = retry(3, 2*time.Second, func() error {
		log.Infof("checking if bucket exists before continuing: %s", bucketName)
		exists, err := s3Service.BucketExists(r.Context(), bucketName)
		if err != nil {
			return err
		}

		if exists {
			log.Infof("bucket %s exists", bucketName)
			return nil
		}

		msg := fmt.Sprintf("s3 bucket (%s) doesn't exist", bucketName)
		return errors.New(msg)
	}); err != nil {
		handleError(w, err)
		return
	}

	// retry tagging
	if err = retry(3, 2*time.Second, func() error {
		if err := s3Service.TagBucket(r.Context(), bucketName, req.Tags); err != nil {
			log.Warnf("error tagging website bucket %s: %s", bucketName, err)
			return err
		}
		return nil
	}); err != nil {
		msg := fmt.Sprintf("failed to tag website bucket %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// enable AWS managed serverside encryption for the website/bucket
	if err = s3Service.UpdateBucketEncryption(r.Context(), &s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucketName),
		ServerSideEncryptionConfiguration: &s3.ServerSideEncryptionConfiguration{
			Rules: []*s3.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &s3.ServerSideEncryptionByDefault{
						SSEAlgorithm: aws.String("AES256"),
					},
				},
			},
		},
	}); err != nil {
		msg := fmt.Sprintf("failed to enable encryption for bucket %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// enable logging access for the website/bucket to a central repo
	if s3Service.LoggingBucket != "" {
		if err = s3Service.UpdateBucketLogging(r.Context(), bucketName, s3Service.LoggingBucket, s3Service.LoggingBucketPrefix); err != nil {
			msg := fmt.Sprintf("failed to enable logging for bucket %s: %s", bucketName, err.Error())
			handleError(w, errors.Wrap(err, msg))
			return
		}
	}

	if err = s3Service.UpdateWebsiteConfig(r.Context(), &s3.PutBucketWebsiteInput{
		Bucket:               aws.String(bucketName),
		WebsiteConfiguration: &req.WebsiteConfiguration,
	}); err != nil {
		msg := fmt.Sprintf("failed to configure bucket %s as website: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	var defaultWebsitePolicy []byte
	if defaultWebsitePolicy, err = iamService.DefaultWebsiteAccessPolicy(aws.String(bucketName)); err != nil {
		msg := fmt.Sprintf("failed building default website bucket access policy for %s: %s", bucketName, err.Error())
		handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
		return
	}

	if err = s3Service.UpdateBucketPolicy(r.Context(), &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucketName),
		Policy: aws.String(string(defaultWebsitePolicy)),
	}); err != nil {
		handleError(w, err)
		return
	}

	// build the default IAM bucket admin policy (from the config and known inputs)
	var defaultBktPolicy []byte
	if defaultBktPolicy, err = iamService.DefaultBucketAdminPolicy(aws.String(bucketName)); err != nil {
		msg := fmt.Sprintf("failed building default IAM policy for bucket %s: %s", bucketName, err.Error())
		handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
		return
	}

	var bktPolicy *iam.Policy
	if bktPolicy, err = iamService.CreatePolicy(r.Context(), &iam.CreatePolicyInput{
		Description:    aws.String(fmt.Sprintf("Admin policy for %s bucket", bucketName)),
		PolicyDocument: aws.String(string(defaultBktPolicy)),
		PolicyName:     aws.String(fmt.Sprintf("%s-BktAdmPlc", bucketName)),
	}); err != nil {
		msg := fmt.Sprintf("failed to create bucket admin policy: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append policy delete to rollback tasks
	rbfunc = func(ctx context.Context) error {
		err := iamService.DeletePolicy(r.Context(), &iam.DeletePolicyInput{PolicyArn: bktPolicy.Arn})
		return err
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	bktGroupName := fmt.Sprintf("%s-BktAdmGrp", bucketName)

	var bktGroup *iam.Group
	if bktGroup, err = iamService.CreateGroup(r.Context(), &iam.CreateGroupInput{
		GroupName: aws.String(bktGroupName),
	}); err != nil {
		msg := fmt.Sprintf("failed to create bucket admin group: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append group delete to rollback tasks
	rbfunc = func(ctx context.Context) error {
		return iamService.DeleteGroup(r.Context(), &iam.DeleteGroupInput{GroupName: aws.String(bktGroupName)})
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	if err = iamService.AttachGroupPolicy(r.Context(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String(bktGroupName),
		PolicyArn: bktPolicy.Arn,
	}); err != nil {
		msg := fmt.Sprintf("failed to attach policy %s to group %s: %s", aws.StringValue(bktPolicy.Arn), bktGroupName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append detach group policy to rollback tasks
	rbfunc = func(ctx context.Context) error {
		return iamService.DetachGroupPolicy(r.Context(), &iam.DetachGroupPolicyInput{
			GroupName: aws.String(bktGroupName),
			PolicyArn: bktPolicy.Arn,
		})
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	// normalize tags
	cfTags := []*cloudfront.Tag{}
	for _, tag := range req.Tags {
		t := &cloudfront.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		}
		cfTags = append(cfTags, t)
	}

	var defaultWebsiteDistribution *cloudfront.DistributionConfig
	if defaultWebsiteDistribution, err = cloudFrontService.DefaultWebsiteDistributionConfig(bucketName); err != nil {
		msg := fmt.Sprintf("failed to generate default website distribution config for %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	var distribution *cloudfront.Distribution
	if distribution, err = cloudFrontService.CreateDistribution(r.Context(), defaultWebsiteDistribution, &cloudfront.Tags{Items: cfTags}); err != nil {
		msg := fmt.Sprintf("failed to create cloudfront distribution for website %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append disable cloudfront distribution to rollback tasks
	rbfunc = func(ctx context.Context) error {
		_, err := cloudFrontService.DisableDistribution(r.Context(), aws.StringValue(distribution.Id))
		return err
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	// build the default IAM web admin policy (from the config and known inputs)
	var defaultWebPolicy []byte
	if defaultWebPolicy, err = iamService.DefaultWebAdminPolicy(distribution.ARN); err != nil {
		msg := fmt.Sprintf("failed building default IAM policy for cloudfront distribution %s: %s", aws.StringValue(distribution.ARN), err.Error())
		handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
		return
	}

	var webPolicy *iam.Policy
	if webPolicy, err = iamService.CreatePolicy(r.Context(), &iam.CreatePolicyInput{
		Description:    aws.String(fmt.Sprintf("Admin policy for %s web distribution", bucketName)),
		PolicyDocument: aws.String(string(defaultWebPolicy)),
		PolicyName:     aws.String(fmt.Sprintf("%s-WebAdmPlc", bucketName)),
	}); err != nil {
		msg := fmt.Sprintf("failed to create web admin policy: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append policy delete to rollback tasks
	rbfunc = func(ctx context.Context) error {
		return iamService.DeletePolicy(r.Context(), &iam.DeletePolicyInput{PolicyArn: webPolicy.Arn})
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	webGroupName := fmt.Sprintf("%s-WebAdmGrp", bucketName)

	var webGroup *iam.Group
	if webGroup, err = iamService.CreateGroup(r.Context(), &iam.CreateGroupInput{
		GroupName: aws.String(webGroupName),
	}); err != nil {
		msg := fmt.Sprintf("failed to create web admin group: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append group delete to rollback tasks
	rbfunc = func(ctx context.Context) error {
		return iamService.DeleteGroup(r.Context(), &iam.DeleteGroupInput{GroupName: aws.String(webGroupName)})
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	if err = iamService.AttachGroupPolicy(r.Context(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String(webGroupName),
		PolicyArn: webPolicy.Arn,
	}); err != nil {
		msg := fmt.Sprintf("failed to attach policy %s to group %s: %s", aws.StringValue(bktPolicy.Arn), webGroupName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append detach group policy to rollback tasks
	rbfunc = func(ctx context.Context) error {
		return iamService.DetachGroupPolicy(r.Context(), &iam.DetachGroupPolicyInput{
			GroupName: aws.String(webGroupName),
			PolicyArn: webPolicy.Arn,
		})
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	var dnsChange *route53.ChangeInfo
	if dnsChange, err = route53Service.CreateRecord(r.Context(), domain.HostedZoneID, &route53.ResourceRecordSet{
		AliasTarget: &route53.AliasTarget{
			DNSName:              distribution.DomainName,
			HostedZoneId:         aws.String("Z2FDTNDATAQYW2"),
			EvaluateTargetHealth: aws.Bool(false),
		},
		Name: aws.String(bucketName),
		Type: aws.String("A"),
	}); err != nil {
		msg := fmt.Sprintf("failed to create route53 alias record for website %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// write index file
	indexMessage := "Hello, " + bucketName + "!"
	if _, err = s3Service.CreateObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Body:        bytes.NewReader([]byte(indexMessage)),
		ContentType: aws.String("text/html"),
		Key:         aws.String("index.html"),
		Tagging:     aws.String("yale:spinup=true"),
	}); err != nil {
		msg := fmt.Sprintf("failed to create default index file for website %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	output := struct {
		Bucket       *string
		Policies     []*iam.Policy
		Groups       []*iam.Group
		Distribution *cloudfront.Distribution
		DnsChange    *route53.ChangeInfo
	}{
		bucketOutput.Location,
		[]*iam.Policy{bktPolicy, webPolicy},
		[]*iam.Group{bktGroup, webGroup},
		distribution,
		dnsChange,
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal reasponse(%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// WebsiteShowHandler returns information about a static website.  Currently,
// this includes:
// - the tags
// - if the bucket is empty
// - the route53 record set
// - the cloudfront distribution summary
func (s *server) WebsiteShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	website := vars["website"]

	s3Service, ok := s.s3Services[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	cloudFrontService, ok := s.cloudFrontServices[account]
	if !ok {
		msg := fmt.Sprintf("CloudFront service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	route53Service, ok := s.route53Services[account]
	if !ok {
		msg := fmt.Sprintf("Route53 service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	// get the tags on the bucket backing the website
	// TODO get tags for other resources (cloudfront, route53, etc)
	tags, err := s3Service.GetBucketTags(r.Context(), website)
	if err != nil {
		handleError(w, err)
		return
	}

	// check if the bucket backing the website is empty, ignore the default index page (we'll clean it up)
	empty, err := s3Service.BucketEmptyWithFilter(r.Context(), website, int64(2), func(key *string) bool {
		log.Debugf("checking if object %s is 'index.html' and has 'yale:spinup=true' tag", aws.StringValue(key))

		if aws.StringValue(key) != "index.html" {
			return true
		}

		tagging, err := s3Service.Service.GetObjectTaggingWithContext(r.Context(), &s3.GetObjectTaggingInput{
			Bucket: aws.String(website),
			Key:    key,
		})
		if err != nil {
			return true
		}

		for _, tag := range tagging.TagSet {
			if aws.StringValue(tag.Key) == "yale:spinup" && aws.StringValue(tag.Value) == "true" {
				return false
			}
		}

		return true
	})
	if err != nil {
		handleError(w, err)
		return
	}

	// get details about the logging configuration for the s3 bucket backing the website
	logging, err := s3Service.GetBucketLogging(r.Context(), website)
	if err != nil {
		handleError(w, err)
		return
	}

	// determine which domain is being referenced
	domain, err := cloudFrontService.WebsiteDomain(website)
	if err != nil {
		handleError(w, err)
		return
	}

	// get the route53 resource record details
	dns, err := route53Service.GetRecordByName(r.Context(), domain.HostedZoneID, website, "A")
	if err != nil {
		handleError(w, err)
		return
	}

	dist, err := cloudFrontService.GetDistributionByName(r.Context(), website)
	if err != nil {
		handleError(w, err)
		return
	}

	// setup output struct
	output := struct {
		Tags         []*s3.Tag
		Logging      *s3.LoggingEnabled
		Empty        bool
		DNSRecord    *route53.ResourceRecordSet
		Distribution *cloudfront.DistributionSummary
	}{
		Tags:         tags,
		Logging:      logging,
		Empty:        empty,
		DNSRecord:    dns,
		Distribution: dist,
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// WebsiteDeleteHandler deletes all of the resources for a static website.  The operations are
// 1. the website bucket is deleted, this will fail if the bucket is not empty
// 2. a list of policies attached to the bucket admin group (<bucketName>-BktAdmGrp) is gathered
// 3. each of those policies is detached from the group and if it starts with '<bucketName>-', it is deleted
// 4. the bucket admin group is deleted
// 5. a list of policies attached to the web admin group (<bucketName>-WebAdmGrp) is gathered
// 6. each of those policies is detached from the group and if it starts with '<bucketName>-', it is deleted
// 7. the web admin group is deleted
// 8. the route53 dns record is deleted
// 9. the cloudfront distribution is disabled for async processing
func (s *server) WebsiteDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	website := vars["website"]

	s3Service, ok := s.s3Services[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	iamService, ok := s.iamServices[account]
	if !ok {
		msg := fmt.Sprintf("IAM service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	cloudFrontService, ok := s.cloudFrontServices[account]
	if !ok {
		msg := fmt.Sprintf("CloudFront service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	route53Service, ok := s.route53Services[account]
	if !ok {
		msg := fmt.Sprintf("Route53 service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	domain, err := cloudFrontService.WebsiteDomain(website)
	if err != nil {
		msg := fmt.Sprintf("failed to validate website domain %s", website)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// check if the bucket backing the website is empty, ignore the default index page (we'll clean it up)
	empty, err := s3Service.BucketEmptyWithFilter(r.Context(), website, int64(2), func(key *string) bool {
		log.Debugf("checking if object %s is 'index.html' and has 'yale:spinup=true' tag", aws.StringValue(key))

		if aws.StringValue(key) != "index.html" {
			return true
		}

		tagging, err := s3Service.Service.GetObjectTaggingWithContext(r.Context(), &s3.GetObjectTaggingInput{
			Bucket: aws.String(website),
			Key:    key,
		})
		if err != nil {
			return true
		}

		for _, tag := range tagging.TagSet {
			if aws.StringValue(tag.Key) == "yale:spinup" && aws.StringValue(tag.Value) == "true" {
				return false
			}
		}

		return true
	})
	if err != nil {
		handleError(w, err)
		return
	}

	if !empty {
		msg := fmt.Sprintf("cannot delete bucket %s, not empty", website)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, nil))
		return
	}

	if _, err := s3Service.DeleteObject(r.Context(), &s3.DeleteObjectInput{
		Bucket: aws.String(website),
		Key:    aws.String("index.html"),
	}); err != nil {
		log.Warnf("error trying to delete default index.html: %s", err)
	}

	if err := s3Service.DeleteEmptyBucket(r.Context(), &s3.DeleteBucketInput{Bucket: aws.String(website)}); err != nil {
		handleError(w, err)
		return
	}

	groupUsers := []*iam.User{}
	deletedPolicies := []*string{}
	groupNames := []string{fmt.Sprintf("%s-BktAdmGrp", website), fmt.Sprintf("%s-WebAdmGrp", website)}
	for _, groupName := range groupNames {
		policies, err := iamService.ListGroupPolicies(r.Context(), &iam.ListAttachedGroupPoliciesInput{GroupName: aws.String(groupName)})
		if err != nil {
			log.Warnf("failed to list group policies when deleting website %s: %s", website, err)
			j, _ := json.Marshal("failed to list group policies: " + err.Error())
			w.Write(j)
		}

		for _, p := range policies {
			if err := iamService.DetachGroupPolicy(r.Context(), &iam.DetachGroupPolicyInput{
				GroupName: aws.String(groupName),
				PolicyArn: p.PolicyArn,
			}); err != nil {
				log.Warnf("failed to detach policy %s from group %s when deleting website %s: %s", aws.StringValue(p.PolicyArn), policies, website, err)
				j, _ := json.Marshal("failed to detatch group policy: " + err.Error())
				w.Write(j)
				continue
			}

			if strings.HasPrefix(aws.StringValue(p.PolicyName), website+"-") {
				if err := iamService.DeletePolicy(r.Context(), &iam.DeletePolicyInput{PolicyArn: p.PolicyArn}); err != nil {
					log.Warnf("failed to delete group policy %s when deleting website %s: %s", aws.StringValue(p.PolicyArn), website, err)
					j, _ := json.Marshal("failed to delete group policy: " + err.Error())
					w.Write(j)
					continue
				}
				deletedPolicies = append(deletedPolicies, p.PolicyName)
			}
		}

		users, err := iamService.ListGroupUsers(r.Context(), &iam.GetGroupInput{GroupName: aws.String(groupName)})
		if err != nil {
			log.Warnf("failed to list group's users when deleting website %s: %s", website, err)
			j, _ := json.Marshal("failed to list group users: " + err.Error())
			w.Write(j)
		}

		for _, u := range users {
			if err := iamService.RemoveUserFromGroup(r.Context(), &iam.RemoveUserFromGroupInput{UserName: u.UserName, GroupName: aws.String(groupName)}); err != nil {
				log.Warnf("failed to remove user %s from group %s when deleting website %s: %s", aws.StringValue(u.UserName), groupName, website, err)
				j, _ := json.Marshal("failed to remove user from bucket admin group: " + err.Error())
				w.Write(j)
				return
			}
		}
		groupUsers = append(groupUsers, users...)

		if err := iamService.DeleteGroup(r.Context(), &iam.DeleteGroupInput{GroupName: aws.String(groupName)}); err != nil {
			log.Warnf("failed to delete group %s when deleting website %s: %s", groupName, website, err)
			j, _ := json.Marshal("failed to delete group: " + err.Error())
			w.Write(j)
		}

	}

	// find the cloudfront distribution from the website name
	distributionSummary, err := cloudFrontService.GetDistributionByName(r.Context(), website)
	if err != nil {
		handleError(w, err)
		return
	}

	// delete the alias record from route53
	dnsChange, err := route53Service.DeleteRecord(r.Context(), domain.HostedZoneID, &route53.ResourceRecordSet{
		AliasTarget: &route53.AliasTarget{
			DNSName:              distributionSummary.DomainName,
			HostedZoneId:         aws.String("Z2FDTNDATAQYW2"),
			EvaluateTargetHealth: aws.Bool(false),
		},
		Name: aws.String(website),
		Type: aws.String("A"),
	})
	if err != nil {
		msg := fmt.Sprintf("failed to delete route53 alias record for website %s: %s", website, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// disable the distribution, deletion will occur asynchronously
	distribution, err := cloudFrontService.DisableDistribution(r.Context(), aws.StringValue(distributionSummary.Id))
	if err != nil {
		msg := fmt.Sprintf("failed to disable cloudfront distribution for website %s: %s", website, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	output := struct {
		Website      *string
		Users        []*iam.User
		Policies     []*string
		Groups       []string
		Distribution *cloudfront.Distribution
		DnsChange    *route53.ChangeInfo
	}{
		aws.String(website),
		groupUsers,
		deletedPolicies,
		groupNames,
		distribution,
		dnsChange,
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal reasponse(%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

func (s *server) WebsitePartialUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	website := vars["website"]

	cloudFrontService, ok := s.cloudFrontServices[account]
	if !ok {
		msg := fmt.Sprintf("CloudFront service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	var req struct {
		CacheInvalidation []string
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create website input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// find the cloudfront distribution from the website name
	distributionSummary, err := cloudFrontService.GetDistributionByName(r.Context(), website)
	if err != nil {
		handleError(w, err)
		return
	}

	out, err := cloudFrontService.InvalidateCache(r.Context(), aws.StringValue(distributionSummary.Id), req.CacheInvalidation)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(out)
	if err != nil {
		log.Errorf("cannot marshal reasponse(%v) into JSON: %s", out, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// WebsiteUpdateHandler handles updating making changes to a website.  Currently supports:
// - Updating the bucket's tags
// - Update the cloudfront distribution's tags
func (s *server) WebsiteUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	website := vars["website"]
	s3Service, ok := s.s3Services[account]
	if !ok {
		msg := fmt.Sprintf("s3 service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	cloudFrontService, ok := s.cloudFrontServices[account]
	if !ok {
		msg := fmt.Sprintf("CloudFront service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	var req struct {
		Tags []*s3.Tag
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into update website input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// append org tag that will get applied to all resources that tag
	req.Tags = append(req.Tags, &s3.Tag{
		Key:   aws.String("spinup:org"),
		Value: aws.String(Org),
	})

	// find the cloudfront distribution from the website name
	distributionSummary, err := cloudFrontService.GetDistributionByName(r.Context(), website)
	if err != nil {
		handleError(w, err)
		return
	}

	if len(req.Tags) > 0 {
		err = s3Service.TagBucket(r.Context(), website, req.Tags)
		if err != nil {
			msg := fmt.Sprintf("failed to tag website bucket %s: %s", website, err.Error())
			handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
			return
		}

		// normalize tags
		cfTags := []*cloudfront.Tag{}
		for _, tag := range req.Tags {
			t := &cloudfront.Tag{
				Key:   tag.Key,
				Value: tag.Value,
			}
			cfTags = append(cfTags, t)
		}

		err = cloudFrontService.TagDistribution(r.Context(), aws.StringValue(distributionSummary.ARN), &cloudfront.Tags{Items: cfTags})
		if err != nil {
			msg := fmt.Sprintf("failed to tag website cloudfront distribution %s: %s", website, err.Error())
			handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}
