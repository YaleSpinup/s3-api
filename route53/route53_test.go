package route53

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
)

// mockRoute53Client is a fake S3 client
type mockRoute53Client struct {
	route53iface.Route53API
	t   *testing.T
	err error
}

func newmockRoute53Client(t *testing.T, err error) route53iface.Route53API {
	return &mockRoute53Client{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(nil, common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "route53.Route53" {
		t.Errorf("expected type to be 'route53.Route53', got %s", to)
	}
}
