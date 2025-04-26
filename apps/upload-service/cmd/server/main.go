package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"

	"github.com/MatheusOliveiraSilva/chat-with-docs/apps/upload-service/modules/config"
	"github.com/MatheusOliveiraSilva/chat-with-docs/apps/upload-service/modules/handler"
)

func main() {
	// 1. load config
	cfg := config.Load()

	// 2. initialize AWS SDK with automatic credentials
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(), awsConfig.WithRegion(cfg.Region))
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}
	s3Client := s3.NewFromConfig(awsCfg)

	// 3. create uploader for multipart in S3
	uploader := manager.NewUploader(s3Client)

	// 4. create upload handler with dependencies injected internally
	uploadHandler := handler.NewUploadHandler(uploader, cfg.Bucket)

	// 5. configure routing with Chi
	r := chi.NewRouter()
	r.Post("/v1/upload", uploadHandler.ServeHTTP)

	// 6. start HTTP server
	addr := ":8080"
	fmt.Printf("Upload-service listening on %s\n", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
