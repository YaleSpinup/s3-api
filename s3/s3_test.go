package s3

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/s3-api/common"
)

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "s3.S3" {
		t.Errorf("expected type to be 's3.S3', got %s", to)
	}
}
