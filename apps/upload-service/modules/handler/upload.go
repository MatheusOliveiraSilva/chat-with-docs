package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	s3Util "github.com/MatheusOliveiraSilva/chat-with-docs/apps/upload-service/modules/s3"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
)

type UploadHandler struct {
	s3Uploader *s3Util.Uploader
}

func NewUploadHandler(u *manager.Uploader, bucket string) *UploadHandler {
	return &UploadHandler{
		s3Uploader: s3Util.NewUploader(u, bucket),
	}
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
	location, err := h.s3Uploader.UploadFile(r.Context(), tmpFile)
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
