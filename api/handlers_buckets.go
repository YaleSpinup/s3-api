package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/YaleSpinup/apierror"
	iamapi "github.com/YaleSpinup/s3-api/iam"
	s3api "github.com/YaleSpinup/s3-api/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// BucketCreateHandler orchestrates the creation of a new s3 bucket with rollback in the event of
// failure.  The operations are
// 1. create the bucket with the given name
// 2. tag the bucket with given tags
// 3. generate the default admin bucket policy
// 4. create the admin bucket policy
// 5. create the bucket admin group, '<bucketName>-BktAdmGrp'
// 6. attach the bucket admin policy to the bucket admin group
// Note: this does _not_ create any users for managing the bucket
func (s *server) BucketCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("s3:*", "iam:*")
	if err != nil {
		log.Errorf("cannot generate policy: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		policy,
	)
	if err != nil {
		log.Errorf("failed to assume role in account: %s", accountId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s3Service := s3api.NewSession(session.Session, s.account, s.mapToAccountName(accountId))
	iamService := iamapi.NewSession(session.Session, s.account)

	var req struct {
		Tags        []*s3.Tag
		Lifecycle   *string
		BucketInput s3.CreateBucketInput
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		msg := fmt.Sprintf("cannot decode body into create bucket input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// append org tag that will get applied to all resources that tag
	req.Tags = append(req.Tags, &s3.Tag{
		Key:   aws.String("spinup:org"),
		Value: aws.String(Org),
	})

	// setup err var, rollback function list and defer execution
	// var err error
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	bucketName := aws.StringValue(req.BucketInput.Bucket)
	var bucketOutput *s3.CreateBucketOutput
	if bucketOutput, err = s3Service.CreateBucket(r.Context(), &req.BucketInput); err != nil {
		msg := fmt.Sprintf("failed to create bucket: %s", err)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append bucket delete to rollback tasks
	rollBackTasks = append(rollBackTasks, func(ctx context.Context) error {
		if err := s3Service.DeleteEmptyBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucketName)}); err != nil {
			return err
		}
		return nil
	})

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
		msg := fmt.Sprintf("failed to create bucket %s, timeout waiting for create: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
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
		msg := fmt.Sprintf("failed to tag bucket %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	if req.Lifecycle != nil {
		// Get the supported lifecycle and error if not
		lifecycle := s3api.Lifecycles.GetLifecycle(*req.Lifecycle)
		if lifecycle == nil {
			handleError(w, errors.Wrap(errors.New("lifecycle doesnt exist in supported lifecycles"), ""))
			return
		}

		// Update the bucket lifecycle config
		if err = s3Service.PutBucketLifecycleConfiguration(r.Context(), &s3.PutBucketLifecycleConfigurationInput{
			Bucket:                 aws.String(bucketName),
			LifecycleConfiguration: &s3.BucketLifecycleConfiguration{Rules: []*s3.LifecycleRule{lifecycle}},
		}); err != nil {
			msg := fmt.Sprintf("failed to update bucket lifecycle configuration%s: %s", bucketName, err.Error())
			handleError(w, errors.Wrap(err, msg))
			return
		}

		// append bucket delete to rollback tasks
		rollBackTasks = append(rollBackTasks, func(ctx context.Context) error {
			if err := s3Service.DeleteBucketLifecycle(ctx, &s3.DeleteBucketLifecycleInput{Bucket: aws.String(bucketName)}); err != nil {
				return err
			}
			return nil
		})
	}

	// enable AWS managed serverside encryption for the bucket
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

	// enable logging access for the bucket to a central repo if the target bucket is set
	fmt.Println("Bucket name ::::::::::::::::::::::::, ", s3Service.LoggingBucket)
	if s3Service.LoggingBucket != "" {
		if err = s3Service.UpdateBucketLogging(r.Context(), bucketName, s3Service.LoggingBucket, s3Service.LoggingBucketPrefix); err != nil {
			msg := fmt.Sprintf("failed to enable logging for bucket %s: %s", bucketName, err.Error())
			handleError(w, errors.Wrap(err, msg))
			return
		}
	}

	// build the default IAM bucket admin policy (from the config and known inputs)
	var defaultPolicy []byte
	if defaultPolicy, err = iamService.DefaultBucketAdminPolicy(aws.String(bucketName)); err != nil {
		msg := fmt.Sprintf("failed creating default IAM policy for bucket %s: %s", bucketName, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	var iamPolicy *iam.Policy
	if iamPolicy, err = iamService.CreatePolicy(r.Context(), &iam.CreatePolicyInput{
		Description:    aws.String(fmt.Sprintf("Admin policy for %s bucket", bucketName)),
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String(fmt.Sprintf("%s-BktAdmPlc", bucketName)),
	}); err != nil {
		msg := fmt.Sprintf("failed to create policy: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append policy delete to rollback tasks
	rollBackTasks = append(rollBackTasks, func(ctx context.Context) error {
		if err := iamService.DeletePolicy(ctx, &iam.DeletePolicyInput{PolicyArn: iamPolicy.Arn}); err != nil {
			return err
		}
		return nil
	})

	groupName := fmt.Sprintf("%s-BktAdmGrp", bucketName)

	var group *iam.Group
	if group, err = iamService.CreateGroup(r.Context(), &iam.CreateGroupInput{
		GroupName: aws.String(groupName),
	}); err != nil {
		msg := fmt.Sprintf("failed to create group: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append group delete to rollback tasks
	rollBackTasks = append(rollBackTasks, func(ctx context.Context) error {
		if err := iamService.DeleteGroup(ctx, &iam.DeleteGroupInput{GroupName: aws.String(groupName)}); err != nil {
			return err
		}
		return nil
	})

	if err = iamService.AttachGroupPolicy(r.Context(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String(groupName),
		PolicyArn: iamPolicy.Arn,
	}); err != nil {
		msg := fmt.Sprintf("failed to create group: %s", err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	output := struct {
		Bucket *string
		Policy *iam.Policy
		Group  *iam.Group
	}{
		bucketOutput.Location,
		iamPolicy,
		group,
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

// BucketListHandler gets a list of all buckets in the account
func (s *server) BucketListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("s3:ListBucket")
	if err != nil {
		log.Errorf("cannot generate policy: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
	)
	if err != nil {
		log.Errorf("failed to assume role in account: %s", accountId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s3Client := s3api.NewSession(session.Session, s.account, s.mapToAccountName(accountId))
	output, err := s3Client.ListBuckets(r.Context(), &s3.ListBucketsInput{})
	if err != nil {
		handleError(w, err)
		return
	}

	buckets := []string{}
	for _, b := range output {
		buckets = append(buckets, aws.StringValue(b.Name))
	}

	j, err := json.Marshal(buckets)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", buckets, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// BucketHeadHandler checks if a bucket exists
func (s *server) BucketHeadHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	bucket := vars["bucket"]
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("s3:ListAllMyBuckets")
	if err != nil {
		log.Errorf("cannot generate policy: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
	)
	if err != nil {
		log.Errorf("failed to assume role in account: %s", accountId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s3Client := s3api.NewSession(session.Session, s.account, s.mapToAccountName(accountId))

	log.Infof("checking if bucket exists: %s", bucket)
	exists, err := s3Client.BucketExists(r.Context(), bucket)
	if err != nil {
		handleError(w, err)
		return
	}

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte{})
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}

// BucketDeleteHandler deletes an empty bucket and all of it's dependencies.  The operations are
// 1. the bucket is deleted, this will fail if the bucket is not empty
// 2. a list of policies attached to the bucket admin group (<bucketName>-BktAdmGrp) is gathered
// 3. each of those policies is detached from the group and if it starts with '<bucketName>-', it is deleted
// 4. the bucket admin group is deleted
func (s *server) BucketDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	bucket := vars["bucket"]

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("s3:*", "iam:*")
	if err != nil {
		log.Errorf("cannot generate policy: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		policy,
	)
	if err != nil {
		log.Errorf("failed to assume role in account: %s", accountId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s3Service := s3api.NewSession(session.Session, s.account, s.mapToAccountName(accountId))
	iamService := iamapi.NewSession(session.Session, s.account)

	err = s3Service.DeleteEmptyBucket(r.Context(), &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		handleError(w, err)
		return
	}

	for _, g := range []string{"BktAdmGrp", "BktRWGrp", "BktROGrp"} {
		groupName := fmt.Sprintf("%s-%s", bucket, g)

		// TODO: if this fails with a NotFound, we should continue on because its probably a legacy bucket
		policies, err := iamService.ListGroupPolicies(r.Context(), &iam.ListAttachedGroupPoliciesInput{GroupName: aws.String(groupName)})
		if err != nil {
			log.Warnf("failed to list group policies when deleting bucket %s: %s", bucket, err)
			continue
		}

		for _, p := range policies {
			if err := iamService.DetachGroupPolicy(r.Context(), &iam.DetachGroupPolicyInput{
				GroupName: aws.String(groupName),
				PolicyArn: p.PolicyArn,
			}); err != nil {
				log.Warnf("failed to detach policy %s from group %s when deleting bucket %s: %s", aws.StringValue(p.PolicyArn), groupName, bucket, err)
			}

			if strings.HasPrefix(aws.StringValue(p.PolicyName), bucket+"-") {
				if err := iamService.DeletePolicy(r.Context(), &iam.DeletePolicyInput{PolicyArn: p.PolicyArn}); err != nil {
					log.Warnf("failed to delete group policy %s when deleting bucket %s: %s", aws.StringValue(p.PolicyArn), bucket, err)
				}
			}
		}

		users, err := iamService.ListGroupUsers(r.Context(), &iam.GetGroupInput{GroupName: aws.String(groupName)})
		if err != nil {
			log.Warnf("failed to list group's users when deleting bucket %s: %s", bucket, err)
			continue
		}

		for _, u := range users {
			// get a users access keys
			keys, err := iamService.ListAccessKeys(r.Context(), &iam.ListAccessKeysInput{UserName: u.UserName})
			if err != nil {
				handleError(w, err)
				return
			}

			// delete the access keys
			for _, k := range keys {
				err = iamService.DeleteAccessKey(r.Context(), &iam.DeleteAccessKeyInput{UserName: u.UserName, AccessKeyId: k.AccessKeyId})
				if err != nil {
					handleError(w, err)
					return
				}
			}

			if err := iamService.RemoveUserFromGroup(r.Context(), &iam.RemoveUserFromGroupInput{UserName: u.UserName, GroupName: aws.String(groupName)}); err != nil {
				log.Warnf("failed to remove user %s from group %s when deleting bucket %s: %s", aws.StringValue(u.UserName), groupName, bucket, err)
			}
		}

		if err := iamService.DeleteGroup(r.Context(), &iam.DeleteGroupInput{GroupName: aws.String(groupName)}); err != nil {
			log.Warnf("failed to delete group %s when deleting bucket %s: %s", groupName, bucket, err)
			continue
		}

		for _, u := range users {
			_, err := iamService.GetUser(r.Context(), &iam.GetUserInput{
				UserName: u.UserName,
			})
			if err == nil {
				err = iamService.DeleteUser(r.Context(), &iam.DeleteUserInput{UserName: u.UserName})
				if err != nil {
					log.Warnf("failed to delete user: %s, %s", aws.StringValue(u.UserName), err)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}

// BucketShowHandler returns information about a bucket
func (s *server) BucketShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	bucket := vars["bucket"]

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("s3:ListBucket")
	if err != nil {
		log.Errorf("cannot generate policy: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
	)
	if err != nil {
		log.Errorf("failed to assume role in account: %s", accountId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s3Client := s3api.NewSession(session.Session, s.account, s.mapToAccountName(accountId))

	tags, err := s3Client.GetBucketTags(r.Context(), bucket)
	if err != nil {
		handleError(w, err)
		return
	}

	empty, err := s3Client.BucketEmpty(r.Context(), bucket)
	if err != nil {
		handleError(w, err)
		return
	}

	logging, err := s3Client.GetBucketLogging(r.Context(), bucket)
	if err != nil {
		handleError(w, err)
		return
	}

	// setup output struct
	output := struct {
		Tags    []*s3.Tag
		Logging *s3.LoggingEnabled
		Empty   bool
	}{
		Tags:    tags,
		Logging: logging,
		Empty:   empty,
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

// BucketUpdateHandler handles updating making changes to a bucket.  Currently supports:
// - Updating the bucket's tags
func (s *server) BucketUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	bucket := vars["bucket"]
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("s3:PutBucketTagging")
	if err != nil {
		log.Errorf("cannot generate policy: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonS3FullAccess",
	)
	if err != nil {
		log.Errorf("failed to assume role in account: %s", accountId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s3Client := s3api.NewSession(session.Session, s.account, s.mapToAccountName(accountId))

	var req struct {
		Tags []*s3.Tag
	}
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into update bucket input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// append org tag that will get applied to all resources that tag
	req.Tags = append(req.Tags, &s3.Tag{
		Key:   aws.String("spinup:org"),
		Value: aws.String(Org),
	})

	if len(req.Tags) > 0 {
		err = s3Client.TagBucket(r.Context(), bucket, req.Tags)
		if err != nil {
			msg := fmt.Sprintf("failed to tag bucket %s: %s", bucket, err.Error())
			handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}
