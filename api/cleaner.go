package api

import (
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	log "github.com/sirupsen/logrus"
)

// cleanerInterval generates the cleaner interval from the baseInterval and a max splay
func cleanerInterval(baseInterval, maxSplay string) (*time.Duration, error) {
	base, err := time.ParseDuration(baseInterval)
	if err != nil {
		return nil, err
	}
	log.Debugf("parsed base interval (%s) of %f seconds", baseInterval, base.Seconds())

	maxs, err := time.ParseDuration(maxSplay)
	if err != nil {
		return nil, err
	}
	log.Debugf("parsed max splay interval (%s) of %f seconds", maxSplay, maxs.Seconds())

	r := rand.Int63n(int64(maxs))
	interval := base + time.Duration(r)
	log.Infof("starting cleaner with interval of %fs", interval.Seconds())

	return &interval, nil
}

// run starts the cleaner and listens for a shutdown call.
func (c *cleaner) run() {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		// Loop that runs forever
		for {
			select {
			case <-ticker.C:
				err := c.action()
				if err != nil {
					log.Errorf("error executing cleaner: %s", err)
				}
			case <-c.context.Done():
				log.Debug("shutting down cleaner timer")
				ticker.Stop()
				return
			}
			log.Debug("starting cleaner loop")
		}
	}()

	log.Println("Cleaner Started")
}

// Action defines what the cleaner does...
// 1. get a list of cloudfront distributions, that are disabled but deployed
// 2. for any found distributions, check if the route53 resource record exists, bucket exists?
// 3. delete if orphaned
func (c *cleaner) action() error {
	log.Debugf("starting cleanup action for account %s", c.account)
	distributions, err := c.cloudFrontService.ListDistributionsWithFilter(c.context, func(dist *cloudfront.DistributionSummary) bool {
		if aws.StringValue(dist.Status) == "Deployed" && !aws.BoolValue(dist.Enabled) {
			log.Debugf("distribution %s (%s) is deployed but disabled", aws.StringValue(dist.DomainName), aws.StringValue(dist.Comment))
			return true
		}

		return false
	})
	if err != nil {
		return err
	}

	for _, dist := range distributions {
		exists, err := c.s3Service.BucketExists(c.context, aws.StringValue(dist.DefaultCacheBehavior.TargetOriginId))
		if err != nil {
			return err
		}

		if !exists {
			id := aws.StringValue(dist.Id)
			origin := aws.StringValue(dist.DefaultCacheBehavior.TargetOriginId)
			log.Infof("cloudfront distribution (%s) is deployed, disabled. bucket %s doesn't exist. deleting.", id, origin)
			if err := c.cloudFrontService.DeleteDistribution(c.context, id); err != nil {
				return err
			}
		}

	}

	return nil
}
