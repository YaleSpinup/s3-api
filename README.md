# s3-api

This API provides simple restful API access to Amazon's S3 service.

## Endpoints

```
GET /v1/s3/ping
GET /v1/s3/version
GET /v1/s3/metrics

# Managing buckets
POST /v1/s3/{account}/buckets
GET /v1/s3/{account}/buckets
HEAD /v1/s3/{account}/buckets/{bucket}
GET /v1/s3/{account}/buckets/{bucket}
PUT /v1/s3/{account}/buckets/{bucket}
DELETE /v1/s3/{account}/buckets/{bucket}

# Managing bucket users
POST /v1/s3/{account}/buckets/{bucket}/users
GET /v1/s3/{account}/buckets/{bucket}/users
GET /v1/s3/{account}/buckets/{bucket}/users/{user}
PUT /v1/s3/{account}/buckets/{bucket}/users/{user}
DELETE /v1/s3/{account}/buckets/{bucket}/users/{user}

# Managing websites
POST /v1/s3/{account}/websites
HEAD /v1/s3/{account}/websites/{website}
GET /v1/s3/{account}/websites/{website}
PUT /v1/s3/{account}/websites/{website}
PATCH /v1/s3/{account}/websites/{website}
DELETE /v1/s3/{account}/websites/{website}

# Managing website users
POST /v1/s3/{account}/websites/{website}/users
GET /v1/s3/{account}/websites/{website}/users
GET /v1/s3/{account}/websites/{website}/users/{user}
PUT /v1/s3/{account}/websites/{website}/users/{user}
DELETE /v1/s3/{account}/websites/{website}/users/{user}
```

## Authentication

Authentication is accomplished via a pre-shared key.  This is done via the `X-Auth-Token` header.

## Access to buckets

When creating a bucket, by default, an IAM policy (of the same name) is created with full access to that
bucket...

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:AbortMultipartUpload",
                "s3:DeleteBucketWebsite",
                "s3:DeleteObject",
                "s3:DeleteObjectVersion",
                "s3:GetAccelerateConfiguration",
                "s3:GetBucketAcl",
                "s3:GetBucketCORS",
                "s3:GetBucketLocation",
                "s3:GetBucketLogging",
                "s3:GetBucketNotification",
                "s3:GetBucketPolicy",
                "s3:GetBucketTagging",
                "s3:GetBucketVersioning",
                "s3:GetBucketWebsite",
                "s3:GetLifecycleConfiguration",
                "s3:GetObject",
                "s3:GetObjectAcl",
                "s3:GetObjectVersion",
                "s3:GetObjectVersionAcl",
                "s3:GetReplicationConfiguration",
                "s3:ListAllMyBuckets",
                "s3:ListBucket",
                "s3:ListBucketMultipartUploads",
                "s3:ListBucketVersions",
                "s3:ListMultipartUploadParts",
                "s3:PutAccelerateConfiguration",
                "s3:PutBucketAcl",
                "s3:PutBucketCORS",
                "s3:PutBucketLogging",
                "s3:PutBucketNotification",
                "s3:PutBucketPolicy",
                "s3:PutBucketRequestPayment",
                "s3:PutBucketTagging",
                "s3:PutBucketVersioning",
                "s3:PutBucketWebsite",
                "s3:PutLifecycleConfiguration",
                "s3:PutReplicationConfiguration",
                "s3:PutObject",
                "s3:PutObjectAcl",
                "s3:PutObjectVersionAcl",
                "s3:ReplicateDelete",
                "s3:ReplicateObject",
                "s3:RestoreObject"
            ],
            "Resource": [
                "arn:aws:s3:::my-awesome-bucket"
            ]
        },
        {
            "Effect": "Allow",
            "Action": "s3:*",
            "Resource": [
                "arn:aws:s3:::my-awesome-bucket/*"
            ]
        }
    ]
}
```

and a group is created with that policy attached.  To allow access to a bucket, create a bucket user
by POSTing to the `/v1/s3/{account}/buckets/{bucket}/users` endpoint.

## Examples

### Get a list of buckets

GET `/v1/s3/{account}/buckets`

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **200 OK**                    | return the list of buckets      |  
| **400 Bad Request**           | badly formed request            |  
| **404 Not Found**             | account not found               |  
| **500 Internal Server Error** | a server error occurred         |

### Create a bucket

POST `/v1/s3/{account}/buckets

#### Request

