package config

import (
	"log"
	"os"
)

type Config struct {
	Region string
	Bucket string
}

func Load() *Config {

	region := os.Getenv("AWS_REGION")
	bucket := os.Getenv("S3_BUCKET")

	if region == "" || bucket == "" {
		log.Fatal("missing AWS_REGION or S3_BUCKET environment variable")
	}
	return &Config{
		Region: region,
		Bucket: bucket,
	}
}
