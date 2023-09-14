package s3

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	Lifecycles = SupportedLifecycles{
		Rules: map[string]s3.LifecycleRule{
			"deep-archive": {
				Status: aws.String(s3.IntelligentTieringStatusEnabled),
				Filter: &s3.LifecycleRuleFilter{
					Prefix: aws.String(""),
				},
				ID: aws.String("deep-archive-rule"),
				Transitions: []*s3.Transition{
					{
						Days:         aws.Int64(1),
						StorageClass: aws.String(s3.TransitionStorageClassDeepArchive),
					},
				},
			},
		},
	}
)

type SupportedLifecycles struct {
	Rules map[string]s3.LifecycleRule
}

func (l *SupportedLifecycles) GetLifecycle(lifecycle string) *s3.LifecycleRule {
	var lifecycleRule s3.LifecycleRule

	for k, v := range l.Rules {
		if k == lifecycle {
			lifecycleRule = v
		}
	}

	return &lifecycleRule
}
