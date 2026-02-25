package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// GetInboxID fetches the user's email and returns its ID.
func (c *Client) GetInboxID() (int, error) {
	email, err := c.GetEmail()
	if err != nil {
		return 0, err
	}
	return email.ID, nil
}

// PresignAttachment requests a presigned PUT URL for uploading an attachment.
func (c *Client) PresignAttachment(req PresignRequest) (*PresignResponse, error) {
	var result PresignResponse
	if err := c.doAuthenticatedRequest(http.MethodPost, PathEmailAttachmentPresign, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UploadToPresignedURL uploads a file to a presigned R2 URL via HTTP PUT.
// This bypasses the normal JSON API client — it's a raw binary upload.
func (c *Client) UploadToPresignedURL(uploadURL, filePath, contentType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, uploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = stat.Size()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// ComposeEmail sends a new email from the given inbox.
func (c *Client) ComposeEmail(inboxID int, req ComposeRequest) (*EmailMessageDetail, error) {
	path := fmt.Sprintf("%s?inbox=%d", PathEmailCompose, inboxID)
	var result EmailMessageDetail
	if err := c.doAuthenticatedRequest(http.MethodPost, path, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReplyEmail sends a reply to a specific email message.
func (c *Client) ReplyEmail(messageID string, req ReplyRequest) (*EmailMessageDetail, error) {
	path := PathEmailMessages + url.PathEscape(messageID) + "/reply/"
	var result EmailMessageDetail
	if err := c.doAuthenticatedRequest(http.MethodPost, path, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReplyAllEmail sends a reply-all to a specific email message.
func (c *Client) ReplyAllEmail(messageID string, req ReplyRequest) (*EmailMessageDetail, error) {
	path := PathEmailMessages + url.PathEscape(messageID) + "/reply-all/"
	var result EmailMessageDetail
	if err := c.doAuthenticatedRequest(http.MethodPost, path, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