```json
{
    "Tags": [
        { "Key": "Application", "Value": "HowToGet" },
        { "Key": "COA", "Value": "Take.My.Money.$$$$" },
        { "Key": "CreatedBy", "Value": "Big Bird" }
    ],
    "BucketInput": {
        "Bucket": "foobarbucketname"
    }
}
```

#### Response

```json
{
    "Bucket": "/foobarbucketname",
    "Policy": {
        "Arn": "arn:aws:iam::12345678910:policy/foobarbucketname-BktAdmPlc",
        "AttachmentCount": 0,
        "CreateDate": "2019-03-01T15:33:52Z",
        "DefaultVersionId": "v1",
        "Description": null,
        "IsAttachable": true,
        "Path": "/",
        "PermissionsBoundaryUsageCount": 0,
        "PolicyId": "ABCDEFGHI12345678",
        "PolicyName": "foobarbucketname-BktAdmPlc",
        "UpdateDate": "2019-03-01T15:33:52Z"
    },
    "Group": {
        "Arn": "arn:aws:iam::12345678910:group/foobarbucketname-BktAdmGrp",
        "CreateDate": "2019-03-01T15:33:52Z",
        "GroupId": "GROUPID123",
        "GroupName": "foobarbucketname-BktAdmGrp",
        "Path": "/"
    }
}
```

| Response Code                 | Definition                           |  
| ----------------------------- | -------------------------------------|  
| **202 Accepted**              | creation request accepted            |  
| **400 Bad Request**           | badly formed request                 |  
| **403 Forbidden**             | you don't have access to bucket      |  
| **404 Not Found**             | account not found                    |  
| **409 Conflict**              | bucket or iam policy  already exists |
| **429 Too Many Requests**     | service or rate limit exceeded       |
| **500 Internal Server Error** | a server error occurred              |
| **503 Service Unavailable**   | an AWS service is unavailable        |

### Update a bucket

Updating a bucket currently only supports updating the bucket's tags

PUT `/v1/s3/{account}/buckets/foobarbucketname`

#### Request

```json
{
    "Tags": [
        { "Key": "Application", "Value": "HowToGet" },
        { "Key": "COA", "Value": "Take.My.Money.$$$$" },
        { "Key": "CreatedBy", "Value": "Big Bird" }
    ]
}
```

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **200 OK**                    | deleted bucket                  |  
| **400 Bad Request**           | badly formed request            |  
| **500 Internal Server Error** | a server error occurred         |

### Check if a bucket exists

HEAD `/v1/s3/{account}/buckets/foobarbucketname`

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **200 OK**                    | bucket exists                   |  
| **403 Forbidden**             | you don't have access to bucket |  
| **404 Not Found**             | account or bucket not found     |  
| **500 Internal Server Error** | a server error occurred         |

### Get information for a bucket

GET `/v1/s3/{account}/buckets/foobarbucketname`

#### Response

```json
{
    "Tags": [
        { "Key": "Application", "Value": "HowToGet" },
        { "Key": "COA", "Value": "Take.My.Money.$$$$" },
        { "Key": "CreatedBy", "Value": "Big Bird" }
    ],
    "Logging": {
        "TargetBucket": "foobar-buckets-access-logs",
        "TargetGrants": null,
        "TargetPrefix": "s3/foobarbucketname/"
    },
    "Empty": true
}
```

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **200 OK**                    | okay                            |  
| **404 Not Found**             | bucket was not found            |  
| **400 Bad Request**           | badly formed request            |  
| **500 Internal Server Error** | a server error occurred         |

### Delete a bucket

DELETE `/v1/s3/{account}/buckets/{bucket}

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **200 OK**                    | deleted bucket                  |  
| **400 Bad Request**           | badly formed request            |  
| **403 Forbidden**             | you don't have access to bucket |  
| **404 Not Found**             | account or bucket not found     |  
| **409 Conflict**              | bucket is not empty             |
| **500 Internal Server Error** | a server error occurred         |

### Create a bucket user

POST `/v1/s3/{account}/buckets/{bucket}/users

#### Request

```json
{
    "UserName": "somebucketuser"
}
```

#### Response

```json
{
    "User": {
        "Arn": "arn:aws:iam::12345678910:user/somebucketuser",
        "CreateDate": "2019-03-01T16:11:00Z",
        "PasswordLastUsed": null,
        "Path": "/",
        "PermissionsBoundary": null,
        "Tags": null,
        "UserId": "AIDAJJSBBEAVOQLFAAUCG",
        "UserName": "somebucketuser"
    },
    "AccessKey": {
        "AccessKeyId": "ABCDEFGHIJ12345678",
        "CreateDate": "2019-03-01T16:11:00Z",
        "SecretAccessKey": "sssshimsupersekretdonttellanyoneyousawme",
        "Status": "Active",
        "UserName": "somebucketuser"
    }
}
```

