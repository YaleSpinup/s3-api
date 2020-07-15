package cloudfront

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var testDistribution1 = &cloudfront.DistributionSummary{
	ARN: aws.String("arn:aws:cloudfront::1234567890:distribution/AAAABBBBCCCCDDDD"),
	Aliases: &cloudfront.Aliases{
		Items:    []*string{aws.String("foobar1.bulldogs.cloud")},
		Quantity: aws.Int64(1),
	},
	Comment: aws.String("foobar1.bulldogs.cloud"),
	DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
		TargetOriginId: aws.String("foobar1.bulldogs.cloud"),
	},
	DomainName: aws.String("zzzzzzzzzzzzz.cloudfront.net"),
	Enabled:    aws.Bool(true),
	Id:         aws.String("AAAABBBBCCCCDDDD"),
	Origins: &cloudfront.Origins{
		Items: []*cloudfront.Origin{
			{
				DomainName: aws.String("foobar1.bulldogs.cloud.s3-website-us-east-1.amazonaws.com"),
				Id:         aws.String("foobar1.bulldogs.cloud"),
			},
		},
		Quantity: aws.Int64(1),
	},
	Status: aws.String("Deployed"),
}
var testDistribution2 = &cloudfront.DistributionSummary{
	ARN: aws.String("arn:aws:cloudfront::1234567890:distribution/EEEEFFFFGGGGHHHH"),
	Aliases: &cloudfront.Aliases{
		Items:    []*string{aws.String("foobar2.bulldogs.cloud")},
		Quantity: aws.Int64(1),
	},
	Comment: aws.String("foobar1.bulldogs.cloud"),
	DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
		TargetOriginId: aws.String("foobar2.bulldogs.cloud"),
	},
	DomainName: aws.String("yyyyyyyyyyyyyy.cloudfront.net"),
	Enabled:    aws.Bool(true),
	Id:         aws.String("EEEEFFFFGGGGHHHH"),
	Origins: &cloudfront.Origins{
		Items: []*cloudfront.Origin{
			{
				DomainName: aws.String("foobar2.bulldogs.cloud.s3-website-us-east-1.amazonaws.com"),
				Id:         aws.String("foobar2.bulldogs.cloud"),
			},
		},
		Quantity: aws.Int64(1),
	},
	Status: aws.String("Deployed"),
}
var testDistribution3 = &cloudfront.DistributionSummary{
	ARN: aws.String("arn:aws:cloudfront::1234567890:distribution/IIIIJJJJKKKKLLLL"),
	Aliases: &cloudfront.Aliases{
		Items:    []*string{aws.String("foobar3.bulldogs.cloud")},
		Quantity: aws.Int64(1),
	},
	Comment: aws.String("foobar3.bulldogs.cloud"),
	DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
		TargetOriginId: aws.String("foobar3.bulldogs.cloud"),
	},
	DomainName: aws.String("xxxxxxxxxxxxxxxx.cloudfront.net"),
	Enabled:    aws.Bool(true),
	Id:         aws.String("IIIIJJJJKKKKLLLL"),
	Origins: &cloudfront.Origins{
		Items: []*cloudfront.Origin{
			{
				DomainName: aws.String("foobar3.bulldogs.cloud.s3-website-us-east-1.amazonaws.com"),
				Id:         aws.String("foobar3.bulldogs.cloud"),
			},
		},
		Quantity: aws.Int64(1),
	},
	Status: aws.String("Deployed"),
}

var testInvalidation = &cloudfront.Invalidation{
	CreateTime: aws.Time(time.Now()),
	Id:         aws.String("AABBCCDDEEFF"),
	InvalidationBatch: &cloudfront.InvalidationBatch{
		CallerReference: aws.String(uuid.New().String()),
		Paths: &cloudfront.Paths{
			Items: []*string{
				aws.String("/*"),
			},
			Quantity: aws.Int64(1),
		},
	},
	Status: aws.String("InProgress"),
}

var testTags = []*cloudfront.Tag{
	{
		Key:   aws.String("foo"),
		Value: aws.String("bar"),
	},
}

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
			{},
		},
	}}, nil
}

