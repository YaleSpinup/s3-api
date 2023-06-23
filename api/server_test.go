package api

import (
	"testing"
)

func TestMapToAccountName(t *testing.T) {
	s := server{
		accountsMap: map[string]string{
			"abc": "123456789061",
			"cde": "754378345844",
		},
	}

	tests := map[string]string{
		"123456789061": "abc",
		"754378345844": "cde",
		"251263623341": "251263623341",
		"":             "",
	}

	for input, expected := range tests {
		actual := s.mapToAccountName(input)

		if actual != expected {
			t.Errorf("unexpected result from input: %s. wanted %s received %s", input, expected, actual)
		}
	}
}
