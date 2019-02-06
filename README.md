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

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **202 Accepted**              | creation request accepted       |  
| **400 Bad Request**           | badly formed request            |  
| **403 Forbidden**             | you don't have access to bucket |  
| **404 Not Found**             | account not found               |  
| **409 Conflict**              | bucket already exists           |
| **500 Internal Server Error** | a server error occurred         |


### Check if a bucket exists

HEAD `/v1/s3/{account}/buckets/foobarbucketname`

| Response Code                 | Definition                      |  
| ----------------------------- | --------------------------------|  
| **200 OK**                    | bucket exists                   |  
| **403 Forbidden**             | you don't have access to bucket |  
| **404 Not Found**             | account or bucket not found     |  
| **500 Internal Server Error** | a server error occurred         |


## Author

E Camden Fisher <camden.fisher@yale.edu>

## License

The MIT License (MIT)  
Copyright (c) 2019 Yale University