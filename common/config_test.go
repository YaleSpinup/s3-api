package common

import (
	"bytes"
	"reflect"
	"testing"
)

var testConfig = []byte(
	`{
		"listenAddress": ":8000",
		"accounts": {
		  "provider1": {
			"region": "us-east-1",
			"akid": "key1",
			"secret": "secret1",
			"defaultS3BucketActions": [
				"s3:abc123",
				"s3:xyz456"
			],
			"defaultS3ObjectActions": [
				"s3:*"
			],
			"accessLog": {
				"bucket": "foobucket",
				"prefix": "spinup"
			},
			"domains": {
				"example.com": {
					"certArn": "arn:123456789:thingy"
				}
			}
		  },
		  "provider2": {
			"region": "us-west-1",
			"akid": "key2",
			"secret": "secret2"
		  }
		},
		"token": "SEKRET",
		"logLevel": "info"
	}`)

var brokenConfig = []byte(`{ "foobar": { "baz": "biz" }`)

func TestReadConfig(t *testing.T) {
	expectedConfig := Config{
		ListenAddress: ":8000",
		Accounts: map[string]Account{
			"provider1": Account{
				Region: "us-east-1",
				Akid:   "key1",
				Secret: "secret1",
				DefaultS3BucketActions: []string{
					"s3:abc123",
					"s3:xyz456",
				},
				DefaultS3ObjectActions: []string{
					"s3:*",
				},
				AccessLog: AccessLog{
					Bucket: "foobucket",
					Prefix: "spinup",
				},
				Domains: map[string]Domain{
					"example.com": Domain{
						CertArn: "arn:123456789:thingy",
					},
				},
			},
			"provider2": Account{
				Region: "us-west-1",
				Akid:   "key2",
				Secret: "secret2",
			},
		},
		Token:    "SEKRET",
		LogLevel: "info",
	}

	actualConfig, err := ReadConfig(bytes.NewReader(testConfig))
	if err != nil {
		t.Error("Failed to read config", err)
	}

	if !reflect.DeepEqual(actualConfig, expectedConfig) {
		t.Errorf("Expected config to be %+v\n got %+v", expectedConfig, actualConfig)
	}

	_, err = ReadConfig(bytes.NewReader(brokenConfig))
	if err == nil {
		t.Error("expected error reading config, got nil")
	}
}
