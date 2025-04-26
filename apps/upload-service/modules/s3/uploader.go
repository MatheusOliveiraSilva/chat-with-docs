package s3

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// Uploader encapsulates S3 upload operations
type Uploader struct {
	client *manager.Uploader
	bucket string
}

// NewUploader creates a new S3 uploader
func NewUploader(client *manager.Uploader, bucket string) *Uploader {
	return &Uploader{
		client: client,
		bucket: bucket,
	}
}

// UploadFile uploads a file to S3 and returns its location
func (u *Uploader) UploadFile(ctx context.Context, file *os.File) (string, error) {
	fileID := uuid.NewString()
	key := filepath.Join("raw", fileID)

	// Timeout of 30 minutes for the entire upload
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	out, err := u.client.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return "", err
	}
	return out.Location, nil
}