func (m *mockCloudFrontClient) ListDistributionsPagesWithContext(ctx context.Context, input *cloudfront.ListDistributionsInput, fn func(*cloudfront.ListDistributionsOutput, bool) bool, opts ...request.Option) error {
	if m.err != nil {
		return m.err
	}

	_ = fn(&cloudfront.ListDistributionsOutput{
		DistributionList: &cloudfront.DistributionList{
			IsTruncated: aws.Bool(false),
			Items: []*cloudfront.DistributionSummary{
				testDistribution1,
				testDistribution2,
				testDistribution3,
			},
			Marker:   nil,
			MaxItems: aws.Int64(100),
			Quantity: aws.Int64(3),
		},
	}, true)

	return nil
}

func (m *mockCloudFrontClient) GetDistributionConfigWithContext(ctx context.Context, input *cloudfront.GetDistributionConfigInput, opts ...request.Option) (*cloudfront.GetDistributionConfigOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	var dist *cloudfront.DistributionSummary
	for _, d := range []*cloudfront.DistributionSummary{testDistribution1, testDistribution2, testDistribution3} {
		if aws.StringValue(d.Id) == aws.StringValue(input.Id) {
			dist = d
			break
		}
	}

	if dist == nil {
		return nil, awserr.New(cloudfront.ErrCodeNoSuchDistribution, "Distribution Not Found", nil)
	}

	return &cloudfront.GetDistributionConfigOutput{
		DistributionConfig: &cloudfront.DistributionConfig{
			Aliases:              dist.Aliases,
			Comment:              dist.Comment,
			DefaultCacheBehavior: dist.DefaultCacheBehavior,
			Enabled:              dist.Enabled,
			Origins:              dist.Origins,
		},
		ETag: aws.String("ETAGETAGETAGETAG"),
	}, nil
}

func (m *mockCloudFrontClient) UpdateDistributionWithContext(ctx context.Context, input *cloudfront.UpdateDistributionInput, opts ...request.Option) (*cloudfront.UpdateDistributionOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	var dist *cloudfront.DistributionSummary
	for _, d := range []*cloudfront.DistributionSummary{testDistribution1, testDistribution2, testDistribution3} {
		if aws.StringValue(d.Id) == aws.StringValue(input.Id) {
			dist = d
			break
		}
	}

	if dist == nil {
		return nil, awserr.New(cloudfront.ErrCodeNoSuchDistribution, "Distribution Not Found", nil)
	}

	if aws.StringValue(input.IfMatch) != "ETAGETAGETAGETAG" {
		return nil, awserr.New(cloudfront.ErrCodeInvalidIfMatchVersion, "ETag missing or invalid", nil)
	}

	return &cloudfront.UpdateDistributionOutput{
		Distribution: &cloudfront.Distribution{
			ARN:                dist.ARN,
			DistributionConfig: input.DistributionConfig,
			Status:             aws.String("InProgress"),
		},
		ETag: aws.String("GATEGATEGATEGATE"),
	}, nil
}

func (m *mockCloudFrontClient) DeleteDistributionWithContext(ctx context.Context, input *cloudfront.DeleteDistributionInput, opts ...request.Option) (*cloudfront.DeleteDistributionOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	var dist *cloudfront.DistributionSummary
	for _, d := range []*cloudfront.DistributionSummary{testDistribution1, testDistribution2, testDistribution3} {
		if aws.StringValue(d.Id) == aws.StringValue(input.Id) {
			dist = d
			break
		}
	}

	if dist == nil {
		return nil, awserr.New(cloudfront.ErrCodeNoSuchDistribution, "Distribution Not Found", nil)
	}

	if aws.StringValue(input.IfMatch) != "ETAGETAGETAGETAG" {
		return nil, awserr.New(cloudfront.ErrCodeInvalidIfMatchVersion, "ETag missing or invalid", nil)
	}

	return &cloudfront.DeleteDistributionOutput{}, nil
}

func (m *mockCloudFrontClient) TagResourceWithContext(ctx context.Context, input *cloudfront.TagResourceInput, opts ...request.Option) (*cloudfront.TagResourceOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, d := range []*cloudfront.DistributionSummary{testDistribution1, testDistribution2, testDistribution3} {
		if aws.StringValue(d.ARN) == aws.StringValue(input.Resource) {
			return &cloudfront.TagResourceOutput{}, nil
		}
	}

	return nil, awserr.New(cloudfront.ErrCodeNoSuchDistribution, "Distribution Not Found", nil)
}

