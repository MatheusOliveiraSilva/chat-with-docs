package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type UploadHandler struct {
	Uploader *manager.Uploader
	Bucket   string
}

func NewUploadHandler(u *manager.Uploader, bucket string) *UploadHandler {
	return &UploadHandler{Uploader: u, Bucket: bucket}
}

func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Validate if we are using POST method
	if err := validatePost(r); err != nil {
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		return
	}

	// 2. Create a temporary file and cleanup function
	tmpFile, cleanup, err := createTempFile()
	if err != nil {
		http.Error(w, fmt.Sprintf("tmp file: %v", err), http.StatusInternalServerError)
		return
	}
	defer cleanup()

	// 3. Read, write to temp file and calculate hash/size
	hash, size, err := hashAndSave(r.Body, tmpFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("save: %v", err), http.StatusBadRequest)
		return
	}

	// 4. Send to S3 and get location
	location, err := h.uploadToS3(r.Context(), tmpFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("s3: %v", err), http.StatusBadGateway)
		return
	}

	// 5. Respond JSON to client
	respondJSON(w, location, hash, size)
}

// validatePost ensures that we are using only the POST method
func validatePost(r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("use POST method")
	}
	return nil
}

// createTempFile creates a temporary file and returns a cleanup function
func createTempFile() (*os.File, func(), error) {
	tmp, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return nil, nil, err
	}
	return tmp, func() { os.Remove(tmp.Name()) }, nil
}

// hashAndSave reads from src, writes to tmp and returns hash and size
func hashAndSave(src io.ReadCloser, tmp *os.File) (string, int64, error) {
	defer src.Close()

	// Limits the reading to 1GB
	r := io.LimitReader(src, 1<<30)

	// MultiWriter writes simultaneously to the file and the hash
	hasher := sha256.New()
	mw := io.MultiWriter(tmp, hasher)

	// Copies in blocks of 32KB
	buf := make([]byte, 32*1024)
	written, err := io.CopyBuffer(mw, r, buf)
	if err != nil {
		return "", 0, err
	}

	// Ensures flush to disk and repositions cursor
	if err := tmp.Sync(); err != nil {
		return "", 0, err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return "", 0, err
	}

	// Returns hash in hex and size in bytes
	return hex.EncodeToString(hasher.Sum(nil)), written, nil
}

// uploadToS3 executes the multipart upload and returns the object URL
func (h *UploadHandler) uploadToS3(ctx context.Context, tmp *os.File) (string, error) {
	fileID := uuid.NewString()
	key := filepath.Join("raw", fileID)

	// Timeout of 30 minutes for the entire upload
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	out, err := h.Uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(h.Bucket),
		Key:    aws.String(key),
		Body:   tmp,
	})
	if err != nil {
		return "", err
	}
	return out.Location, nil
}

// respondJSON sends the final response to the client
func respondJSON(w http.ResponseWriter, location, hash string, size int64) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"file_id":  filepath.Base(location),
		"size":     size,
		"sha256":   hash,
		"location": location,
	}
	json.NewEncoder(w).Encode(resp)
}
