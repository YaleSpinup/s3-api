package s3api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gorilla/mux"
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

	// setup rollback function list and defer recovery and execution
	var rollBackTasks []func() error
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("recovering from panic: %s", err)
			executeRollBack(&rollBackTasks)
		}
	}()

	userOutput, err := iamService.CreateUser(r.Context(), &req)
	if err != nil {
		handleError(w, err)
		msg := fmt.Sprintf("failed to create user for bucket %s: %s", bucket, err)
		panic(msg)
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
		handleError(w, err)
		msg := fmt.Sprintf("failed to create access key for user: %s, bucket %s", aws.StringValue(userOutput.User.UserName), bucket)
		panic(msg)
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
		handleError(w, err)
		msg := fmt.Sprintf("failed to add user: %s to group %s for bucket %s", aws.StringValue(userOutput.User.UserName), groupName, bucket)
		panic(msg)
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
		_, err = iamService.RemoveUserFromGroup(r.Context(), &iam.RemoveUserFromGroupInput{UserName: aws.String(user), GroupName: g.GroupName})
		if err != nil {
			handleError(w, err)
			return
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
