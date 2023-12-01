package iam

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/iam"
)

var testGroup = iam.Group{
	Arn:        aws.String("arn:aws:iam::12345678910:group/testgroup"),
	CreateDate: &testTime,
	GroupId:    aws.String("TESTGROUPID123"),
	GroupName:  aws.String("testgroup"),
	Path:       aws.String("/"),
}

var testGroupPolicy1 = iam.AttachedPolicy{
	PolicyArn:  aws.String("arn:aws:iam::12345678910:group/testpolicy1"),
	PolicyName: aws.String("testpolicy1"),
}

var testGroupPolicy2 = iam.AttachedPolicy{
	PolicyArn:  aws.String("arn:aws:iam::12345678910:group/testpolicy2"),
	PolicyName: aws.String("testpolicy2"),
}

var testGroupPolicy3 = iam.AttachedPolicy{
	PolicyArn:  aws.String("arn:aws:iam::12345678910:group/testpolicy3"),
	PolicyName: aws.String("testpolicy3"),
}

var testGroupPolicies1 = []*iam.AttachedPolicy{&testGroupPolicy1, &testGroupPolicy2, &testGroupPolicy3}

type TestFormatGroupNameInput struct {
	Base  string
	Path  string
	Group string
}

type TestFormatGroupNameExpected string
type TestFormatGroupNameSet struct {
	Input    TestFormatGroupNameInput
	Expected TestFormatGroupNameExpected
}

var testFormatGroupNameInputs = []TestFormatGroupNameSet{
	{
		Input: TestFormatGroupNameInput{
			Base:  "testwebsite.yalepages.org",
			Path:  "/",
			Group: "BktAdmGrp",
		},
		Expected: "testwebsite.yalepages.org-BktAdmGrp",
	},
	{
		Input: TestFormatGroupNameInput{
			Base:  "thesite.yalepages.org",
			Path:  "/spinup",
			Group: "BktAdmGrp",
		},
		Expected: "thesite.yalepages.org-spinup-BktAdmGrp",
	},
	{
		Input: TestFormatGroupNameInput{
			Base:  "thesite.yalepages.org",
			Path:  "/one/two/three",
			Group: "BktAdmGrp",
		},
		Expected: "thesite.yalepages.org-one_two_three-BktAdmGrp",
	},
}

var testListGroupsDataMarkers = []string{
	"mymarker1",
	"mymarker2",
	"mymarker3",
}

var testListGroupsData = []*iam.Group{
	{
		Arn:        aws.String("arn:aws:iam::12345678910:group/testsite.yalepages.org-spinup-BktAdmGrp"),
		CreateDate: &testTime,
		GroupId:    aws.String("1234"),
		GroupName:  aws.String("testsite.yalepages.org-spinup-BktAdmGrp"),
		Path:       aws.String("/spinup/"),
	},
	{
		Arn:        aws.String("arn:aws:iam::12345678910:group/testsite.yalepages.org-BktAdmGrp"),
		CreateDate: &testTime,
		GroupId:    aws.String("1234"),
		GroupName:  aws.String("testsite.yalepages.org-BktAdmGrp"),
		Path:       aws.String("/"),
	},
	{
		Arn:        aws.String("arn:aws:iam::12345678910:group/anothersite.yalepages.org-BktAdmGrp"),
		CreateDate: &testTime,
		GroupId:    aws.String("1234"),
		GroupName:  aws.String("anothersite.yalepages.org-BktAdmGrp"),
		Path:       aws.String("/"),
	},
}

var testListGroupsExpected = []*iam.Group{
	{
		Arn:        aws.String("arn:aws:iam::12345678910:group/testsite.yalepages.org-spinup-BktAdmGrp"),
		CreateDate: &testTime,
		GroupId:    aws.String("1234"),
		GroupName:  aws.String("testsite.yalepages.org-spinup-BktAdmGrp"),
		Path:       aws.String("/spinup/"),
	},
	{
		Arn:        aws.String("arn:aws:iam::12345678910:group/testsite.yalepages.org-BktAdmGrp"),
		CreateDate: &testTime,
		GroupId:    aws.String("1234"),
		GroupName:  aws.String("testsite.yalepages.org-BktAdmGrp"),
		Path:       aws.String("/"),
	},
}

