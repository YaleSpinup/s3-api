package route53

import (
	"context"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

// CreateRecord creates a route53 resource record
func (r *Route53) CreateRecord(ctx context.Context, zoneID string, record *route53.ResourceRecordSet) (*route53.ChangeInfo, error) {
	if record == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	out, err := r.Service.ChangeResourceRecordSetsWithContext(ctx, &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				&route53.Change{
					Action:            aws.String("CREATE"),
					ResourceRecordSet: record,
				},
			},
			Comment: aws.String("Created by s3-api"),
		},
		HostedZoneId: aws.String(zoneID),
	})

	if err != nil {
		return nil, ErrCode("failed to create route53 record", err)
	}

	return out.ChangeInfo, nil
}
