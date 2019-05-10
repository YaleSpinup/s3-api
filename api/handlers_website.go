package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/s3-api/apierror"
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
// 9. create alias record in route53
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
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create website input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// setup err var, rollback function list and defer execution, note that we depend on the err variable defined above this
	var rollBackTasks []func() error
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	bucketName := aws.StringValue(req.BucketInput.Bucket)
	domain, err := cloudFrontService.WebsiteDomain(bucketName)
	if err != nil {
		msg := fmt.Sprintf("failed to validate website domain %s", bucketName)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	bucketOutput, err := s3Service.CreateBucket(r.Context(), &req.BucketInput)
	if err != nil {
		msg := fmt.Sprintf("failed to create bucket %s", bucketName)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append bucket delete to rollback tasks
	rbfunc := func() error {
		return func() error {
			_, err := s3Service.DeleteEmptyBucket(r.Context(), &s3.DeleteBucketInput{Bucket: aws.String(bucketName)})
			return err
		}()
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	err = s3Service.TagBucket(r.Context(), bucketName, req.Tags)
	if err != nil {
		msg := fmt.Sprintf("failed to tag bucket %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// enable AWS managed serverside encryption for the website/bucket
	err = s3Service.UpdateBucketEncryption(r.Context(), &s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucketName),
		ServerSideEncryptionConfiguration: &s3.ServerSideEncryptionConfiguration{
			Rules: []*s3.ServerSideEncryptionRule{
				&s3.ServerSideEncryptionRule{
					ApplyServerSideEncryptionByDefault: &s3.ServerSideEncryptionByDefault{
						SSEAlgorithm: aws.String("AES256"),
					},
				},
			},
		},
	})
	if err != nil {
		msg := fmt.Sprintf("failed to enable encryption for bucket %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// enable logging access for the website/bucket to a central repo
	if s3Service.LoggingBucket != "" {
		err = s3Service.UpdateBucketLogging(r.Context(), bucketName, s3Service.LoggingBucket, s3Service.LoggingBucketPrefix)
		if err != nil {
			msg := fmt.Sprintf("failed to enable logging for bucket %s: %s", bucketName, err.Error())
			handleError(w, errors.Wrap(err, msg))
			return
		}
	}

	err = s3Service.UpdateWebsiteConfig(r.Context(), &s3.PutBucketWebsiteInput{
		Bucket:               aws.String(bucketName),
		WebsiteConfiguration: &req.WebsiteConfiguration,
	})
	if err != nil {
		msg := fmt.Sprintf("failed to configure bucket %s as website: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	defaultWebsitePolicy, err := iamService.DefaultWebsiteAccessPolicy(aws.String(bucketName))
	if err != nil {
		msg := fmt.Sprintf("failed building default website bucket access policy for %s: %s", bucketName, err.Error())
		handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
		return
	}

	err = s3Service.UpdateBucketPolicy(r.Context(), &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucketName),
		Policy: aws.String(string(defaultWebsitePolicy)),
	})

	// build the default IAM bucket admin policy (from the config and known inputs)
	defaultPolicy, err := iamService.DefaultBucketAdminPolicy(aws.String(bucketName))
	if err != nil {
		msg := fmt.Sprintf("failed building default IAM policy for bucket %s: %s", bucketName, err.Error())
		handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
		return
	}

	policyOutput, err := iamService.CreatePolicy(r.Context(), &iam.CreatePolicyInput{
		Description:    aws.String(fmt.Sprintf("Admin policy for %s bucket", bucketName)),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String(fmt.Sprintf("%s-BktAdmPlc", bucketName)),
	})

	if err != nil {
		msg := fmt.Sprintf("failed to create policy: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append policy delete to rollback tasks
	rbfunc = func() error {
		return func() error {
			_, err := iamService.DeletePolicy(r.Context(), &iam.DeletePolicyInput{PolicyArn: policyOutput.Policy.Arn})
			return err
		}()
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	groupName := fmt.Sprintf("%s-BktAdmGrp", bucketName)
	group, err := iamService.CreateGroup(r.Context(), &iam.CreateGroupInput{
		GroupName: aws.String(groupName),
	})

	if err != nil {
		msg := fmt.Sprintf("failed to create group: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append group delete to rollback tasks
	rbfunc = func() error {
		return func() error {
			_, err := iamService.DeleteGroup(r.Context(), &iam.DeleteGroupInput{GroupName: aws.String(groupName)})
			return err
		}()
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	if _, err = iamService.AttachGroupPolicy(r.Context(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String(groupName),
		PolicyArn: policyOutput.Policy.Arn,
	}); err != nil {
		msg := fmt.Sprintf("failed to attach policy %s to group %s: %s", aws.StringValue(policyOutput.Policy.Arn), groupName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append detach group policy to rollback tasks
	rbfunc = func() error {
		return func() error {
			return iamService.DetachGroupPolicy(r.Context(), &iam.DetachGroupPolicyInput{
				GroupName: aws.String(groupName),
				PolicyArn: policyOutput.Policy.Arn,
			})
		}()
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

	defaultWebsiteDistribution, err := cloudFrontService.DefaultWebsiteDistributionConfig(bucketName)
	if err != nil {
		msg := fmt.Sprintf("failed to generate default website distribution config for %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	distribution, err := cloudFrontService.CreateDistribution(r.Context(), defaultWebsiteDistribution, &cloudfront.Tags{Items: cfTags})
	if err != nil {
		msg := fmt.Sprintf("failed to create cloudfront distribution for website %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}
	// TODO: rollback for cloudfront distribution
	// rbfunc = func() error {
	// 	return func() error {
	// 		return cloudFrontService.DeleteDistribution(r.Context(), ...)
	// 	}()
	// }
	// rollBackTasks = append(rollBackTasks, rbfunc)

	dnsChange, err := route53Service.CreateRecord(r.Context(), domain.HostedZoneID, &route53.ResourceRecordSet{
		AliasTarget: &route53.AliasTarget{
			DNSName:              distribution.DomainName,
			HostedZoneId:         aws.String("Z2FDTNDATAQYW2"),
			EvaluateTargetHealth: aws.Bool(false),
		},
		Name: aws.String(bucketName),
		Type: aws.String("A"),
	})
	if err != nil {
		msg := fmt.Sprintf("failed to create route53 alias record for website %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	output := struct {
		Bucket       *string
		Policy       *iam.Policy
		Group        *iam.Group
		Distribution *cloudfront.Distribution
		DnsChange    *route53.ChangeInfo
	}{
		bucketOutput.Location,
		policyOutput.Policy,
		group.Group,
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

// WebsiteShowHandler returns information about a static website
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

	// check if the bucket backing the website is empty
	empty, err := s3Service.BucketEmpty(r.Context(), website)
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
