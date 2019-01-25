package s3

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
)

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "ecs.ECS" {
		t.Errorf("expected type to be 'ecs.ECS', got %s", to)
	}
}