| Response Code                 | Definition                                  |  
| ----------------------------- | --------------------------------------------|  
| **200 OK**                    | user created                                |  
| **400 Bad Request**           | badly formed request                        |  
| **403 Forbidden**             | you don't have access to bucket             |  
| **404 Not Found**             | account or user not found when creating key |  
| **409 Conflict**              | user already exists                         |  
| **429 Too Many Requests**     | service or rate limit exceeded              |  
| **500 Internal Server Error** | a server error occurred                     |

### Get a bucket user's details

GET `/v1/s3/{account}/bucket/users/{user}`

#### Response

```json
{
    "User": {
        "Arn": "arn:aws:iam::12345678910:user/somebucketuser",
        "CreateDate": "2019-03-19T18:31:14Z",
        "PasswordLastUsed": null,
        "Path": "/",
        "PermissionsBoundary": null,
        "Tags": null,
        "UserId": "AIDAJJSBBEAVOQLFAAUCG",
        "UserName": "somebucketuser"
    },
    "AccessKeys": [
        {
            "AccessKeyId": "AKIAJTGA5ITTTJ7WOR7A",
            "CreateDate": "2019-03-19T18:31:14Z",
            "Status": "Active",
            "UserName": "somebucketuser"
        }
    ],
    "Groups": [
        {
            "Arn": "arn:aws:iam::12345678910:group/somebucketuser",
            "CreateDate": "2019-03-19T14:20:01Z",
            "GroupId": "AGPAJ6SYNPMFP6O5KXQJW",
            "GroupName": "somebucketuser-BktAdmGrp",
            "Path": "/"
        }
    ],
    "Policies": [
        {
            "Arn": "arn:aws:iam::12345678910:policy/somebucketuser-BktAdmPlc",
            "PolicyName": "somebucketuser-BktAdmPlc"
        }
    ]
}
```


### Reset access keys for a bucket user

PUT `/v1/s3/{account}/buckets/{bucket}/users/{user}`

#### Response

```json
{
    "DeletedKeyIds": [
        "ABCDEFGHIJK123456789"
    ],
    "AccessKey": {
        "AccessKeyId": "LMNOPQRSTUVW123456789",
        "CreateDate": "2019-03-01T16:14:07Z",
        "SecretAccessKey": "sssshimsupersekretdonttellanyoneyousawme",
        "Status": "Active",
        "UserName": "someuser-admin1"
    }
}
```

| Response Code                 | Definition                               |  
| ----------------------------- | -----------------------------------------|  
| **200 OK**                    | keys reset successfully                  |  
| **400 Bad Request**           | badly formed request                     |  
| **403 Forbidden**             | you don't have access to delete the user |  
| **404 Not Found**             | account or user not found                |  
| **429 Too Many Requests**     | service or rate limit exceeded           |  
| **500 Internal Server Error** | a server error occurred                  |

### List users for a bucket

GET `/v1/s3/{account}/buckets/{bucket}/users/{user}

#### Response

```json
[
    {
        "Arn": "arn:aws:iam::12345678910:user/someuser-admin1",
        "CreateDate": "2019-03-01T16:11:00Z",
        "PasswordLastUsed": null,
        "Path": "/",
        "PermissionsBoundary": null,
        "Tags": null,
        "UserId": "ABCDEFGHI12345678",
        "UserName": "someuser-admin1"
    },
        {
        "Arn": "arn:aws:iam::12345678910:user/someuser-admin2",
        "CreateDate": "2019-03-01T16:11:00Z",
        "PasswordLastUsed": null,
        "Path": "/",
        "PermissionsBoundary": null,
        "Tags": null,
        "UserId": "ZYXWUTS87654321",
        "UserName": "someuser-admin2"
    }
]
```

### Delete a bucket user

DELETE `/v1/s3/{account}/buckets/{bucket}/users/{user}

| Response Code                 | Definition                               |  
| ----------------------------- | -----------------------------------------|  
| **200 OK**                    | deleted user                             |  
| **400 Bad Request**           | badly formed request                     |  
| **403 Forbidden**             | you don't have access to delete the user |  
| **404 Not Found**             | account or user not found                |  
| **429 Too Many Requests**     | service or rate limit exceeded           |  
| **500 Internal Server Error** | a server error occurred                  |

