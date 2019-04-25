package cloudfront

import (
	"context"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/service/cloudfront"
)

// CreateDistribution creates a cloudfront distribution with tags
func (c *CloudFront) CreateDistribution(ctx context.Context, distribution *cloudfront.DistributionConfig, tags *cloudfront.Tags) (*cloudfront.Distribution, error) {
	if distribution == nil || tags == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	out, err := c.Service.CreateDistributionWithTagsWithContext(ctx, &cloudfront.CreateDistributionWithTagsInput{
		DistributionConfigWithTags: &cloudfront.DistributionConfigWithTags{
			DistributionConfig: distribution,
			Tags:               tags,
		},
	})
	if err != nil {
		return nil, ErrCode("failed to create cloudfront distribution", err)
	}
	return out.Distribution, nil
}
