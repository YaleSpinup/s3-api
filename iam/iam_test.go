package iam

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/s3-api/common"
)

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "iam.IAM" {
		t.Errorf("expected type to be 'iam.IAM', got %s", to)
	}
}