var testListGroupsExpected2 = []*iam.Group{
	{
		Arn:        aws.String("arn:aws:iam::12345678910:group/anothersite.yalepages.org-BktAdmGrp"),
		CreateDate: &testTime,
		GroupId:    aws.String("1234"),
		GroupName:  aws.String("anothersite.yalepages.org-BktAdmGrp"),
		Path:       aws.String("/"),
	},
}

var testListGroupsExpected3 []*iam.Group

var testListGroupsInput = &iam.ListGroupsInput{}

var testUser1 = iam.User{
	Arn:        aws.String("arn:aws:iam::12345678910:user/testuser1"),
	CreateDate: &testTime,
	// Tags []*Tag `type:"list"`,
	UserId:   aws.String("TESTUSERID123"),
	UserName: aws.String("testuser1"),
	Path:     aws.String("/"),
}

var testUser2 = iam.User{
	Arn:        aws.String("arn:aws:iam::12345678910:user/testuser2"),
	CreateDate: &testTime,
	// Tags []*Tag `type:"list"`,
	UserId:   aws.String("TESTUSERID223"),
	UserName: aws.String("testuser2"),
	Path:     aws.String("/"),
}

var testUser3 = iam.User{
	Arn:        aws.String("arn:aws:iam::12345678910:user/testuser3"),
	CreateDate: &testTime,
	// Tags []*Tag `type:"list"`,
	UserId:   aws.String("TESTUSERID323"),
	UserName: aws.String("testuser3"),
	Path:     aws.String("/"),
}

var testUsers1 = []*iam.User{&testUser1, &testUser2, &testUser3}

func (m *mockIAMClient) ListGroupsWithContext(ctx context.Context, input *iam.ListGroupsInput, opts ...request.Option) (*iam.ListGroupsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if input.Marker == nil && input.MaxItems == nil {
		return &iam.ListGroupsOutput{Groups: testListGroupsData}, nil
	}

	if input.Marker == nil && input.MaxItems != nil {
		maxItems := aws.Int64Value(input.MaxItems) << 32
		itemsLen := len(testListGroupsData)
		g := int(maxItems)

		var groups []*iam.Group

		if g > itemsLen {
			g = itemsLen
		}

		for i := 0; i < g; i++ {
			groups = append(groups, testListGroupsData[i])
		}

		return &iam.ListGroupsOutput{Groups: groups}, nil
	}

	if input.Marker != nil && input.MaxItems != nil {
		markerIndex := -1
		startIndex := -1
		itemsLen := len(testListGroupsData)

		for i := 0; i < len(testListGroupsDataMarkers); i++ {
			m := testListGroupsDataMarkers[i]
			if m == *input.Marker {
				markerIndex = i
			}
		}

		if markerIndex == -1 {
			return &iam.ListGroupsOutput{}, nil
		}

		startIndex = markerIndex
		maxItems := aws.Int64Value(input.MaxItems) << 32
		itemsLen = len(testListGroupsData) - startIndex
		g := int(maxItems)

		var groups []*iam.Group

		if g > itemsLen {
			g = itemsLen
		}

		for i := startIndex; i < g; i++ {
			groups = append(groups, testListGroupsData[i])
		}

		return &iam.ListGroupsOutput{Groups: groups}, nil
	}

	return &iam.ListGroupsOutput{}, nil
}

func (m *mockIAMClient) CreateGroupWithContext(ctx context.Context, input *iam.CreateGroupInput, opts ...request.Option) (*iam.CreateGroupOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.CreateGroupOutput{Group: &testGroup}, nil
}

