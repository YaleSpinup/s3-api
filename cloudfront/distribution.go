package cloudfront

import (
	"context"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	log "github.com/sirupsen/logrus"
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

// ListDistributions lists all cloudfront distributions
func (c *CloudFront) ListDistributions(ctx context.Context) ([]*cloudfront.DistributionSummary, error) {
	distributions := []*cloudfront.DistributionSummary{}

	log.Info("listing cloudfrnot distributions ")

	input := cloudfront.ListDistributionsInput{MaxItems: aws.Int64(100)}
	truncated := true
	for truncated {
		output, err := c.Service.ListDistributionsWithContext(ctx, &input)
		if err != nil {
			return nil, ErrCode("failed to list cloudfront distributions", err)
		}

		truncated = aws.BoolValue(output.DistributionList.IsTruncated)
		distributions = append(distributions, output.DistributionList.Items...)
		input.Marker = output.DistributionList.Marker
	}

	return distributions, nil
}
