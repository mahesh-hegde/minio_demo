## USAGE

Set environment variables:
	MINIO_ACCESSKEY
	MINIO_SECRETKEY

after creating a normal (non-admin) user in minio web console.

then run uploader
```
go run uploader.go
```

To run bucket notifications feature, also run in another tab
```
go run ../bucketlistener/bucket_listener.go
```
