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

// CreateRecord creates a route53 resource record.  This will fail if the record already exists.
func (r *Route53) CreateRecord(ctx context.Context, zoneID string, record *route53.ResourceRecordSet) (*route53.ChangeInfo, error) {
	if record == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	out, err := r.Service.ChangeResourceRecordSetsWithContext(ctx, &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
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

// DeleteRecord deletes a route53 resource record.
func (r *Route53) DeleteRecord(ctx context.Context, zoneID string, record *route53.ResourceRecordSet) (*route53.ChangeInfo, error) {
	if record == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	out, err := r.Service.ChangeResourceRecordSetsWithContext(ctx, &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action:            aws.String("DELETE"),
					ResourceRecordSet: record,
				},
			},
			Comment: aws.String("Deleted by s3-api"),
		},
		HostedZoneId: aws.String(zoneID),
	})

	if err != nil {
		return nil, ErrCode("failed to delete route53 record", err)
	}

	return out.ChangeInfo, nil
}

// GetRecordByName gets a route53 resource record by name and by type if one is specified.
func (r *Route53) GetRecordByName(ctx context.Context, zoneID, name, recordType string) (*route53.ResourceRecordSet, error) {
	log.Infof("getting route53 record for zone ID %s, name %s, type '%s'", zoneID, name, recordType)

	if !strings.HasSuffix(name, ".") {
		name = name + "."
	}

	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		MaxItems:     aws.String("100"),
	}

	var recordSet *route53.ResourceRecordSet
	err := r.Service.ListResourceRecordSetsPagesWithContext(ctx, input,
		func(out *route53.ListResourceRecordSetsOutput, lastPage bool) bool {
			for _, rs := range out.ResourceRecordSets {
				log.Debugf("checking %+v against name %s and type %s", rs, name, recordType)
				if aws.StringValue(rs.Name) == name && aws.StringValue(rs.Type) == recordType {
					recordSet = rs
					return false
				}
			}
			return true
		})
	if err != nil {
		return nil, ErrCode("failed to list route53 resource record sets", err)
	}

	if recordSet == nil {
		msg := fmt.Sprintf("route53 record not found in zone %s with name %s and type '%s'", zoneID, name, recordType)
		err = apierror.New(apierror.ErrNotFound, msg, nil)
	}

	return recordSet, err
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