### Create a website

POST `/v1/s3/{account}/websites`

#### Request

```json
{
    "Tags": [
        { "Key": "Application", "Value": "HowToGet" },
        { "Key": "COA" "Value", "Value": "Take.My.Money.$$$$" },
        { "Key": "CreatedBy", "Value": "Big Bird" }
    ],
    "BucketInput": {
        "Bucket": "foobar.bulldogs.cloud"
    },
    "WebsiteConfiguration": {
        "IndexDocument": { "Suffix": "index.html" }
    }
}
```

#### Response

```json
{
    "Bucket": "/foobar.bulldogs.cloud",
    "Policy": {
        "Arn": "arn:aws:iam::12345678910:policy/foobar.bulldogs.cloud-BktAdmPlc",
        "AttachmentCount": 0,
        "CreateDate": "2019-03-01T15:33:52Z",
        "DefaultVersionId": "v1",
        "Description": null,
        "IsAttachable": true,
        "Path": "/",
        "PermissionsBoundaryUsageCount": 0,
        "PolicyId": "ABCDEFGHI12345678",
        "PolicyName": "foobar.bulldogs.cloud-BktAdmPlc",
        "UpdateDate": "2019-03-01T15:33:52Z"
    },
    "Group": {
        "Arn": "arn:aws:iam::12345678910:group/foobar.bulldogs.cloud-BktAdmGrp",
        "CreateDate": "2019-03-01T15:33:52Z",
        "GroupId": "GROUPID123",
        "GroupName": "foobar.bulldogs.cloud-BktAdmGrp",
        "Path": "/"
    },
    "Distribution": {
        "ARN": "arn:aws:cloudfront::12345678910:distribution/ABCDEFGHIJKL",
        "DistributionConfig": {
            "Aliases": {
                "Items": [
                    "foobar.bulldogs.cloud"
                ],
                "Quantity": 1
            },
            "CallerReference": "12345678-9012-3456-6789-094d26464c6c",
            "Comment": "foobar.bulldogs.cloud",
            "DefaultCacheBehavior": {
                ...
                "TargetOriginId": "foobar.bulldogs.cloud",
                "TrustedSigners": {
                    "Enabled": false,
                    "Items": null,
                    "Quantity": 0
                },
                "ViewerProtocolPolicy": "redirect-to-https"
            },
            "DefaultRootObject": "index.html",
            "Enabled": true,
            "HttpVersion": "http2",
            "IsIPV6Enabled": true,
            "Logging": {
                "Bucket": "",
                "Enabled": false,
                "IncludeCookies": false,
                "Prefix": ""
            },
            "Origins": {
                "Items": [
                    {
                        ...
                        "CustomOriginConfig": {
                            "HTTPPort": 80,
                            "HTTPSPort": 443,
                            "OriginKeepaliveTimeout": 5,
                            "OriginProtocolPolicy": "http-only",
                            "OriginReadTimeout": 30,
                            "OriginSslProtocols": {
                                "Items": [
                                    "TLSv1",
                                    "TLSv1.1",
                                    "TLSv1.2"
                                ],
                                "Quantity": 3
                            }
                        },
                        "DomainName": "foobar.bulldogs.cloud.s3-website-us-east-1.amazonaws.com",
                        "Id": "foobar.bulldogs.cloud",
                        "OriginPath": "",
                        "S3OriginConfig": null
                    }
                ],
                "Quantity": 1
            },
            "PriceClass": "PriceClass_100",
            "Restrictions": {
                "GeoRestriction": {
                    "Items": [
                        "US"
                    ],
                    "Quantity": 1,
                    "RestrictionType": "whitelist"
                }
            },
            "ViewerCertificate": {
                "ACMCertificateArn": "arn:aws:acm:us-east-1:12345678910:certificate/111111111-2222-3333-4444-55555555555",
                "Certificate": "arn:aws:acm:us-east-1:12345678910:certificate/111111111-2222-3333-4444-55555555555",
                "CertificateSource": "acm",
                "CloudFrontDefaultCertificate": null,
                "IAMCertificateId": null,
                "MinimumProtocolVersion": "TLSv1.1_2016",
                "SSLSupportMethod": "sni-only"
            },
            ...
        },
        "DomainName": "1234567abcdef.cloudfront.net",
        "Id": "ABCDEFGHIJKLMNOP",
        "InProgressInvalidationBatches": 0,
        "LastModifiedTime": "2019-05-09T10:50:35.79Z",
        "Status": "InProgress"
    },
    "DnsChange": {
        "Comment": "Created by s3-api",
        "Id": "/change/C176E51B123456",
        "Status": "PENDING",
        "SubmittedAt": "2019-05-09T10:50:37.194Z"
    }
}
```

