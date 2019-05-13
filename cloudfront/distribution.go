package cloudfront

import (
	"context"
	"fmt"

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

// GetDistributionByName gets a cloudfront distribution by the name (by searching until it finds the matching alias)
func (c *CloudFront) GetDistributionByName(ctx context.Context, name string) (*cloudfront.DistributionSummary, error) {
	log.Infof("searching for cloudfront distribution %s", name)

	input := &cloudfront.ListDistributionsInput{MaxItems: aws.Int64(100)}

	var distribution *cloudfront.DistributionSummary
	err := c.Service.ListDistributionsPagesWithContext(ctx, input,
		func(out *cloudfront.ListDistributionsOutput, lastPage bool) bool {
			for _, dist := range out.DistributionList.Items {
				log.Debugf("checking %+v aliases against name %s", dist, name)
				for _, alias := range dist.Aliases.Items {
					log.Debugf("checking alias %s against name %s", aws.StringValue(alias), name)
					if aws.StringValue(alias) == name {
						distribution = dist
						return false
					}
				}
			}
			return true
		})
	if err != nil {
		return nil, ErrCode("failed to list cloudfront distributions", err)
	}

	if distribution == nil {
		msg := fmt.Sprintf("cloudfront distribution not found with name %s", name)
		err = apierror.New(apierror.ErrNotFound, msg, nil)
	}

	return distribution, err
}
