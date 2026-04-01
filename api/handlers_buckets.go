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
// 3. configure lifecycle, encryption, and logging
// Note: IAM policies are now attached as inline policies when users are created (see UserCreateHandler)
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

	output := struct {
		Bucket *string
	}{
		bucketOutput.Location,
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
// 2. (legacy) group-based cleanup: detach/delete policies, remove users from groups, delete groups
// 3. (new) inline-policy-based cleanup: find users by prefix, delete inline policies, access keys, and users
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

	// --- Legacy group-based cleanup ---
	for _, g := range []string{"BktAdmGrp", "BktRWGrp", "BktROGrp"} {
		groupName := fmt.Sprintf("%s-%s", bucket, g)

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
			keys, err := iamService.ListAccessKeys(r.Context(), &iam.ListAccessKeysInput{UserName: u.UserName})
			if err != nil {
				handleError(w, err)
				return
			}

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

	// --- New inline-policy-based cleanup ---
	// Find users by bucket name prefix (e.g., "mybucket-")
	inlineUsers, err := iamService.ListUsers(r.Context(), bucket+"-")
	if err != nil {
		log.Warnf("failed to list inline policy users for bucket %s: %s", bucket, err)
	}

	for _, u := range inlineUsers {
		userName := aws.StringValue(u.UserName)

		// Delete inline policies
		inlinePolicies, err := iamService.ListUserInlinePolicies(r.Context(), &iam.ListUserPoliciesInput{UserName: u.UserName})
		if err != nil {
			log.Warnf("failed to list inline policies for user %s: %s", userName, err)
		}
		for _, pName := range inlinePolicies {
			if err := iamService.DeleteUserPolicy(r.Context(), &iam.DeleteUserPolicyInput{
				UserName:   u.UserName,
				PolicyName: pName,
			}); err != nil {
				log.Warnf("failed to delete inline policy %s for user %s: %s", aws.StringValue(pName), userName, err)
			}
		}

		// Delete access keys
		keys, err := iamService.ListAccessKeys(r.Context(), &iam.ListAccessKeysInput{UserName: u.UserName})
		if err != nil {
			log.Warnf("failed to list access keys for user %s: %s", userName, err)
		}
		for _, k := range keys {
			if err := iamService.DeleteAccessKey(r.Context(), &iam.DeleteAccessKeyInput{UserName: u.UserName, AccessKeyId: k.AccessKeyId}); err != nil {
				log.Warnf("failed to delete access key for user %s: %s", userName, err)
			}
		}

		// Remove from any remaining groups
		groups, err := iamService.ListUserGroups(r.Context(), &iam.ListGroupsForUserInput{UserName: u.UserName})
		if err != nil {
			log.Warnf("failed to list groups for user %s: %s", userName, err)
		}
		for _, g := range groups {
			if err := iamService.RemoveUserFromGroup(r.Context(), &iam.RemoveUserFromGroupInput{UserName: u.UserName, GroupName: g.GroupName}); err != nil {
				log.Warnf("failed to remove user %s from group %s: %s", userName, aws.StringValue(g.GroupName), err)
			}
		}

		// Detach managed policies
		managedPolicies, err := iamService.ListUserPolicies(r.Context(), &iam.ListAttachedUserPoliciesInput{UserName: u.UserName})
		if err != nil {
			log.Warnf("failed to list managed policies for user %s: %s", userName, err)
		}
		for _, p := range managedPolicies {
			if err := iamService.DetachUserPolicy(r.Context(), &iam.DetachUserPolicyInput{UserName: u.UserName, PolicyArn: p.PolicyArn}); err != nil {
				log.Warnf("failed to detach policy from user %s: %s", userName, err)
			}
		}

		// Delete the user
		if err := iamService.DeleteUser(r.Context(), &iam.DeleteUserInput{UserName: u.UserName}); err != nil {
			log.Warnf("failed to delete inline policy user %s: %s", userName, err)
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
	policy, err := generatePolicy("s3:PutBucketTagging", "s3:PutBucketPolicy")
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
		BucketPolicy *string
		Tags         []*s3.Tag
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

	// If there are tags to update
	if len(req.Tags) > 0 {
		err = s3Client.TagBucket(r.Context(), bucket, req.Tags)
		if err != nil {
			msg := fmt.Sprintf("failed to tag bucket %s: %s", bucket, err.Error())
			handleError(w, apierror.New(apierror.ErrInternalError, msg, err))
			return
		}
	}

	// If there is a policy to update
	if req.BucketPolicy != nil {
		if err = s3Client.UpdateBucketPolicy(r.Context(), &s3.PutBucketPolicyInput{
			Bucket: aws.String(bucket),
			Policy: aws.String(string(*req.BucketPolicy)),
		}); err != nil {
			handleError(w, err)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}
