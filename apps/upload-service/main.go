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

	"github.com/MatheusOliveiraSilva/chat-with-docs/apps/upload-service/config"
	"github.com/MatheusOliveiraSilva/chat-with-docs/apps/upload-service/handler"
)

func main() {
	// 1. Carrega configurações de ambiente (bucket, região)
	cfg := config.Load()

	// 2. Inicializa AWS SDK com credenciais automáticas
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(), awsConfig.WithRegion(cfg.Region))
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}
	s3Client := s3.NewFromConfig(awsCfg)

	// 3. Cria uploader para multipart no S3
	uploader := manager.NewUploader(s3Client)

	// 4. Instancia o upload handler com dependências inj intern
	uploadHandler := handler.NewUploadHandler(uploader, cfg.Bucket)

	// 5. Configura roteamento com Chi
	r := chi.NewRouter()
	r.Post("/v1/upload", uploadHandler.ServeHTTP)

	// 6. Inicia o servidor HTTP
	addr := ":8080"
	fmt.Printf("Upload-service listening on %s\n", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
