package aws

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
)

var (
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
)

func InitStorage() {
	// Load .env file
	err := godotenv.Load("./storage/aws/.env")
	if err != nil {
		log.Fatal("Error loading .env file for aws s3 storage:", err)
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		log.Fatal("Loaded aws region from .env file is not specified")
	}
	bucket = os.Getenv("BUCKET")
	if bucket == "" {
		log.Fatal("Loaded aws bucket from .env file is not specified")
	}

	// LoadDefaultConfig reads AWS_SECRET_ACCESS_KEY and AWS_ACCESS_KEY_ID from
	// environment variables.
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatal("Failed to load configuration for aws:", err)
	}
	client = s3.NewFromConfig(cfg)
	presigner = s3.NewPresignClient(client)
}
