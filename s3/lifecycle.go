package s3

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"time"
)

var (
	Lifecycles = SupportedLifecycles{
		Rules: map[string]s3.LifecycleRule{
			"deep-archive": {
				Status: aws.String(s3.IntelligentTieringStatusEnabled),
				Transitions: []*s3.Transition{
					{
						Date:         aws.Time(getMidnightTomorrow()),
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

// getMidnightTomorrow gets the current time, adds a day, then sets it to midnight
func getMidnightTomorrow() time.Time {
	sometimeTomorrow := time.Now().Add(time.Hour * 24)
	timeStr := fmt.Sprintf("%d-%d-%dT00:00:00Z",
		sometimeTomorrow.Year(),
		sometimeTomorrow.Month(),
		sometimeTomorrow.Day(),
	)

	out, _ := time.Parse(time.RFC3339, timeStr)

	return out
}