| Response Code                 | Definition                           |  
| ----------------------------- | -------------------------------------|  
| **202 Accepted**              | creation request accepted            |  
| **400 Bad Request**           | badly formed request                 |  
| **403 Forbidden**             | you don't have access to bucket      |  
| **404 Not Found**             | account not found                    |  
| **409 Conflict**              | bucket or iam policy  already exists |
| **429 Too Many Requests**     | service or rate limit exceeded       |
| **500 Internal Server Error** | a server error occurred              |
| **503 Service Unavailable**   | an AWS service is unavailable        |

### Check if a website exists

HEAD `/v1/s3/{account}/websites/{website}`

*See [Check if a bucket exists](#check-if-a-bucket-exists)*

### Get information for a website

GET `/v1/s3/{account}/websites/{website}`

#### Response

```json
{
    "Tags": [
        { "Key": "Application", "Value": "HowToGet" },
        { "Key": "COA", "Value": "Take.My.Money.$$$$" },
        { "Key": "CreatedBy", "Value": "Big Bird" }
    ],
    "Logging": {
        "TargetBucket": "foobar-sites-access-logs",
        "TargetGrants": null,
        "TargetPrefix": "s3/foobar.bulldogs.cloud/"
    },
    "Empty": true,
    "DNSRecord": {
        "AliasTarget": {
            "DNSName": "abcdefgh12345.cloudfront.net.",
            "EvaluateTargetHealth": false,
            "HostedZoneId": "ABCDEFGHIJ12345"
        },
        "Failover": null,
        "GeoLocation": null,
        "HealthCheckId": null,
        "MultiValueAnswer": null,
        "Name": "foobar.bulldogs.cloud.",
        "Region": null,
        "ResourceRecords": null,
        "SetIdentifier": null,
        "TTL": null,
        "TrafficPolicyInstanceId": null,
        "Type": "A",
        "Weight": null
    },
    "Distribution": {
        "ARN": "arn:aws:cloudfront::12345678910:distribution/ABCDEFGHIJKL",
        "DistributionConfig": {
            "Aliases": {
                "Items": [
                    "foobar.bulldogs.cloud"
                ],
                "Quantity": 1
            },
            "CallerReference": "12345678-9012-3456-6789-094d26464c6c",
            "Comment": "foobar.bulldogs.cloud",
            "DefaultCacheBehavior": {
                ...
                "TargetOriginId": "foobar.bulldogs.cloud",
                "TrustedSigners": {
                    "Enabled": false,
                    "Items": null,
                    "Quantity": 0
                },
                "ViewerProtocolPolicy": "redirect-to-https"
            },
            "DefaultRootObject": "index.html",
            "Enabled": true,
            "HttpVersion": "http2",
            "IsIPV6Enabled": true,
            "Logging": {
                "Bucket": "",
                "Enabled": false,
                "IncludeCookies": false,
                "Prefix": ""
            },
            "Origins": {
                "Items": [
                    {
                        ...
                        "CustomOriginConfig": {
                            "HTTPPort": 80,
                            "HTTPSPort": 443,
                            "OriginKeepaliveTimeout": 5,
                            "OriginProtocolPolicy": "http-only",
                            "OriginReadTimeout": 30,
                            "OriginSslProtocols": {
                                "Items": [
                                    "TLSv1",
                                    "TLSv1.1",
                                    "TLSv1.2"
                                ],
                                "Quantity": 3
                            }
                        },
                        "DomainName": "foobar.bulldogs.cloud.s3-website-us-east-1.amazonaws.com",
                        "Id": "foobar.bulldogs.cloud",
                        "OriginPath": "",
                        "S3OriginConfig": null
                    }
                ],
                "Quantity": 1
            },
            "PriceClass": "PriceClass_100",
            "Restrictions": {
                "GeoRestriction": {
                    "Items": [
                        "US"
                    ],
                    "Quantity": 1,
                    "RestrictionType": "whitelist"
                }
            },
            "ViewerCertificate": {
                "ACMCertificateArn": "arn:aws:acm:us-east-1:12345678910:certificate/111111111-2222-3333-4444-55555555555",
                "Certificate": "arn:aws:acm:us-east-1:12345678910:certificate/111111111-2222-3333-4444-55555555555",
                "CertificateSource": "acm",
                "CloudFrontDefaultCertificate": null,
                "IAMCertificateId": null,
                "MinimumProtocolVersion": "TLSv1.1_2016",
                "SSLSupportMethod": "sni-only"
            },
            ...
        },
        "DomainName": "1234567abcdef.cloudfront.net",
        "Id": "ABCDEFGHIJKLMNOP",
        "InProgressInvalidationBatches": 0,
        "LastModifiedTime": "2019-05-09T10:50:35.79Z",
        "Status": "InProgress"
    },
}
```

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **200 OK**                    | okay                            |  
| **400 Bad Request**           | badly formed request            |  
| **500 Internal Server Error** | a server error occurred         |

### Update a website

Updating a website currently only supports updating the bucket's tags

PUT `/v1/s3/{account}/websites/{website}`

*See [Update a bucket](#update-a-bucket)*

### Delete a website

DELETE `/v1/s3/{account}/websites/{website}`

#### Response

Responds with a status code and the deleted objects

```json
{
    "Website": "foobar.bulldogs.cloud",
    "Users": [],
    "Policy": "foobar.bulldogs.cloud-BktAdmPlc",
    "Group": "foobar.bulldogs.cloud-BktAdmGrp",
    "DNSRecord": {
        "AliasTarget": {
            "DNSName": "abcdefgh12345.cloudfront.net.",
            "EvaluateTargetHealth": false,
            "HostedZoneId": "ABCDEFGHIJ12345"
        },
        "Name": "foobar.bulldogs.cloud.",
        "Type": "A",
        ...
    },
    "Distribution": {
        "ARN": "arn:aws:cloudfront::12345678910:distribution/ABCDEFGHIJKL",
        "DistributionConfig": {
            "Aliases": {
                "Items": [
                    "foobar.bulldogs.cloud"
                ],
                "Quantity": 1
            },
            ...
        },
        "DomainName": "1234567abcdef.cloudfront.net",
        "Id": "ABCDEFGHIJKLMNOP",
        "Status": "InProgress"
        ...
    },
}
```

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **200 OK**                    | deleted website                 |  
| **400 Bad Request**           | badly formed request            |  
| **403 Forbidden**             | you don't have access           |  
| **404 Not Found**             | account or website not found    |  
| **409 Conflict**              | website bucket is not empty     |
| **500 Internal Server Error** | a server error occurred         |

### Partially update a website

PATCH `/v1/s3/{account}/websites/{website}`

#### Request

```json
{
    "CacheInvalidation": ["/*"]
}
```

#### Response

Responds with a status code and the changes

```json
{
    "Invalidation": {
        "CreateTime": "2019-05-20T19:51:54.715Z",
        "Id": "GGHHIIJJKKLLOO",
        "InvalidationBatch": {
            "CallerReference": "2b0fd0c2-e683-44a0-8d4d-3922e965d4a4",
            "Paths": {
                "Items": [
                    "/*"
                ],
                "Quantity": 1
            }
        },
        "Status": "Completed"
    },
    "Location": "https://cloudfront.amazonaws.com/2018-11-05/distribution/AABBCCDDEEFF/invalidation/GGHHIIJJKKLLOO"
}
```

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **200 OK**                    | deleted website                 |  
| **400 Bad Request**           | badly formed request            |  
| **403 Forbidden**             | you don't have access           |  
| **404 Not Found**             | account or website not found    |  
| **500 Internal Server Error** | a server error occurred         |

### Create a website user

POST `/v1/s3/{account}/websites/{website}/users`

*See [Create a bucket user](#create-a-bucket-user)*

### Get a website user's details

GET `/v1/s3/{account}/websites/{website}/users/{user}`

*See [Get a bucket user's details](#get-a-bucket-users-details)*

### List users for a website

GET `/v1/s3/{account}/websites/{website}/users/{user}`

*See [List users for a bucket](#list-users-for-a-bucket)*

### Reset access keys for a website user

PUT `/v1/s3/{account}/websites/{website}/users/{user}`

*See [Reset access keys for a bucket user](#reset-access-keys-for-a-bucket-user)*

### Delete a website user

DELETE `/v1/s3/{account}/websites/{website}/users/{user}`

*See [Delete a bucket user](#delete-a-bucket-user)*


## Author

E Camden Fisher <camden.fisher@yale.edu>

## License

GNU Affero General Public License v3.0 (GNU AGPLv3)  
Copyright (c) 2019 Yale University
