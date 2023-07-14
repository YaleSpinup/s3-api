package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/s3-api/common"
	iamapi "github.com/YaleSpinup/s3-api/iam"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// UserCreateHandler creates a new user for a bucket
func (s *server) UserCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	accountId := s.mapAccountNumber(vars["account"])

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("iam:*")
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

	iamService := iamapi.NewSession(session.Session, s.account)

	var req struct {
		User   *iam.CreateUserInput
		Groups []string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		msg := fmt.Sprintf("cannot decode body into create user input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	if req.User == nil || req.Groups == nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "user and groups input are required", nil))
		return
	}

	// setup err var, rollback function list and defer execution
	// var err error
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	userOutput, err := iamService.CreateUser(r.Context(), req.User)
	if err != nil {
		msg := fmt.Sprintf("failed to create user for bucket %s: %s", bucket, err)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// wait for the user to exist
	if err = retry(3, 2*time.Second, func() error {
		log.Infof("checking if user exists before continuing: %s", aws.StringValue(userOutput.User.UserName))
		out, err := iamService.GetUser(r.Context(), &iam.GetUserInput{
			UserName: userOutput.User.UserName,
		})
		if err != nil {
			return err
		}

		log.Debugf("got user output: %s", awsutil.Prettify(out))
		return nil
	}); err != nil {
		msg := fmt.Sprintf("failed to create user %s for bucket %s: timeout waiting for create %s", aws.StringValue(req.User.UserName), bucket, err)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append user delete to rollback tasks
	rbfunc := func(ctx context.Context) error {
		if err := iamService.DeleteUser(ctx, &iam.DeleteUserInput{UserName: req.User.UserName}); err != nil {
			return err
		}
		return nil
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	for _, group := range req.Groups {
		groupName := fmt.Sprintf("%s-%s", bucket, group)
		_, err = iamService.GetGroup(r.Context(), groupName)
		if err != nil {
			if aerr, ok := err.(apierror.Error); ok && aerr.Code == apierror.ErrNotFound {
				var rbTasks []rollbackFunc
				rbTasks, err = s.CreateBucketGroupPolicy(r.Context(), iamService, bucket, group)
				if err != nil {
					handleError(w, err)
					return
				}
				rollBackTasks = append(rollBackTasks, rbTasks...)
			} else {
				handleError(w, err)
				return
			}
		}

		if err = iamService.AddUserToGroup(r.Context(), &iam.AddUserToGroupInput{
			UserName:  userOutput.User.UserName,
			GroupName: aws.String(groupName),
		}); err != nil {
			msg := fmt.Sprintf("failed to add user: %s to group %s for bucket %s", aws.StringValue(userOutput.User.UserName), group, bucket)
			handleError(w, errors.Wrap(err, msg))
			return
		}

		// append detach group to rollback funciton
		rbfunc = func(ctx context.Context) error {
			if err := iamService.RemoveUserFromGroup(ctx, &iam.RemoveUserFromGroupInput{
				UserName:  userOutput.User.UserName,
				GroupName: aws.String(groupName),
			}); err != nil {
				return err
			}
			return nil
		}
		rollBackTasks = append(rollBackTasks, rbfunc)
	}

	output := struct {
		User *iam.User
	}{
		userOutput.User,
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

// UserDeleteHandler deletes an iam user and their keys
func (s *server) UserDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	user := vars["user"]
	bucket := vars["bucket"]

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("iam:*")
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

	iamService := iamapi.NewSession(session.Session, s.account)

	// get a users access keys
	keys, err := iamService.ListAccessKeys(r.Context(), &iam.ListAccessKeysInput{UserName: aws.String(user)})
	if err != nil {
		handleError(w, err)
		return
	}

	// delete the access keys
	for _, k := range keys {
		err = iamService.DeleteAccessKey(r.Context(), &iam.DeleteAccessKeyInput{UserName: aws.String(user), AccessKeyId: k.AccessKeyId})
		if err != nil {
			handleError(w, err)
			return
		}
	}

	// get the list of groups that the user belongs to
	groups, err := iamService.ListUserGroups(r.Context(), &iam.ListGroupsForUserInput{UserName: aws.String(user)})
	if err != nil {
		handleError(w, err)
		return
	}

	// remove the user from all group membership
	for _, g := range groups {
		err = iamService.RemoveUserFromGroup(r.Context(), &iam.RemoveUserFromGroupInput{UserName: aws.String(user), GroupName: g.GroupName})
		if err != nil {
			handleError(w, err)
			return
		}
	}

	// get a list of all of the attached user policies for a user.  this should be empty for "new" s3 buckets, but buckets created
	// with the legacy service have the policy directly attached to the user with the same name as the user/bucket
	policies, err := iamService.ListUserPolicies(r.Context(), &iam.ListAttachedUserPoliciesInput{UserName: aws.String(user)})
	if err != nil {
		handleError(w, err)
		return
	}

	// detatch and delete all of the policies that we found if the name is the same as the bucket or if the name starts
	// with the the name of the bucket
	for _, p := range policies {
		pname := aws.StringValue(p.PolicyName)
		if strings.HasPrefix(pname, bucket+"-") || pname == bucket {
			if err := iamService.DetachUserPolicy(r.Context(), &iam.DetachUserPolicyInput{
				UserName:  aws.String(user),
				PolicyArn: p.PolicyArn,
			}); err != nil {
				j, _ := json.Marshal("failed to detatch user policy: " + err.Error())
				w.Write(j)
				return
			}
		}
	}

	err = iamService.DeleteUser(r.Context(), &iam.DeleteUserInput{UserName: aws.String(user)})
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}

func (s *server) UserUpdateKeyHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	bucket := vars["bucket"]
	user := vars["user"]

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("iam:*")
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

	iamService := iamapi.NewSession(session.Session, s.account)

	// get a list of users access keys
	keys, kerr := iamService.ListAccessKeys(r.Context(), &iam.ListAccessKeysInput{UserName: aws.String(user)})
	if kerr != nil {
		handleError(w, kerr)
		return
	}

	// setup err var, rollback function list and defer execution
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	newKeyOutput, err := iamService.CreateAccessKey(r.Context(), &iam.CreateAccessKeyInput{UserName: aws.String(user)})
	if err != nil {
		msg := fmt.Sprintf("failed to create access key for user: %s, bucket %s", user, bucket)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append access key delete to rollback tasks
	rbfunc := func(ctx context.Context) error {
		if err := iamService.DeleteAccessKey(ctx, &iam.DeleteAccessKeyInput{
			UserName:    newKeyOutput.AccessKey.UserName,
			AccessKeyId: newKeyOutput.AccessKey.AccessKeyId,
		}); err != nil {
			return err
		}
		return nil
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	deletedKeyIds := []*string{}
	// delete the old access keys
	for _, k := range keys {
		err = iamService.DeleteAccessKey(r.Context(), &iam.DeleteAccessKeyInput{UserName: aws.String(user), AccessKeyId: k.AccessKeyId})
		if err != nil {
			msg := fmt.Sprintf("unable to delete access key id %s for user %s", user, aws.StringValue(k.AccessKeyId))
			handleError(w, errors.Wrap(err, msg))
			return
		}
		deletedKeyIds = append(deletedKeyIds, k.AccessKeyId)
	}

	var output = struct {
		DeletedKeyIds []*string
		AccessKey     *iam.AccessKey
	}{
		deletedKeyIds,
		newKeyOutput.AccessKey,
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

// UserListHandler lists the users for a bucket.  It tries to return the members of the predefined
// bucket management groups: <<bucket>>-BktAdmGrp,  <<bucket>>-BktRWGrp, <<bucket>>-BktROGrp. It also
// looks for a user with the same name as the bucket and returns that if it exists.
func (s *server) UserListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	//account := vars["account"]
	bucket := vars["bucket"]
	accountId := s.mapAccountNumber(vars["account"])
	print("acccccccccc", accountId)
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("s3:Get*", "iam:*")
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

	iamService := iamapi.NewSession(session.Session, common.Account{})
	// iamService, _ = s.iamServices[vars["account"]]

	// TODO check if bucket exists and fail if it doesn't?

	users := []*iam.User{}
	for _, g := range []string{"BktAdmGrp", "BktRWGrp", "BktROGrp"} {
		groupName := fmt.Sprintf("%s-%s", bucket, g)
		u, err := iamService.ListGroupUsers(r.Context(), &iam.GetGroupInput{GroupName: aws.String(groupName)})
		if err != nil {
			log.Warnf("error listing bucket %s group %s users %s", bucket, groupName, err)
			continue
		}
		users = append(users, u...)
	}

	// check if there is a user with the same name as the bucket to support legacy buckets
	user, err := iamService.GetUser(r.Context(), &iam.GetUserInput{UserName: aws.String(bucket)})
	fmt.Println(err)
	if err == nil {
		users = append(users, user.User)
		//users = []*iam.User{user.User}
	}

	log.Debugf("%+v", users)

	j, err := json.Marshal(users)
	if err != nil {
		log.Errorf("cannot marshal reasponse(%v) into JSON: %s", users, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// UserShowHandler gets and returns details of a bucket user.  This is accomplished by getting all of the
// users for a bucket's management groups and then comparing that to the passed user.  This would be more
// efficient if we just GetUser for the passed in user, but then we can't be sure it's associated with the
// bucket.
func (s *server) UserShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	bucket := vars["bucket"]
	user := vars["user"]

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("iam:*")
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

	iamService := iamapi.NewSession(session.Session, s.account)

	// collect the list of users in the various management groups
	users := []*iam.User{}
	for _, g := range []string{"BktAdmGrp", "BktRWGrp", "BktROGrp"} {
		groupName := fmt.Sprintf("%s-%s", bucket, g)
		grpUsers, err := iamService.ListGroupUsers(r.Context(), &iam.GetGroupInput{GroupName: aws.String(groupName)})
		if err != nil {
			log.Warnf("error getting users for the %s goup", groupName)
			continue
		}

		users = append(users, grpUsers...)
	}

	// check if there is a user with the same name as the bucket to support legacy buckets
	u, err := iamService.GetUser(r.Context(), &iam.GetUserInput{UserName: aws.String(bucket)})
	if err == nil {
		users = append(users, u.User)
	}

	// range over all of the users we found and return the user if it matches the requested user
	for _, u := range users {
		if aws.StringValue(u.UserName) == user {
			var userDetails = struct {
				User       *iam.User
				AccessKeys []*iam.AccessKeyMetadata
				Groups     []*iam.Group
				Policies   []*iam.AttachedPolicy
			}{
				User: u,
			}

			keys, err := iamService.ListAccessKeys(r.Context(), &iam.ListAccessKeysInput{UserName: aws.String(user)})
			if err != nil {
				handleError(w, err)
				return
			}
			userDetails.AccessKeys = keys

			groups, err := iamService.ListUserGroups(r.Context(), &iam.ListGroupsForUserInput{UserName: aws.String(user)})
			if err != nil {
				handleError(w, err)
				return
			}
			userDetails.Groups = groups

			policies, err := iamService.ListUserPolicies(r.Context(), &iam.ListAttachedUserPoliciesInput{UserName: aws.String(user)})
			if err != nil {
				handleError(w, err)
				return
			}
			userDetails.Policies = policies

			j, err := json.Marshal(userDetails)
			if err != nil {
				log.Errorf("cannot marshal reasponse(%v) into JSON: %s", u, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(j)

			return
		}
	}

	log.Warnf("requested user %s does not exist or does not belong to bucket %s", user, bucket)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte{})
}
