package cloudfront

import (
	"context"
	"reflect"
	"testing"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/pkg/errors"
)

func (m *mockCloudFrontClient) CreateDistributionWithTagsWithContext(ctx context.Context, input *cloudfront.CreateDistributionWithTagsInput, opts ...request.Option) (*cloudfront.CreateDistributionWithTagsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &cloudfront.CreateDistributionWithTagsOutput{Distribution: &cloudfront.Distribution{}}, nil
}

func (m *mockCloudFrontClient) ListDistributionsWithContext(ctx context.Context, input *cloudfront.ListDistributionsInput, opts ...request.Option) (*cloudfront.ListDistributionsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &cloudfront.ListDistributionsOutput{DistributionList: &cloudfront.DistributionList{
		Items: []*cloudfront.DistributionSummary{
			&cloudfront.DistributionSummary{},
		},
	}}, nil
}

func TestCreateDistribution(t *testing.T) {
	c := CloudFront{
		Service: newmockCloudFrontClient(t, nil),
		Domains: map[string]common.Domain{
			"hyper.converged": common.Domain{
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
		WebsiteEndpoint: "s3-website-us-east-1.amazonaws.com",
	}

	distConfig, err := c.DefaultWebsiteDistributionConfig("foobar.hyper.converged")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	tags := &cloudfront.Tags{
		Items: []*cloudfront.Tag{
			&cloudfront.Tag{
				Key:   aws.String("key1"),
				Value: aws.String("value1"),
			},
		},
	}

	// test success
	expected := &cloudfront.Distribution{}
	out, err := c.CreateDistribution(context.TODO(), distConfig, tags)
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = c.CreateDistribution(context.TODO(), nil, nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeInconsistentQuantities "InconsistentQuantities"
	// The value of Quantity and the size of Items don't match.
	c.Service.(*mockCloudFrontClient).err = awserr.New(cloudfront.ErrCodeInconsistentQuantities, "InconsistentQuantities", nil)
	_, err = c.CreateDistribution(context.TODO(), distConfig, tags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeInvalidArgument "InvalidArgument"
	// The argument is invalid.
	c.Service.(*mockCloudFrontClient).err = awserr.New(cloudfront.ErrCodeInvalidArgument, "InvalidArgument", nil)
	_, err = c.CreateDistribution(context.TODO(), distConfig, tags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeNoSuchFieldLevelEncryptionProfile "NoSuchFieldLevelEncryptionProfile"
	// The specified profile for field-level encryption doesn't exist.
	c.Service.(*mockCloudFrontClient).err = awserr.New(cloudfront.ErrCodeNoSuchFieldLevelEncryptionProfile, "InvalidArgument", nil)
	_, err = c.CreateDistribution(context.TODO(), distConfig, tags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeFieldLevelEncryptionConfigAlreadyExists "FieldLevelEncryptionConfigAlreadyExists"
	// The specified configuration for field-level encryption already exists.
	c.Service.(*mockCloudFrontClient).err = awserr.New(cloudfront.ErrCodeFieldLevelEncryptionConfigAlreadyExists, "InvalidArgument", nil)
	_, err = c.CreateDistribution(context.TODO(), distConfig, tags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeTooManyFieldLevelEncryptionConfigs "TooManyFieldLevelEncryptionConfigs"
	// The maximum number of configurations for field-level encryption have been
	// created.
	c.Service.(*mockCloudFrontClient).err = awserr.New(cloudfront.ErrCodeTooManyFieldLevelEncryptionConfigs, "InvalidArgument", nil)
	_, err = c.CreateDistribution(context.TODO(), distConfig, tags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeTooManyFieldLevelEncryptionQueryArgProfiles "TooManyFieldLevelEncryptionQueryArgProfiles"
	// The maximum number of query arg profiles for field-level encryption have
	// been created.
	c.Service.(*mockCloudFrontClient).err = awserr.New(cloudfront.ErrCodeTooManyFieldLevelEncryptionQueryArgProfiles, "InvalidArgument", nil)
	_, err = c.CreateDistribution(context.TODO(), distConfig, tags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeTooManyFieldLevelEncryptionContentTypeProfiles "TooManyFieldLevelEncryptionContentTypeProfiles"
	// The maximum number of content type profiles for field-level encryption have
	// been created.
	c.Service.(*mockCloudFrontClient).err = awserr.New(cloudfront.ErrCodeTooManyFieldLevelEncryptionContentTypeProfiles, "InvalidArgument", nil)
	_, err = c.CreateDistribution(context.TODO(), distConfig, tags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// * ErrCodeQueryArgProfileEmpty "QueryArgProfileEmpty"
	// No profile specified for the field-level encryption query argument.
	c.Service.(*mockCloudFrontClient).err = awserr.New(cloudfront.ErrCodeQueryArgProfileEmpty, "InvalidArgument", nil)
	_, err = c.CreateDistribution(context.TODO(), distConfig, tags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	c.Service.(*mockCloudFrontClient).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	_, err = c.CreateDistribution(context.TODO(), distConfig, tags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	c.Service.(*mockCloudFrontClient).err = errors.New("things blowing up!")
	_, err = c.CreateDistribution(context.TODO(), distConfig, tags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestListDistribution(t *testing.T) {
	c := CloudFront{
		Service: newmockCloudFrontClient(t, nil),
		Domains: map[string]common.Domain{
			"hyper.converged": common.Domain{
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
		WebsiteEndpoint: "s3-website-us-east-1.amazonaws.com",
	}

	// test success
	expected := []*cloudfront.DistributionSummary{
		&cloudfront.DistributionSummary{},
	}
	out, err := c.ListDistributions(context.TODO())
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// * ErrCodeInvalidArgument "InvalidArgument"
	// The argument is invalid.
	c.Service.(*mockCloudFrontClient).err = awserr.New(cloudfront.ErrCodeInvalidArgument, "InvalidArgument", nil)
	_, err = c.ListDistributions(context.TODO())
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	c.Service.(*mockCloudFrontClient).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	_, err = c.ListDistributions(context.TODO())
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	c.Service.(*mockCloudFrontClient).err = errors.New("things blowing up!")
	_, err = c.ListDistributions(context.TODO())
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
