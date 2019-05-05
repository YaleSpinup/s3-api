package route53

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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
		DNSName:      aws.String("abcdefg1234567.cloudfront.net"),
		HostedZoneId: aws.String("Z2FDTNDATAQYW2"),
	},
	Name: aws.String("foobar.hyper.converged"),
	Type: aws.String("A"),
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

func TestCreateRecord(t *testing.T) {
	r := Route53{
		Service: newmockRoute53Client(t, nil),
		Domains: map[string]common.Domain{
			"hyper.converged": common.Domain{
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
