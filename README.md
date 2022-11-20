# absto

Abstraction of file storage for golang (currently for filesystem and S3).

## Usage

```bash
Usage of absto:
  -fileSystemDirectory /data
        [filesystem] Path to directory. Default is dynamic. /data on a server and Current Working Directory in a terminal. {ABSTO_FILE_SYSTEM_DIRECTORY} (default "$(PWD)")
  -objectAccessKey string
        [s3] Storage Object Access Key {ABSTO_OBJECT_ACCESS_KEY}
  -objectBucket string
        [s3] Storage Object Bucket {ABSTO_OBJECT_BUCKET}
  -objectClass string
        [s3] Storage Object Class {ABSTO_OBJECT_CLASS}
  -objectEndpoint string
        [s3] Storage Object endpoint {ABSTO_OBJECT_ENDPOINT}
  -objectRegion string
        [s3] Storage Object Region {ABSTO_OBJECT_REGION}
  -objectSSL
        [s3] Use SSL {ABSTO_OBJECT_SSL} (default true)
  -objectSecretAccess string
        [s3] Storage Object Secret Access {ABSTO_OBJECT_SECRET_ACCESS}
  -partSize uint
        [s3] PartSize configuration {ABSTO_PART_SIZE} (default 5242880)
```
