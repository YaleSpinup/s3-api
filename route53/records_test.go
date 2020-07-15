package route53

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/route53"
)

var now = time.Now()
var testHostedZoneID = "ABCDEF12345"
var testResourceRecordSet = route53.ResourceRecordSet{
	AliasTarget: &route53.AliasTarget{
		DNSName:              aws.String("abcdefg1234567.cloudfront.net"),
		EvaluateTargetHealth: aws.Bool(false),
		HostedZoneId:         aws.String("Z2FDTNDATAQYW2"),
	},
	Name: aws.String("foobar.hyper.converged."),
	Type: aws.String("A"),
}
var testResourceRecordSet1 = route53.ResourceRecordSet{
	Name: aws.String("hyper.converged."),
	ResourceRecords: []*route53.ResourceRecord{
		{
			Value: aws.String("ns-1111.awsdns-01.com."),
		},
		{
			Value: aws.String("ns-2222.awsdns-02.org."),
		},
		{
			Value: aws.String("ns-3333.awsdns-03.net."),
		},
		{
			Value: aws.String("ns-4444.awsdns-04.co.uk."),
		},
	},
	TTL:  aws.Int64(172800),
	Type: aws.String("NS"),
}
var testResourceRecordSet2 = route53.ResourceRecordSet{
	Name: aws.String("_3344556677889900aabbccddeeffgghhiijjkk.hyper.converged."),
	ResourceRecords: []*route53.ResourceRecord{
		{
			Value: aws.String("_012345678910abcdefghijkl.mnopqrstuvwxyz0.acm-validations.aws."),
		},
	},
	TTL:  aws.Int64(300),
	Type: aws.String("CNAME"),
}
var testChangeInfo = route53.ChangeInfo{
	Comment:     aws.String("Test Change Info"),
	Id:          aws.String("abcdefg1234567"),
	Status:      aws.String("INSYNC"),
	SubmittedAt: &now,
}

func (m *mockRoute53Client) ChangeResourceRecordSetsWithContext(ctx context.Context, input *route53.ChangeResourceRecordSetsInput, opts ...request.Option) (*route53.ChangeResourceRecordSetsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if input.HostedZoneId == nil || aws.StringValue(input.HostedZoneId) != testHostedZoneID {
		msg := fmt.Sprintf("expected valid hosted (%s) zone id, got %s", testHostedZoneID, aws.StringValue(input.HostedZoneId))
		return nil, errors.New(msg)
	}

	if input.ChangeBatch == nil || len(input.ChangeBatch.Changes) == 0 {
		return nil, errors.New("expected valid batch of changes")
	}

	change := input.ChangeBatch.Changes[0]
	if !reflect.DeepEqual(change.ResourceRecordSet, &testResourceRecordSet) {
		msg := fmt.Sprintf("expected valid resource record set (%+v) zone id, got %+v", testResourceRecordSet, change.ResourceRecordSet)
		return nil, errors.New(msg)
	}

	return &route53.ChangeResourceRecordSetsOutput{
		ChangeInfo: &testChangeInfo,
	}, nil
}

