package api

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
)

// UploadAttachment handles the full presign-then-upload flow for a single file.
// Returns the attachment UUID on success.
func (c *Client) UploadAttachment(filePath string) (string, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot access file: %w", err)
	}

	filename := filepath.Base(filePath)
	size := stat.Size()

	if err := ValidateAttachment(filename, size); err != nil {
		return "", err
	}

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	presign, err := c.PresignAttachment(PresignRequest{
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
	})
	if err != nil {
		return "", fmt.Errorf("presign failed: %w", err)
	}

	if err := c.UploadToPresignedURL(presign.UploadURL, filePath, contentType); err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	return presign.UUID, nil
}
