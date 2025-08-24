package aws

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/joho/godotenv"
)

type Storage struct {
	Client    *s3.Client
	Presigner *s3.PresignClient
	Bucket    string
}

func InitStorage(sClient *s3.Client, sPresigner *s3.PresignClient, sBucket *string) {
	// Load .env file
	err := godotenv.Load("./storage/aws/.env")
	if err != nil {
		log.Fatal("Error loading .env file for aws s3 storage: ", err)
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		log.Fatal("Loaded aws region from .env file is not specified")
	}
	bucket := os.Getenv("BUCKET")
	if bucket == "" {
		log.Fatal("Loaded aws bucket from .env file is not specified")
	}
	*sBucket = bucket

	// LoadDefaultConfig reads AWS_SECRET_ACCESS_KEY and AWS_ACCESS_KEY_ID from
	// environment variables.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		log.Fatal("Failed to load configuration for aws: ", err)
	}
	*sClient = *s3.NewFromConfig(cfg)
	*sPresigner = *s3.NewPresignClient(sClient)

	_, err = sClient.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucket,
		ACL:    types.BucketCannedACLPrivate,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraintEuCentral1,
		},
	})
	var terr *types.BucketAlreadyOwnedByYou
	if errors.As(err, &terr) {
		return
	}
	if err != nil {
		log.Fatal("Error creating an aws bucket: ", err)
	}
}