func (m *mockIAMClient) DeleteGroupWithContext(ctx context.Context, input *iam.DeleteGroupInput, opts ...request.Option) (*iam.DeleteGroupOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.DeleteGroupOutput{}, nil
}

func (m *mockIAMClient) AttachGroupPolicyWithContext(ctx context.Context, input *iam.AttachGroupPolicyInput, opts ...request.Option) (*iam.AttachGroupPolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.AttachGroupPolicyOutput{}, nil
}

func (m *mockIAMClient) DetachGroupPolicyWithContext(ctx context.Context, input *iam.DetachGroupPolicyInput, opts ...request.Option) (*iam.DetachGroupPolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.DetachGroupPolicyOutput{}, nil
}

func (m *mockIAMClient) ListAttachedGroupPoliciesWithContext(ctx context.Context, input *iam.ListAttachedGroupPoliciesInput, opts ...request.Option) (*iam.ListAttachedGroupPoliciesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.ListAttachedGroupPoliciesOutput{AttachedPolicies: testGroupPolicies1}, nil
}

func (m *mockIAMClient) GetGroupWithContext(ctx context.Context, input *iam.GetGroupInput, opts ...request.Option) (*iam.GetGroupOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.GetGroupOutput{Group: &testGroup, Users: testUsers1}, nil
}

func TestFormatGroupName(t *testing.T) {
	for _, set := range testFormatGroupNameInputs {
		actual := FormatGroupName(set.Input.Base, set.Input.Path, set.Input.Group)
		expected := string(set.Expected)

		if actual != expected {
			t.Errorf("actual %s did not match expected %s", actual, expected)
		}
	}
}

func TestListGroups(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	listResult, err := i.ListGroups(context.TODO(), &iam.ListGroupsInput{}, "testsite.yalepages.org")
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(listResult, testListGroupsExpected) {
		t.Errorf("expected %+v, got %+v", testListGroupsExpected, listResult)
	}

	listResultMaxItemsConstrain, err := i.ListGroups(context.TODO(), &iam.ListGroupsInput{MaxItems: aws.Int64(1)}, "testsite.yalepages.org")
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(listResultMaxItemsConstrain, testListGroupsExpected) {
		t.Errorf("expected %+v, got %+v", testListGroupsExpected, listResultMaxItemsConstrain)
	}

	listResult, err = i.ListGroups(context.TODO(), &iam.ListGroupsInput{}, "anothersite.yalepages.org")
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(listResult, testListGroupsExpected2) {
		t.Errorf("expected %+v, got %+v", testListGroupsExpected2, listResult)
	}

	listResult, err = i.ListGroups(context.TODO(), &iam.ListGroupsInput{}, "foo.yalepages.org")
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(listResult, testListGroupsExpected3) {
		t.Errorf("expected %+v, got %+v", testListGroupsExpected3, listResult)
	}
}

