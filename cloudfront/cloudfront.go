package cloudfront

import (
	"errors"
	"strings"

	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// CloudFront is a wrapper around the aws cloudfront service with some default config info
type CloudFront struct {
	Service         cloudfrontiface.CloudFrontAPI
	Domains         map[string]*common.Domain
	WebsiteEndpoint string
}

// NewSession creates a new cloudfront session
func NewSession(account common.Account) CloudFront {
	c := CloudFront{}
	log.Infof("creating new aws session for cloudfront with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	c.Service = cloudfront.New(sess)
	c.Domains = account.Domains
	c.WebsiteEndpoint = "s3-website-" + account.Region + ".amazonaws.com"
	return c
}

// WebsiteDomain validates the name of the website, ensuring we have a cert for the domain and returning the domain.  It
// splits the website name in 2 pieces since we are using a wildcard.  This would need to change if we supported
// certificates per website.
func (c *CloudFront) WebsiteDomain(name string) (*common.Domain, error) {
	log.Infof("validating website name %s", name)

	if name == "" {
		return nil, errors.New("website cannot be empty")
	}

	nameParts := strings.SplitN(name, ".", 2)
	if nameParts == nil || len(nameParts) < 2 {
		return nil, errors.New("invalid website length, not enough parts")
	}

	log.Infof("split website name into parts: %v", nameParts)

	domain, ok := c.Domains[nameParts[1]]
	if !ok {
		return nil, errors.New("domain not found for website")
	}

	return domain, nil
}

// DefaultWebsiteDistributionConfig generates the cloudfront distribution configuration for an s3 website
// https://docs.aws.amazon.com/sdk-for-go/api/service/cloudfront/#DistributionConfig
func (c *CloudFront) DefaultWebsiteDistributionConfig(name string) (*cloudfront.DistributionConfig, error) {
	domain, err := c.WebsiteDomain(name)
	if err != nil {
		return nil, err
	}

	config := cloudfront.DistributionConfig{
		Aliases: &cloudfront.Aliases{
			Items: []*string{
				aws.String(name),
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
			DefaultTTL:     aws.Int64(3600),
			TargetOriginId: aws.String(name),
			TrustedSigners: &cloudfront.TrustedSigners{
				Enabled:  aws.Bool(false),
				Quantity: aws.Int64(0),
			},
			ViewerProtocolPolicy: aws.String("redirect-to-https"),
		},
		CallerReference:   aws.String(uuid.New().String()),
		Comment:           aws.String(name),
		DefaultRootObject: aws.String("index.html"),
		Enabled:           aws.Bool(true),
		Origins: &cloudfront.Origins{
			Items: []*cloudfront.Origin{
				{
					DomainName: aws.String(name + "." + c.WebsiteEndpoint),
					Id:         aws.String(name),
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
		ViewerCertificate: &cloudfront.ViewerCertificate{
			ACMCertificateArn:      aws.String(domain.CertArn),
			MinimumProtocolVersion: aws.String("TLSv1.1_2016"),
			SSLSupportMethod:       aws.String("sni-only"),
		},
	}

	log.Debugf("Generated Distribution Config: %+v", config)

	return &config, nil
}
