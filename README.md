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
GET /v1/s3/{account}/buckets/{bucket}
```

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

```json
{
    "Bucket": "foobarbucketname"
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

### Check if a bucket exists

HEAD `/v1/s3/{account}/buckets/foobarbucketname`

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **200 OK**                    | bucket exists                   |  
| **403 Forbidden**             | you don't have access to bucket |  
| **404 Not Found**             | account or bucket not found     |  
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

## Author

E Camden Fisher <camden.fisher@yale.edu>

## License

The MIT License (MIT)  
Copyright (c) 2019 Yale University