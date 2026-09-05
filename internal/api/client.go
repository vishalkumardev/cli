package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client is the central HTTP client for BuildShare API calls.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// New creates a new API client.
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// APIResponse mirrors the backend's standard envelope.
type APIResponse struct {
	StatusCode int             `json:"statusCode"`
	Success    bool            `json:"success"`
	Message    string          `json:"message"`
	Data       json.RawMessage `json:"data"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return req, nil
}

func (c *Client) do(req *http.Request) (*APIResponse, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w\n\nPlease check:\n  - your internet connection\n  - BuildShare API URL (%s)", err, c.BaseURL)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		// Non-JSON response
		return nil, fmt.Errorf("unexpected response from server (HTTP %d)", resp.StatusCode)
	}

	if !apiResp.Success {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    apiResp.Message,
		}
	}

	return &apiResp, nil
}

// ── HTTP verbs ────────────────────────────────────────────────────────────────

// Get sends a GET request and returns the parsed response.
func (c *Client) Get(ctx context.Context, path string) (*APIResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// Post sends a JSON POST request.
func (c *Client) Post(ctx context.Context, path string, payload any) (*APIResponse, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader([]byte("{}"))
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

// Delete sends a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) (*APIResponse, error) {
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// Patch sends a JSON PATCH request.
func (c *Client) Patch(ctx context.Context, path string, payload any) (*APIResponse, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader([]byte("{}"))
	}
	req, err := c.newRequest(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

// UploadFile uploads a file as multipart/form-data and reports progress.
func (c *Client) UploadFile(ctx context.Context, path, filePath, fieldName string, extra map[string]string, progressFn func(pct int)) (*APIResponse, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	totalSize := info.Size()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer mw.Close()

		// Extra text fields
		for k, v := range extra {
			_ = mw.WriteField(k, v)
		}

		part, err := mw.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		buf := make([]byte, 32*1024)
		var uploaded int64
		for {
			n, readErr := f.Read(buf)
			if n > 0 {
				if _, writeErr := part.Write(buf[:n]); writeErr != nil {
					pw.CloseWithError(writeErr)
					return
				}
				uploaded += int64(n)
				if progressFn != nil && totalSize > 0 {
					pct := int(uploaded * 100 / totalSize)
					progressFn(pct)
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				pw.CloseWithError(readErr)
				return
			}
		}
	}()

	req, err := c.newRequest(ctx, http.MethodPost, path, pr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	return c.do(req)
}

// Decode parses JSON from a RawMessage into dst.
func Decode(raw json.RawMessage, dst any) error {
	return json.Unmarshal(raw, dst)
}
