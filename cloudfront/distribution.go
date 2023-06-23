package cloudfront

import (
	"context"
	"fmt"
	"strings"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/google/uuid"
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

// DisableDistribution disables a cloudfront distribution
func (c *CloudFront) DisableDistribution(ctx context.Context, id string) (*cloudfront.Distribution, error) {
	if id == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("disabling cloudfront distributions Id: %s", id)

	// Get the distribution config from the passed distribution id.  This is required to get the most recent ETag for the distribution.
	config, err := c.Service.GetDistributionConfigWithContext(ctx, &cloudfront.GetDistributionConfigInput{Id: aws.String(id)})
	if err != nil {
		return nil, ErrCode("failed to get details about cloudfront distribution Id: "+id, err)
	}

	config.DistributionConfig.Enabled = aws.Bool(false)
	out, err := c.Service.UpdateDistributionWithContext(ctx, &cloudfront.UpdateDistributionInput{
		DistributionConfig: config.DistributionConfig,
		IfMatch:            config.ETag,
		Id:                 aws.String(id),
	})
	if err != nil {
		return nil, ErrCode("failed to disable cloudfront distribution Id:"+id, err)
	}

	return out.Distribution, nil
}

// DeleteDistribution deletes a cloudfront distribution
func (c *CloudFront) DeleteDistribution(ctx context.Context, id string) error {
	if id == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting cloudfront distributions Id: %s", id)

	// Get the distribution config from the passed distribution id.  This is required to get the most recent ETag for the distribution.
	config, err := c.Service.GetDistributionConfigWithContext(ctx, &cloudfront.GetDistributionConfigInput{Id: aws.String(id)})
	if err != nil {
		return ErrCode("failed to get details about cloudfront distribution Id: "+id, err)
	}

	_, err = c.Service.DeleteDistributionWithContext(ctx, &cloudfront.DeleteDistributionInput{
		IfMatch: config.ETag,
		Id:      aws.String(id),
	})
	if err != nil {
		return ErrCode("failed to delete cloudfront distribution Id:"+id, err)
	}

	return nil
}

// TagDistribution updates the tags for a cloudfront distribution
func (c *CloudFront) TagDistribution(ctx context.Context, arn string, tags *cloudfront.Tags) error {
	if arn == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("tagging cloudfront distributions ARN: %s", arn)

	_, err := c.Service.TagResourceWithContext(ctx, &cloudfront.TagResourceInput{
		Resource: aws.String(arn),
		Tags:     tags,
	})
	if err != nil {
		return ErrCode("failed to tag cloudfront distribution ARN:"+arn, err)
	}

	return nil
}

// ListDistributions lists all cloudfront distributions.
func (c *CloudFront) ListDistributions(ctx context.Context) ([]*cloudfront.DistributionSummary, error) {
	distributions := []*cloudfront.DistributionSummary{}

	log.Info("listing cloudfront distributions")

	input := cloudfront.ListDistributionsInput{MaxItems: aws.Int64(100)}
	truncated := true
	for truncated {
		output, err := c.Service.ListDistributionsWithContext(ctx, &input)
		if err != nil {
			return nil, ErrCode("failed to list cloudfront distributions", err)
		}

		distributions = append(distributions, output.DistributionList.Items...)
		truncated = aws.BoolValue(output.DistributionList.IsTruncated)
		input.Marker = output.DistributionList.Marker
	}

	return distributions, nil
}

// ListDistributionsWithFilter lists all cloudfront distributions and passes each DistributionSummary into the filter func to decide if it should be added or discarded
func (c *CloudFront) ListDistributionsWithFilter(ctx context.Context, filter func(*cloudfront.DistributionSummary) bool) ([]*cloudfront.DistributionSummary, error) {
	var distributions []*cloudfront.DistributionSummary

	log.Info("listing cloudfront distributions")

	input := cloudfront.ListDistributionsInput{MaxItems: aws.Int64(100)}
	truncated := true
	for truncated {
		output, err := c.Service.ListDistributionsWithContext(ctx, &input)
		if err != nil {
			return nil, ErrCode("failed to list cloudfront distributions", err)
		}

		for _, item := range output.DistributionList.Items {
			if filter(item) {
				distributions = append(distributions, item)
			}
		}

		truncated = aws.BoolValue(output.DistributionList.IsTruncated)
		input.Marker = output.DistributionList.Marker
	}

	log.Debugf("returing cloudfront distributions list: %+v", distributions)

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

// InvalidateCache submits a cache invalidation request to cloudfront
func (c *CloudFront) InvalidateCache(ctx context.Context, id string, paths []string) (*cloudfront.CreateInvalidationOutput, error) {
	if id == "" || len(paths) == 0 {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("invalidating paths %+v for cloudfront distribution Id: %s", strings.Join(paths, ","), id)

	out, err := c.Service.CreateInvalidationWithContext(ctx, &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(id),
		InvalidationBatch: &cloudfront.InvalidationBatch{
			CallerReference: aws.String(uuid.New().String()),
			Paths: &cloudfront.Paths{
				Items:    aws.StringSlice(paths),
				Quantity: aws.Int64(int64(len(paths))),
			},
		},
	})
	if err != nil {
		return nil, ErrCode("failed to invalidate cloudfront distributions", err)
	}

	log.Debugf("got cache invalidation output %+v", out)

	return out, nil
}

// ListTags lists the tags for an ARN
func (c *CloudFront) ListTags(ctx context.Context, arn string) ([]*cloudfront.Tag, error) {
	if arn == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing tags for cloudfront resource %s", arn)

	out, err := c.Service.ListTagsForResourceWithContext(ctx, &cloudfront.ListTagsForResourceInput{Resource: aws.String(arn)})
	if err != nil {
		return nil, ErrCode("failed to list tags cloudfront distributions", err)
	}

	log.Debugf("got tags list output for resource %s: %+v", arn, out)

	tags := []*cloudfront.Tag{}
	if out.Tags != nil && len(out.Tags.Items) > 0 {
		tags = out.Tags.Items
	}

	return tags, nil
}
