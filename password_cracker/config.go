package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const (
	awsBucket = "cracking-algorithm-data"
	awsRegion = "sa-east-1"

	awsAccessKeyID     = ""
	awsSecretAccessKey = ""
)

func LoadAWSConfig() (aws.Config, error) {
	credsProvider := credentials.NewStaticCredentialsProvider(awsAccessKeyID, awsSecretAccessKey, "")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
		config.WithCredentialsProvider(credsProvider),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load configuration, %v", err)
	}
	return cfg, nil
}