func (m *mockRoute53Client) ListResourceRecordSetsWithContext(ctx context.Context, input *route53.ListResourceRecordSetsInput, opts ...request.Option) (*route53.ListResourceRecordSetsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if input.HostedZoneId == nil || aws.StringValue(input.HostedZoneId) != testHostedZoneID {
		msg := fmt.Sprintf("expected valid hosted (%s) zone id, got %s", testHostedZoneID, aws.StringValue(input.HostedZoneId))
		return nil, errors.New(msg)
	}

	allItems := []*route53.ResourceRecordSet{&testResourceRecordSet, &testResourceRecordSet1, &testResourceRecordSet2}

	max := 100
	if input.MaxItems != nil {
		m, err := strconv.ParseInt(aws.StringValue(input.MaxItems), 10, 64)
		if err != nil {
			return nil, err
		}
		max = int(m)
	}

	start := 0
	if input.StartRecordName != nil {
		// expect a parsable number for testing purposes
		s, err := strconv.ParseInt(aws.StringValue(input.StartRecordName), 10, 64)
		if err != nil {
			return nil, err
		}
		start = int(s)

		if start > len(allItems) {
			return nil, errors.New("starting record is greater than last record")
		}
	}

	end := max
	if end > len(allItems) {
		end = len(allItems)
	}

	truncated := false
	if start+max < len(allItems) {
		truncated = true
	}

	next := 0
	items := []*route53.ResourceRecordSet{}
	for i := start; i < end; i++ {
		items = append(items, allItems[i])
		next = i + 1
	}

	return &route53.ListResourceRecordSetsOutput{
		IsTruncated:        aws.Bool(truncated),
		MaxItems:           aws.String(strconv.Itoa(len(items))),
		NextRecordName:     aws.String(strconv.Itoa(next)),
		ResourceRecordSets: items,
	}, nil
}

func (m *mockRoute53Client) ListResourceRecordSetsPagesWithContext(ctx aws.Context, input *route53.ListResourceRecordSetsInput, fn func(*route53.ListResourceRecordSetsOutput, bool) bool, opts ...request.Option) error {
	if m.err != nil {
		return m.err
	}

	_ = fn(&route53.ListResourceRecordSetsOutput{
		IsTruncated: aws.Bool(false),
		ResourceRecordSets: []*route53.ResourceRecordSet{
			&testResourceRecordSet,
			&testResourceRecordSet1,
			&testResourceRecordSet2,
		},
		MaxItems: aws.String("100"),
	}, true)

	return nil
}

