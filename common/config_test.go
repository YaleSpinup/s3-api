package common

import (
	"bytes"
	"reflect"
	"testing"
)

var testConfig = []byte(
	`{
		"listenAddress": ":8000",
		"account": {
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
			"defaultCloudfrontDistributionActions": [
				"cloudfront:ListInvalidations",
				"cloudfront:CreateInvalidation"
			],
			"accessLog": {
				"bucket": "foobucket",
				"prefix": "spinup"
			},
			"domains": {
				"example.com": {
					"certArn": "arn:123456789:thingy",
					"hostedZoneId": "AABBCCDDEEFF"
				}
			},
			"cleaner": {
				"interval": "300s",
				"maxSplay": "60s"
			}
		},
		"token": "SEKRET",
		"logLevel": "info",
		"org": "test"
	}`)

var testConfig2 = []byte(
	`{
		"listenAddress": ":8000",
		"account": {
		  	"region": "us-west-1",
			"akid": "key2",
			"secret": "secret2"
		},
		"token": "SEKRET",
		"logLevel": "info",
		"org": "test"
	}`)

var brokenConfig = []byte(`{ "foobar": { "baz": "biz" }`)

var workingConfigs = [][]byte{
	testConfig,
	testConfig2,
}

var testAccessLogsInput = []AccessLog{
	{
		Bucket: "foo-{account_id}-logs",
		Prefix: "foo",
	},
	{
		Bucket: "bar-[what_id]-logs",
		Prefix: "foo",
	},
}

var testAccessLogsIdsInput = []string{
	"123456789",
	"632623623",
}

var testAccessLogsExpected = []string{
	"foo-123456789-logs",
	"bar-[what_id]-logs",
}

func TestReadConfig(t *testing.T) {
	expectedConfigs := []Config{
		{
			ListenAddress: ":8000",
			Account: Account{
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
				DefaultCloudfrontDistributionActions: []string{
					"cloudfront:ListInvalidations",
					"cloudfront:CreateInvalidation",
				},
				AccessLog: AccessLog{
					Bucket: "foobucket",
					Prefix: "spinup",
				},
				Domains: map[string]*Domain{
					"example.com": {
						CertArn:      "arn:123456789:thingy",
						HostedZoneID: "AABBCCDDEEFF",
					},
				},
				Cleaner: &Cleaner{
					Interval: "300s",
					MaxSplay: "60s",
				},
			},
			Token:    "SEKRET",
			LogLevel: "info",
			Org:      "test",
		},
		{
			ListenAddress: ":8000",
			Account: Account{
				Region: "us-west-1",
				Akid:   "key2",
				Secret: "secret2",
			},
			Token:    "SEKRET",
			LogLevel: "info",
			Org:      "test",
		},
	}

	for i, config := range workingConfigs {
		actual, err := ReadConfig(bytes.NewReader(config))
		if err != nil {
			t.Error("Failed to read config", err)
		}

		expected := expectedConfigs[i]
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Expected config to be %+v\n got %+v", expected, actual)
		}
	}

	_, err := ReadConfig(bytes.NewReader(brokenConfig))
	if err == nil {
		t.Error("expected error reading config, got nil")
	}
}

func TestAccessLog_GetBucket(t *testing.T) {
	for i, input := range testAccessLogsInput {
		id := testAccessLogsIdsInput[i]
		expected := testAccessLogsExpected[i]
		actual := input.GetBucket(id)

		if actual != expected {
			t.Errorf("unexpected result from GetBucket. wanted %s got %s. input bucket: %s, input account id: %s", expected, actual, input.Bucket, id)
		}
	}
}
