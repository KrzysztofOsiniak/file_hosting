## Main features
Currently you can:
- Upload, download and delete files
- Resume and abort in-progress uploads
- Create and delete folders
- Change file/folder names
- Create and delete repositories
- Share files by making a repository public

## How to run with docker and aws s3
- Create in aws an s3 bucket
- Go to your bucket -> permissions -> scroll down to "Cross-origin resource sharing" and change it to:
```json
[
    {
        "AllowedHeaders": [
            "*"
        ],
        "AllowedMethods": [
            "PUT",
            "DELETE",
            "POST",
            "GET",
            "HEAD"
        ],
        "AllowedOrigins": [
            "*"
        ],
        "ExposeHeaders": [
            "ETag"
        ],
        "MaxAgeSeconds": 3000
    }
]
```
- Create in aws IAM service AWS_SECRET_ACCESS_KEY and AWS_ACCESS_KEY_ID
- Create a ".env" file in backend/storage/aws/ containing:
```
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_ACCESS_KEY_ID=your-access-key
AWS_REGION=your-s3-bucket-region
BUCKET=your-s3-bucket-name
```
- Run:
```bash
docker compose -f compose.cloud.yaml up --build
```

## How to run with docker and local s3
- Set "-volumeSizeLimitMB=1000" in compose.local.yaml to desired space for files
- Run:
```bash
docker compose -f compose.local.yaml up --build
```

## How to set up accounts once app is running