func TestCreateRecord(t *testing.T) {
	r := Route53{
		Service: newmockRoute53Client(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
	}

	// test success
	expected := &testChangeInfo
	out, err := r.CreateRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = r.CreateRecord(context.TODO(), testHostedZoneID, nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeNoSuchHostedZone "NoSuchHostedZone"
	// No hosted zone exists with the ID that you specified.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeNoSuchHostedZone, "NoSuchHostedZone", nil)
	_, err = r.CreateRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeNoSuchHealthCheck "NoSuchHealthCheck"
	// No health check exists with the specified ID.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeNoSuchHealthCheck, "NoSuchHealthCheck", nil)
	_, err = r.CreateRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeInvalidChangeBatch "InvalidChangeBatch"
	// This exception contains a list of messages that might contain one or more
	// error messages. Each error message indicates one error in the change batch.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeInvalidChangeBatch, "InvalidChangeBatch", nil)
	_, err = r.CreateRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeInvalidInput "InvalidInput"
	// The input is not valid.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeInvalidInput, "InvalidInput", nil)
	_, err = r.CreateRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodePriorRequestNotComplete "PriorRequestNotComplete"
	// If Amazon Route 53 can't process a request before the next request arrives,
	// it will reject subsequent requests for the same hosted zone and return an
	// HTTP 400 error (Bad request). If Route 53 returns this error repeatedly for
	// the same request, we recommend that you wait, in intervals of increasing
	// duration, before you try the request again.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodePriorRequestNotComplete, "PriorRequestNotComplete", nil)
	_, err = r.CreateRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	r.Service.(*mockRoute53Client).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	_, err = r.CreateRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	r.Service.(*mockRoute53Client).err = errors.New("things blowing up!")
	_, err = r.CreateRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteRecord(t *testing.T) {
	r := Route53{
		Service: newmockRoute53Client(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
	}

	// test success
	expected := &testChangeInfo
	out, err := r.DeleteRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = r.DeleteRecord(context.TODO(), testHostedZoneID, nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeNoSuchHostedZone "NoSuchHostedZone"
	// No hosted zone exists with the ID that you specified.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeNoSuchHostedZone, "NoSuchHostedZone", nil)
	_, err = r.DeleteRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeNoSuchHealthCheck "NoSuchHealthCheck"
	// No health check exists with the specified ID.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeNoSuchHealthCheck, "NoSuchHealthCheck", nil)
	_, err = r.DeleteRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeInvalidChangeBatch "InvalidChangeBatch"
	// This exception contains a list of messages that might contain one or more
	// error messages. Each error message indicates one error in the change batch.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeInvalidChangeBatch, "InvalidChangeBatch", nil)
	_, err = r.DeleteRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeInvalidInput "InvalidInput"
	// The input is not valid.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeInvalidInput, "InvalidInput", nil)
	_, err = r.DeleteRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodePriorRequestNotComplete "PriorRequestNotComplete"
	// If Amazon Route 53 can't process a request before the next request arrives,
	// it will reject subsequent requests for the same hosted zone and return an
	// HTTP 400 error (Bad request). If Route 53 returns this error repeatedly for
	// the same request, we recommend that you wait, in intervals of increasing
	// duration, before you try the request again.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodePriorRequestNotComplete, "PriorRequestNotComplete", nil)
	_, err = r.DeleteRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	r.Service.(*mockRoute53Client).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	_, err = r.DeleteRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	r.Service.(*mockRoute53Client).err = errors.New("things blowing up!")
	_, err = r.DeleteRecord(context.TODO(), testHostedZoneID, &testResourceRecordSet)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestListRecords(t *testing.T) {
	r := Route53{
		Service: newmockRoute53Client(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
	}

	// test success
	expected := []*route53.ResourceRecordSet{&testResourceRecordSet, &testResourceRecordSet1, &testResourceRecordSet2}
	out, err := r.ListRecords(context.TODO(), testHostedZoneID)
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// * ErrCodeNoSuchHostedZone "NoSuchHostedZone"
	// No hosted zone exists with the ID that you specified.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeNoSuchHostedZone, "NoSuchHostedZone", nil)
	_, err = r.ListRecords(context.TODO(), testHostedZoneID)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeInvalidInput "InvalidInput"
	// The input is not valid.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeInvalidInput, "InvalidInput", nil)
	_, err = r.ListRecords(context.TODO(), testHostedZoneID)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	r.Service.(*mockRoute53Client).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	_, err = r.ListRecords(context.TODO(), testHostedZoneID)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	r.Service.(*mockRoute53Client).err = errors.New("things blowing up!")
	_, err = r.ListRecords(context.TODO(), testHostedZoneID)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestGetRecordByName(t *testing.T) {
	r := Route53{
		Service: newmockRoute53Client(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
	}

	expected := &testResourceRecordSet
	out, err := r.GetRecordByName(context.TODO(), testHostedZoneID, "foobar.hyper.converged", "A")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	out, err = r.GetRecordByName(context.TODO(), testHostedZoneID, "foobar.hyper.converged.", "A")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	_, err = r.GetRecordByName(context.TODO(), testHostedZoneID, "foobaz.hyper.converged", "A")
	if err == nil {
		t.Error("expected error for non-existing record, got nil")
	}

	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test with wrong type
	_, err = r.GetRecordByName(context.TODO(), testHostedZoneID, "foobar.hyper.converged", "CNAME")
	if err == nil {
		t.Error("expected error got nil")
	}

	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// Test an error from the ListRecords call
	//
	// * ErrCodeNoSuchHostedZone "NoSuchHostedZone"
	// No hosted zone exists with the ID that you specified.
	r.Service.(*mockRoute53Client).err = awserr.New(route53.ErrCodeNoSuchHostedZone, "NoSuchHostedZone", nil)
	_, err = r.GetRecordByName(context.TODO(), testHostedZoneID, "some.other.host", "A")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	r.Service.(*mockRoute53Client).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	_, err = r.GetRecordByName(context.TODO(), testHostedZoneID, "some.other.host", "A")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	r.Service.(*mockRoute53Client).err = errors.New("things blowing up!")
	_, err = r.GetRecordByName(context.TODO(), testHostedZoneID, "some.other.host", "A")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