func TestCreateGroup(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := &testGroup
	out, err := i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.CreateGroup(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty group name
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeEntityAlreadyExistsException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeEntityAlreadyExistsException, "already exists", nil)
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failed", nil)
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.CreateGroup(context.TODO(), &iam.CreateGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteGroup(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	err := i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// test nil input
	err = i.DeleteGroup(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty group name
	err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeDeleteConflictException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "delete conflict", nil)
	err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "service failed", nil)
	err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeEntityAlreadyExistsException, "entity already exists", nil)
	err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	err = i.DeleteGroup(context.TODO(), &iam.DeleteGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestAttachGroupPolicy(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	err := i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// test nil input
	err = i.AttachGroupPolicy(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty group name and empty policy arn
	err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeInvalidInputException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeInvalidInputException, "limit exceeded", nil)
	err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodePolicyNotAttachableException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodePolicyNotAttachableException, "limit exceeded", nil)
	err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "limit exceeded", nil)
	err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "entity already exists", nil)
	err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	err = i.AttachGroupPolicy(context.TODO(), &iam.AttachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDetachGroupPolicy(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	err := i.DetachGroupPolicy(context.TODO(), &iam.DetachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// test nil input
	err = i.DetachGroupPolicy(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty group name and empty policy arn
	err = i.DetachGroupPolicy(context.TODO(), &iam.DetachGroupPolicyInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	err = i.DetachGroupPolicy(context.TODO(), &iam.DetachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "limit exceeded", nil)
	err = i.DetachGroupPolicy(context.TODO(), &iam.DetachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeInvalidInputException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeInvalidInputException, "limit exceeded", nil)
	err = i.DetachGroupPolicy(context.TODO(), &iam.DetachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "limit exceeded", nil)
	err = i.DetachGroupPolicy(context.TODO(), &iam.DetachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "entity already exists", nil)
	err = i.DetachGroupPolicy(context.TODO(), &iam.DetachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	err = i.DetachGroupPolicy(context.TODO(), &iam.DetachGroupPolicyInput{
		GroupName: aws.String("testgroup"),
		PolicyArn: aws.String("arn:aws:iam::12345678910:policy/testPolicy"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestListGroupPolicies(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := testGroupPolicies1
	out, err := i.ListGroupPolicies(context.TODO(), &iam.ListAttachedGroupPoliciesInput{GroupName: aws.String("testgroup")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.ListGroupPolicies(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty group name
	_, err = i.ListGroupPolicies(context.TODO(), &iam.ListAttachedGroupPoliciesInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	_, err = i.ListGroupPolicies(context.TODO(), &iam.ListAttachedGroupPoliciesInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeInvalidInputException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeInvalidInputException, "limit exceeded", nil)
	_, err = i.ListGroupPolicies(context.TODO(), &iam.ListAttachedGroupPoliciesInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "limit exceeded", nil)
	_, err = i.ListGroupPolicies(context.TODO(), &iam.ListAttachedGroupPoliciesInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "entity already exists", nil)
	_, err = i.ListGroupPolicies(context.TODO(), &iam.ListAttachedGroupPoliciesInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.ListGroupPolicies(context.TODO(), &iam.ListAttachedGroupPoliciesInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestListGroupUsers(t *testing.T) {
	i := IAM{
		Service:                newMockIAMClient(t, nil),
		DefaultS3BucketActions: []string{"gti", "golfr", "jetta", "passat"},
		DefaultS3ObjectActions: []string{"blue", "green", "yellow", "red"},
	}

	// test success
	expected := testUsers1
	out, err := i.ListGroupUsers(context.TODO(), &iam.GetGroupInput{GroupName: aws.String("testgroup")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.ListGroupUsers(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty group name
	_, err = i.ListGroupUsers(context.TODO(), &iam.GetGroupInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	_, err = i.ListGroupUsers(context.TODO(), &iam.GetGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "limit exceeded", nil)
	_, err = i.ListGroupUsers(context.TODO(), &iam.GetGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "entity already exists", nil)
	_, err = i.ListGroupUsers(context.TODO(), &iam.GetGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.ListGroupUsers(context.TODO(), &iam.GetGroupInput{GroupName: aws.String("testgroup")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestGetGroup(t *testing.T) {
	i := IAM{
		Service: newMockIAMClient(t, nil),
	}

	// test success
	expected := &testGroup
	out, err := i.GetGroup(context.TODO(), "testgroup")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test empty group name
	_, err = i.GetGroup(context.TODO(), "")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	_, err = i.GetGroup(context.TODO(), "testgroup")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "limit exceeded", nil)
	_, err = i.GetGroup(context.TODO(), "testgroup")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "entity already exists", nil)
	_, err = i.GetGroup(context.TODO(), "testgroup")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up!")
	_, err = i.GetGroup(context.TODO(), "testgroup")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
