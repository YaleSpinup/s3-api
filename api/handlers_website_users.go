package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/YaleSpinup/apierror"
	iamapi "github.com/YaleSpinup/s3-api/iam"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// WebsiteUserCreateHandler creates a new user for a website
func (s *server) WebsiteUserCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	website := vars["website"]

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
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create user input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// setup err var, rollback function list and defer execution, note that we depend on the err variable defined above this
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	userOutput, err := iamService.CreateUser(r.Context(), req.User)
	if err != nil {
		msg := fmt.Sprintf("failed to create user for website %s: %s", website, err)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// wait for the user to exist
	err = retry(3, 2*time.Second, func() error {
		log.Infof("checking if user exists before continuing: %s", aws.StringValue(userOutput.User.UserName))
		out, err := iamService.GetUser(r.Context(), &iam.GetUserInput{
			UserName: userOutput.User.UserName,
		})
		if err != nil {
			return err
		}

		log.Debugf("got user output: %s", awsutil.Prettify(out))
		return nil
	})

	if err != nil {
		msg := fmt.Sprintf("failed to create user %s for website %s: timeout waiting for create %s", aws.StringValue(req.User.UserName), website, err)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append user delete to rollback tasks
	rbfunc := func(ctx context.Context) error {
		if err := iamService.DeleteUser(r.Context(), &iam.DeleteUserInput{UserName: req.User.UserName}); err != nil {
			return err
		}
		return nil
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	keyOutput, err := iamService.CreateAccessKey(r.Context(), &iam.CreateAccessKeyInput{UserName: userOutput.User.UserName})
	if err != nil {
		msg := fmt.Sprintf("failed to create access key for user: %s, website %s", aws.StringValue(userOutput.User.UserName), website)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// append access key delete to rollback tasks
	rbfunc = func(ctx context.Context) error {
		if err := iamService.DeleteAccessKey(r.Context(), &iam.DeleteAccessKeyInput{
			UserName:    keyOutput.AccessKey.UserName,
			AccessKeyId: keyOutput.AccessKey.AccessKeyId,
		}); err != nil {
			return err
		}
		return nil
	}
	rollBackTasks = append(rollBackTasks, rbfunc)

	groupNames := req.Groups
	if groupNames == nil {
		groupNames = []string{"BktAdmGrp"}
	}

	for _, group := range groupNames {
		groupName := fmt.Sprintf("%s-%s", website, group)
		if err = iamService.AddUserToGroup(r.Context(), &iam.AddUserToGroupInput{
			UserName:  userOutput.User.UserName,
			GroupName: aws.String(groupName),
		}); err != nil {
			msg := fmt.Sprintf("failed to add user: %s to group %s for website %s", aws.StringValue(userOutput.User.UserName), group, website)
			handleError(w, errors.Wrap(err, msg))
			return
		}

		// append detach group to rollback funciton
		rbfunc = func(ctx context.Context) error {
			if err := iamService.RemoveUserFromGroup(r.Context(), &iam.RemoveUserFromGroupInput{
				UserName:  keyOutput.AccessKey.UserName,
				GroupName: aws.String(groupName),
			}); err != nil {
				return err
			}
			return nil
		}
		rollBackTasks = append(rollBackTasks, rbfunc)
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
