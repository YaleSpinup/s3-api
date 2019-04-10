package s3api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// UserCreateHandler creates a new user for a bucket
func (s *server) UserCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	bucket := vars["bucket"]

	iamService, ok := s.iamServices[account]
	if !ok {
		msg := fmt.Sprintf("IAM service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	var req iam.CreateUserInput
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create user input: %s", err)
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

	userOutput, err := iamService.CreateUser(r.Context(), &req)
	if err != nil {
		msg := fmt.Sprintf("failed to create user for bucket %s: %s", bucket, err)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append user delete to rollback tasks
	rbfunc := func() error {
		return func() error {
			if _, err := iamService.DeleteUser(r.Context(), &iam.DeleteUserInput{UserName: req.UserName}); err != nil {
				return err
			}
			return nil
		}()
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	keyOutput, err := iamService.CreateAccessKey(r.Context(), &iam.CreateAccessKeyInput{UserName: userOutput.User.UserName})
	if err != nil {
		msg := fmt.Sprintf("failed to create access key for user: %s, bucket %s", aws.StringValue(userOutput.User.UserName), bucket)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append access key delete to rollback tasks
	rbfunc = func() error {
		return func() error {
			if _, err := iamService.DeleteAccessKey(r.Context(), &iam.DeleteAccessKeyInput{
				UserName:    keyOutput.AccessKey.UserName,
				AccessKeyId: keyOutput.AccessKey.AccessKeyId,
			}); err != nil {
				return err
			}
			return nil
		}()
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	groupName := fmt.Sprintf("%s-BktAdmGrp", bucket)
	if _, err := iamService.AddUserToGroup(r.Context(), &iam.AddUserToGroupInput{
		UserName:  userOutput.User.UserName,
		GroupName: aws.String(groupName),
	}); err != nil {
		msg := fmt.Sprintf("failed to add user: %s to group %s for bucket %s", aws.StringValue(userOutput.User.UserName), groupName, bucket)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	output := struct {
		User      *iam.User
		AccessKey *iam.AccessKey
	}{
		userOutput.User,
		keyOutput.AccessKey,
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
	account := vars["account"]
	user := vars["user"]
	bucket := vars["bucket"]

	iamService, ok := s.iamServices[account]
	if !ok {
		msg := fmt.Sprintf("IAM service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	// get a users access keys
	keys, err := iamService.ListAccessKeys(r.Context(), &iam.ListAccessKeysInput{UserName: aws.String(user)})
	if err != nil {
		handleError(w, err)
		return
	}

	// delete the access keys
	for _, k := range keys {
		_, err = iamService.DeleteAccessKey(r.Context(), &iam.DeleteAccessKeyInput{UserName: aws.String(user), AccessKeyId: k.AccessKeyId})
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

			if _, err := iamService.DeletePolicy(r.Context(), &iam.DeletePolicyInput{PolicyArn: p.PolicyArn}); err != nil {
				j, _ := json.Marshal("failed to delete user policy: " + err.Error())
				w.Write(j)
				return
			}
		}
	}

	_, err = iamService.DeleteUser(r.Context(), &iam.DeleteUserInput{UserName: aws.String(user)})
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
	account := vars["account"]
	bucket := vars["bucket"]
	user := vars["user"]

	iamService, ok := s.iamServices[account]
	if !ok {
		msg := fmt.Sprintf("IAM service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	// get a list of users access keys
	keys, err := iamService.ListAccessKeys(r.Context(), &iam.ListAccessKeysInput{UserName: aws.String(user)})
	if err != nil {
		handleError(w, err)
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

	newKeyOutput, err := iamService.CreateAccessKey(r.Context(), &iam.CreateAccessKeyInput{UserName: aws.String(user)})
	if err != nil {
		msg := fmt.Sprintf("failed to create access key for user: %s, bucket %s", user, bucket)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append access key delete to rollback tasks
	rbfunc := func() error {
		return func() error {
			if _, err := iamService.DeleteAccessKey(r.Context(), &iam.DeleteAccessKeyInput{
				UserName:    newKeyOutput.AccessKey.UserName,
				AccessKeyId: newKeyOutput.AccessKey.AccessKeyId,
			}); err != nil {
				return err
			}
			return nil
		}()
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	deletedKeyIds := []*string{}
	// delete the old access keys
	for _, k := range keys {
		_, err = iamService.DeleteAccessKey(r.Context(), &iam.DeleteAccessKeyInput{UserName: aws.String(user), AccessKeyId: k.AccessKeyId})
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

// UserListHandler lists the users for a bucket.  It tries to return the members of the
// bucket admin group: <<bucket>>-BktAdmGrp, but if that returns an error, it looks for
// a user with the same name as the bucket and returns that if it exists, otherwise it
// returns an error code.
func (s *server) UserListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	bucket := vars["bucket"]

	iamService, ok := s.iamServices[account]
	if !ok {
		msg := fmt.Sprintf("IAM service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	// TODO: if we support more than admins, we need to add those group names here
	groupName := fmt.Sprintf("%s-BktAdmGrp", bucket)
	// get a list of users in a group
	users, err := iamService.ListGroupUsers(r.Context(), &iam.GetGroupInput{GroupName: aws.String(groupName)})
	if err != nil {
		// check if there is a user with the same name as the bucket to support legacy buckets
		user, err := iamService.GetUser(r.Context(), &iam.GetUserInput{UserName: aws.String(bucket)})
		if err != nil {
			handleError(w, err)
			return
		}

		users = []*iam.User{user.User}
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
// users for a bucket's [group] and then comparing that to the passed user.  This would be more efficient if we
// just GetUser for the passed in user, but then we can't be sure it's associated with the bucket.
func (s *server) UserShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	bucket := vars["bucket"]
	user := vars["user"]

	iamService, ok := s.iamServices[account]
	if !ok {
		msg := fmt.Sprintf("IAM service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	// TODO: if we support more than admins, we need to add those group names here
	groupName := fmt.Sprintf("%s-BktAdmGrp", bucket)
	// get a list of users in a group
	users, err := iamService.ListGroupUsers(r.Context(), &iam.GetGroupInput{GroupName: aws.String(groupName)})
	if err != nil {
		// check if there is a user with the same name as the bucket to support legacy buckets
		u, err := iamService.GetUser(r.Context(), &iam.GetUserInput{UserName: aws.String(bucket)})
		if err != nil {
			handleError(w, err)
			return
		}

		users = []*iam.User{u.User}
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

	log.Warnf("requested user %s does not belong to bucket %s", user, bucket)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte{})
}
