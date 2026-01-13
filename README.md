## About
Currently you can:
- Upload, download and delete files
- Resume and abort in-progress uploads
- Create and delete folders
- Change file/folder names
- Create and delete repositories
- Share files by making a repository public

This project can be run with storage in the cloud (aws s3) or locally (seaweedfs s3)

### Example images
Uploading a file
<img width="3829" height="1903" alt="image" src="https://github.com/user-attachments/assets/008d93c9-653f-4cb0-9b25-aee7589d66fd" />
Folder with total file size
<img width="2181" height="157" alt="image" src="https://github.com/user-attachments/assets/3eb2e239-7497-4c44-8ce9-89667e66848c" />




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
- Create in aws using the IAM service AWS_SECRET_ACCESS_KEY and AWS_ACCESS_KEY_ID
- Create a ".env" file in this project's files in backend/storage/aws/ containing:
```
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_ACCESS_KEY_ID=your-access-key
AWS_REGION=your-s3-bucket-region
BUCKET=your-s3-bucket-name
```
- Run:
```bash
docker compose -f compose.cloud.yaml up --build --attach backend
```
Once the backend service prints "starting server" the app should be available on: http://localhost:5173

## How to run with docker and local s3
- Run:
```bash
docker compose -f compose.local.yaml up --build --attach backend
```
Once the backend service prints "starting server" the app should be available on: http://localhost:5173

## How to set up accounts once the app is running
After creating a user account, in order to be able to upload files you have to set in the database that user's role to admin or user, and space to how many bytes that user can upload

You can do it by running these commands:
```bash
docker exec -it database sh
psql
\c app
UPDATE user_ SET role_ = 'user', space_ = 1000000000 WHERE username_ = 'username';
```
