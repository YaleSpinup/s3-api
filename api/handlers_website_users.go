package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	s3api "github.com/YaleSpinup/s3-api/s3"
	"github.com/aws/aws-sdk-go/service/s3"
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
	policy, err := generatePolicy("iam:*", "s3:*")
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
	s3Service := s3api.NewSession(session.Session, s.account, s.mapToAccountName(accountId))

	// REQUEST
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

	groupNames := req.Groups
	if groupNames == nil {
		groupNames = []string{"BktAdmGrp"}
	}

	path := "/"
	if aws.StringValue(req.User.Path) != "" {
		path = aws.StringValue(req.User.Path)
	}

	for _, group := range groupNames {
		groupName := iamapi.FormatGroupName(website, path, group)

		_, err := iamService.GetGroup(r.Context(), groupName)
		if err != nil {
			if aerr, ok := err.(apierror.Error); ok && aerr.Code == apierror.ErrNotFound {
				var rbTasks []rollbackFunc
				rbTasks, err = s.CreateWebsiteBucketPolicy(r.Context(), iamService, website, path, group)
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
			msg := fmt.Sprintf("failed to add user: %s to group %s for website %s", aws.StringValue(userOutput.User.UserName), group, website)
			handleError(w, errors.Wrap(err, msg))
			return
		}

		if path == "/" && group == "BktAdmGrp" {
			webGroupName := iamapi.FormatGroupName(website, path, "WebAdmGrp")

			if err = iamService.AddUserToGroup(r.Context(), &iam.AddUserToGroupInput{
				UserName:  userOutput.User.UserName,
				GroupName: aws.String(webGroupName),
			}); err != nil {
				msg := fmt.Sprintf("failed to add user: %s to group %s for website %s", aws.StringValue(userOutput.User.UserName), "WebAdmGrp", website)
				handleError(w, errors.Wrap(err, msg))
				return
			}
		}

		// append detach group to rollback funciton
		rbfunc = func(ctx context.Context) error {
			if err := iamService.RemoveUserFromGroup(r.Context(), &iam.RemoveUserFromGroupInput{
				UserName:  userOutput.User.UserName,
				GroupName: aws.String(groupName),
			}); err != nil {
				return err
			}
			return nil
		}
		rollBackTasks = append(rollBackTasks, rbfunc)
	}

	if path != "/" {
		hasIndexFile, err := s3Service.HasObject(r.Context(), &s3.GetObjectInput{
			Bucket: aws.String(website),
			Key:    aws.String(path + "index.html"),
		})
		if !hasIndexFile || err != nil {
			indexMessage := "Hello, " + website + path + "!"
			if _, err = s3Service.CreateObject(r.Context(), &s3.PutObjectInput{
				Bucket:      aws.String(website),
				Body:        bytes.NewReader([]byte(indexMessage)),
				ContentType: aws.String("text/html"),
				Key:         aws.String(path + "index.html"),
				Tagging:     aws.String("yale:spinup=true"),
			}); err != nil {
				msg := fmt.Sprintf("failed to create default index file for website %s: %s", website, err.Error())
				handleError(w, errors.Wrap(err, msg))
				return
			}
		}
		// write index file
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

// WebsiteUserShowHandler gets and returns details of a bucket user.  This is accomplished by getting all of the
// users for a bucket's management groups and then comparing that to the passed user.  This would be more
// efficient if we just GetUser for the passed in user, but then we can't be sure it's associated with the
// bucket.
func (s *server) WebsiteUserShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	accountId := s.mapAccountNumber(vars["account"])
	bucket := vars["bucket"]
	user := vars["user"]
	path := iamapi.GetUsernamePath(bucket, user)

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
		log.Debugf("formatting group name with parts | bucket: %s, path: %s, group: %s", bucket, path, g)
		groupName := iamapi.FormatGroupName(bucket, path, g)
		log.Debugf("list group users for group name: %s", groupName)
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
