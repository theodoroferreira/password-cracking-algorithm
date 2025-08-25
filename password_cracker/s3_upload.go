package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func uploadToS3(fileName string, cfg aws.Config) error {
	client := s3.NewFromConfig(cfg)

	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file %s for upload: %w", fileName, err)
	}
	defer file.Close()

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(awsBucket), // Using bucket from config.go
		Key:    aws.String(fileName),
		Body:   file,
	})

	if err != nil {
		return fmt.Errorf("failed to upload object, %w", err)
	}

	fmt.Printf("Successfully uploaded %s to S3 bucket %s\n", fileName, awsBucket)
	return nil
}
