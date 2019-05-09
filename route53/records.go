package route53

import (
	"context"
	"fmt"
	"strings"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"

	log "github.com/sirupsen/logrus"
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

// GetRecord gets a route53 resource record
func (r *Route53) GetRecord(ctx context.Context, zoneID, host, recordType string) (*route53.ResourceRecordSet, error) {
	log.Infof("getting route53 record for zone ID %s, host %s", zoneID, host)

	records, err := r.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	for _, r := range records {
		if !strings.HasSuffix(host, ".") {
			host = host + "."
		}

		log.Debugf("checking %+v against host %s and type %s", r, host, recordType)

		if aws.StringValue(r.Name) == host && aws.StringValue(r.Type) == recordType {
			return r, nil
		}
	}

	msg := fmt.Sprintf("route53 record not found in zone %s with name %s and type %s", zoneID, host, recordType)
	return nil, apierror.New(apierror.ErrNotFound, msg, nil)
}

// ListRecords lists the route53 resource records for a zone
func (r *Route53) ListRecords(ctx context.Context, zoneID string) ([]*route53.ResourceRecordSet, error) {
	recordSets := []*route53.ResourceRecordSet{}

	log.Infof("listing route53 records for zone ID %s", zoneID)

	input := route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		MaxItems:     aws.String("100"),
	}
	truncated := true
	for truncated {
		output, err := r.Service.ListResourceRecordSetsWithContext(ctx, &input)
		if err != nil {
			return nil, ErrCode("failed to list route53 resource record sets", err)
		}

		truncated = aws.BoolValue(output.IsTruncated)
		recordSets = append(recordSets, output.ResourceRecordSets...)
		input.StartRecordName = output.NextRecordName
		input.StartRecordType = output.NextRecordType

		log.Debugf("adding %+v to record sets", output.ResourceRecordSets)
	}

	return recordSets, nil
}