func (m *mockCloudFrontClient) CreateInvalidationWithContext(ctx context.Context, input *cloudfront.CreateInvalidationInput, opts ...request.Option) (*cloudfront.CreateInvalidationOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.InvalidationBatch.Paths.Items[0]) == "errorme" {
		return nil, awserr.New(cloudfront.ErrCodeTooManyInvalidationsInProgress, "Too many invalidations!", nil)
	}

	return &cloudfront.CreateInvalidationOutput{
		Invalidation: testInvalidation,
		Location:     aws.String("https://cloudfront.amazonaws.com/2018-11-05/distribution/" + aws.StringValue(testInvalidation.Id) + "/invalidation/" + aws.StringValue(testInvalidation.Id)),
	}, nil
}

func (m *mockCloudFrontClient) ListTagsForResourceWithContext(ctx context.Context, input *cloudfront.ListTagsForResourceInput, opts ...request.Option) (*cloudfront.ListTagsForResourceOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &cloudfront.ListTagsForResourceOutput{
		Tags: &cloudfront.Tags{
			Items: testTags,
		},
	}, nil
}

func TestCreateDistribution(t *testing.T) {
	c := CloudFront{
		Service: newmockCloudFrontClient(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
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
			{
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
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
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

func TestDisableDistribution(t *testing.T) {
	c := CloudFront{
		Service: newmockCloudFrontClient(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
		WebsiteEndpoint: "s3-website-us-east-1.amazonaws.com",
	}

	tests := map[*cloudfront.DistributionSummary]*cloudfront.Distribution{
		testDistribution1: {
			ARN: testDistribution1.ARN,
			DistributionConfig: &cloudfront.DistributionConfig{
				Aliases:              testDistribution1.Aliases,
				Comment:              testDistribution1.Comment,
				DefaultCacheBehavior: testDistribution1.DefaultCacheBehavior,
				Origins:              testDistribution1.Origins,
				Enabled:              aws.Bool(false),
			},
			Status: aws.String("InProgress"),
		},
		testDistribution2: {
			ARN: testDistribution2.ARN,
			DistributionConfig: &cloudfront.DistributionConfig{
				Aliases:              testDistribution2.Aliases,
				Comment:              testDistribution2.Comment,
				DefaultCacheBehavior: testDistribution2.DefaultCacheBehavior,
				Origins:              testDistribution2.Origins,
				Enabled:              aws.Bool(false),
			},
			Status: aws.String("InProgress"),
		},
		testDistribution3: {
			ARN: testDistribution3.ARN,
			DistributionConfig: &cloudfront.DistributionConfig{
				Aliases:              testDistribution3.Aliases,
				Comment:              testDistribution3.Comment,
				DefaultCacheBehavior: testDistribution3.DefaultCacheBehavior,
				Origins:              testDistribution3.Origins,
				Enabled:              aws.Bool(false),
			},
			Status: aws.String("InProgress"),
		},
	}

	// test success
	for _, testDist := range []*cloudfront.DistributionSummary{testDistribution1, testDistribution2, testDistribution3} {
		expected := tests[testDist]
		out, err := c.DisableDistribution(context.TODO(), aws.StringValue(testDist.Id))
		if err != nil {
			t.Errorf("expected nil error, got: %s", err)
		}

		if !reflect.DeepEqual(out, expected) {
			t.Errorf("expected %+v, got %+v", expected, out)
		}
	}

	// test empty id input
	_, err := c.DisableDistribution(context.TODO(), "")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test not found id input
	_, err = c.DisableDistribution(context.TODO(), "notfoundid")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteDistribution(t *testing.T) {
	c := CloudFront{
		Service: newmockCloudFrontClient(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
		WebsiteEndpoint: "s3-website-us-east-1.amazonaws.com",
	}

	// test success
	tests := []*string{testDistribution1.Id, testDistribution2.Id, testDistribution3.Id}
	for _, testDist := range tests {
		err := c.DeleteDistribution(context.TODO(), aws.StringValue(testDist))
		if err != nil {
			t.Errorf("expected nil error, got: %s", err)
		}
	}

	// test empty id input
	err := c.DeleteDistribution(context.TODO(), "")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test not found id input
	err = c.DeleteDistribution(context.TODO(), "notfoundid")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestTagDistribution(t *testing.T) {
	c := CloudFront{
		Service: newmockCloudFrontClient(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
		WebsiteEndpoint: "s3-website-us-east-1.amazonaws.com",
	}

	testTags := &cloudfront.Tags{
		Items: []*cloudfront.Tag{
			{
				Key:   aws.String("foo"),
				Value: aws.String("bar"),
			},
		},
	}

	// test success
	tests := []*string{testDistribution1.ARN, testDistribution2.ARN, testDistribution3.ARN}
	for _, testDist := range tests {
		err := c.TagDistribution(context.TODO(), aws.StringValue(testDist), testTags)
		if err != nil {
			t.Errorf("expected nil error, got: %s", err)
		}
	}

	// test empty arn input
	err := c.TagDistribution(context.TODO(), "", testTags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test not found arn input
	err = c.TagDistribution(context.TODO(), "notfoundid", testTags)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestListDistribution(t *testing.T) {
	c := CloudFront{
		Service: newmockCloudFrontClient(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
		WebsiteEndpoint: "s3-website-us-east-1.amazonaws.com",
	}

	// test success
	expected := []*cloudfront.DistributionSummary{
		{},
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

func TestGetDistributionByName(t *testing.T) {
	c := CloudFront{
		Service: newmockCloudFrontClient(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
		WebsiteEndpoint: "s3-website-us-east-1.amazonaws.com",
	}

	// test success
	expected := testDistribution1
	out, err := c.GetDistributionByName(context.TODO(), "foobar1.bulldogs.cloud")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	_, err = c.GetDistributionByName(context.TODO(), "foobaz.bulldogs.cloud")
	if err == nil {
		t.Error("expected error for non-existing distribution, got nil")
	}

	// * ErrCodeInvalidArgument "InvalidArgument"
	// The argument is invalid.
	c.Service.(*mockCloudFrontClient).err = awserr.New(cloudfront.ErrCodeInvalidArgument, "InvalidArgument", nil)
	_, err = c.GetDistributionByName(context.TODO(), "fooerr.bulldogs.cloud")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	c.Service.(*mockCloudFrontClient).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	_, err = c.GetDistributionByName(context.TODO(), "fooerr.bulldogs.cloud")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	c.Service.(*mockCloudFrontClient).err = errors.New("things blowing up!")
	_, err = c.GetDistributionByName(context.TODO(), "fooerr.bulldogs.cloud")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

}

func TestInvalidateCache(t *testing.T) {
	c := CloudFront{
		Service: newmockCloudFrontClient(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
		WebsiteEndpoint: "s3-website-us-east-1.amazonaws.com",
	}

	for _, dist := range []*cloudfront.DistributionSummary{testDistribution1, testDistribution2, testDistribution3} {
		expected := &cloudfront.CreateInvalidationOutput{
			Invalidation: testInvalidation,
			Location:     aws.String("https://cloudfront.amazonaws.com/2018-11-05/distribution/" + aws.StringValue(testInvalidation.Id) + "/invalidation/" + aws.StringValue(testInvalidation.Id)),
		}
		out, err := c.InvalidateCache(context.TODO(), aws.StringValue(dist.Id), aws.StringValueSlice(testInvalidation.InvalidationBatch.Paths.Items))
		if err != nil {
			t.Errorf("expected nil error, got: %s", err)
		}

		if !reflect.DeepEqual(out, expected) {
			t.Errorf("expected %+v, got %+v", expected, out)
		}
	}

	_, err := c.InvalidateCache(context.TODO(), "", aws.StringValueSlice(testInvalidation.InvalidationBatch.Paths.Items))
	if err == nil {
		t.Error("expected error for non-existing distribution, got nil")
	}

	_, err = c.InvalidateCache(context.TODO(), "foobar3.bulldogs.cloud", []string{})
	if err == nil {
		t.Error("expected error for empty path list, got nil")
	}

	_, err = c.InvalidateCache(context.TODO(), "foobar3.bulldogs.cloud", []string{"errorme"})
	if err == nil {
		t.Error("expected error for empty path list, got nil")
	}
}

func TestListTags(t *testing.T) {
	c := CloudFront{
		Service: newmockCloudFrontClient(t, nil),
		Domains: map[string]*common.Domain{
			"hyper.converged": {
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
		WebsiteEndpoint: "s3-website-us-east-1.amazonaws.com",
	}

	// test success
	expected := testTags
	out, err := c.ListTags(context.TODO(), "arn:aws:cloudfront::12345678910:distribution/AAABBBCCCDDDEEE")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	_, err = c.ListTags(context.TODO(), "")
	if err == nil {
		t.Error("expected error for empty distribution id, got nil")
	}
}
