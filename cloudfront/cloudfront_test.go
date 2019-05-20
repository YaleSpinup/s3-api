package cloudfront

import (
	"reflect"
	"testing"
	"time"

	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
	"github.com/google/uuid"
)

var testTime = time.Now()

// mockCloudFrontClient is a fake cloudfront client
type mockCloudFrontClient struct {
	cloudfrontiface.CloudFrontAPI
	t   *testing.T
	err error
}

func newmockCloudFrontClient(t *testing.T, err error) cloudfrontiface.CloudFrontAPI {
	return &mockCloudFrontClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	if to := reflect.TypeOf(e).String(); to != "cloudfront.CloudFront" {
		t.Errorf("expected type to be 'cloudfront.CloudFront', got %s", to)
	}
}

func TestWebsiteDomain(t *testing.T) {
	e := NewSession(common.Account{
		Domains: map[string]*common.Domain{
			"hyper.converged": &common.Domain{
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
	})

	if _, err := e.WebsiteDomain(""); err == nil {
		t.Error("expected empty website to result in error, got nil")
	}

	if _, err := e.WebsiteDomain("some.other.domain"); err == nil {
		t.Error("expected empty website to result in error, got nil")
	}

	if _, err := e.WebsiteDomain("someotherdomain"); err == nil {
		t.Error("expected empty website to result in error, got nil")
	}

	domain, err := e.WebsiteDomain("im.hyper.converged")
	if err != nil {
		t.Errorf("expected va;id website to result in nil error, got %s", err)
	}

	if to := reflect.TypeOf(domain).String(); to != "*common.Domain" {
		t.Errorf("expected type common.Domain, got %s", to)
	}
}

func TestDefaultWebsiteDistributionConfig(t *testing.T) {
	callerRef := uuid.New().String()
	expected := &cloudfront.DistributionConfig{
		Aliases: &cloudfront.Aliases{
			Items: []*string{
				aws.String("im.hyper.converged"),
			},
			Quantity: aws.Int64(1),
		},
		DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
			ForwardedValues: &cloudfront.ForwardedValues{
				Cookies: &cloudfront.CookiePreference{
					Forward: aws.String("none"),
				},
				QueryString: aws.Bool(false),
			},
			MinTTL:         aws.Int64(0),
			MaxTTL:         aws.Int64(600),
			TargetOriginId: aws.String("im.hyper.converged"),
			TrustedSigners: &cloudfront.TrustedSigners{
				Enabled:  aws.Bool(false),
				Quantity: aws.Int64(0),
			},
			ViewerProtocolPolicy: aws.String("redirect-to-https"),
		},
		CallerReference:   aws.String(callerRef),
		Comment:           aws.String("im.hyper.converged"),
		DefaultRootObject: aws.String("index.html"),
		Enabled:           aws.Bool(true),
		Origins: &cloudfront.Origins{
			Items: []*cloudfront.Origin{
				&cloudfront.Origin{
					DomainName: aws.String("im.hyper.converged.s3-website-us-east-1.amazonaws.com"),
					Id:         aws.String("im.hyper.converged"),
					CustomOriginConfig: &cloudfront.CustomOriginConfig{
						HTTPPort:             aws.Int64(80),
						HTTPSPort:            aws.Int64(443),
						OriginProtocolPolicy: aws.String("http-only"),
					},
				},
			},
			Quantity: aws.Int64(1),
		},
		PriceClass: aws.String("PriceClass_100"),
		Restrictions: &cloudfront.Restrictions{
			GeoRestriction: &cloudfront.GeoRestriction{
				Items: []*string{
					aws.String("US"),
				},
				Quantity:        aws.Int64(1),
				RestrictionType: aws.String("whitelist"),
			},
		},
		ViewerCertificate: &cloudfront.ViewerCertificate{
			ACMCertificateArn:      aws.String("arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555"),
			MinimumProtocolVersion: aws.String("TLSv1.1_2016"),
			SSLSupportMethod:       aws.String("sni-only"),
		},
	}

	e := NewSession(common.Account{
		Domains: map[string]*common.Domain{
			"hyper.converged": &common.Domain{
				CertArn: "arn:aws:acm::12345678910:certificate/111111111-2222-3333-4444-555555555555",
			},
		},
		Region: "us-east-1",
	})

	if _, err := e.DefaultWebsiteDistributionConfig(""); err == nil {
		t.Error("expected empty website to result in error, got nil")
	}

	if _, err := e.DefaultWebsiteDistributionConfig("some.other.domain"); err == nil {
		t.Error("expected empty website to result in error, got nil")
	}

	if _, err := e.DefaultWebsiteDistributionConfig("someotherdomain"); err == nil {
		t.Error("expected empty website to result in error, got nil")
	}

	config, err := e.DefaultWebsiteDistributionConfig("im.hyper.converged")
	if err != nil {
		t.Errorf("expected success for valid domain, got error: %s", err)
	}
	config.CallerReference = aws.String(callerRef)

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected %+v, got %+v", expected, config)
	}
}
