package kappelas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"time"
)

const defaultBase = "https://api.kappelas.com"

var (
	retryCodes  = map[int]bool{429: true, 500: true, 502: true, 503: true, 504: true}
	retryDelays = []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second}
)

type httpClient struct {
	base       string
	maxRetries int
	auth       map[string]string
	client     *http.Client
}

func newHTTPClient(base string, maxRetries int, timeout time.Duration) *httpClient {
	return &httpClient{
		base:       base,
		maxRetries: maxRetries,
		auth:       make(map[string]string),
		client:     &http.Client{Timeout: timeout},
	}
}

func (c *httpClient) setAuth(headers map[string]string) {
	c.auth = headers
}

// do executes makeReq with retry logic for transient server errors.
func (c *httpClient) do(ctx context.Context, makeReq func() (*http.Request, error)) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		req, err := makeReq()
		if err != nil {
			return nil, err
		}
		for k, v := range c.auth {
			req.Header.Set(k, v)
		}
		resp, err := c.client.Do(req.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		if retryCodes[resp.StatusCode] && attempt < c.maxRetries {
			resp.Body.Close()
			delay := retryDelays[min(attempt, len(retryDelays)-1)]
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		return resp, nil
	}
}

func (c *httpClient) get(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, c.base+path, nil)
	})
}

func (c *httpClient) postJSON(ctx context.Context, path string, body any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
}

type formFile struct {
	fieldName   string
	filename    string
	contentType string
	data        []byte
}

func (c *httpClient) postForm(ctx context.Context, path string, ff formFile, fields map[string]string) (*http.Response, error) {
	data, ct, err := buildMultipartForm(ff, fields)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", ct)
		return req, nil
	})
}

func buildMultipartForm(ff formFile, fields map[string]string) ([]byte, string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	filename := ff.filename
	if filename == "" {
		filename = ff.fieldName
	}
	ct := ff.contentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	part, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {fmt.Sprintf(`form-data; name=%q; filename=%q`, ff.fieldName, filename)},
		"Content-Type":        {ct},
	})
	if err != nil {
		return nil, "", err
	}
	if _, err = part.Write(ff.data); err != nil {
		return nil, "", err
	}

	for k, v := range fields {
		if err = mw.WriteField(k, v); err != nil {
			return nil, "", err
		}
	}
	if err = mw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), mw.FormDataContentType(), nil
}

// ─── Generic response decoders ───────────────────────────────────────────────

type apiEnvelope[T any] struct {
	OK        bool      `json:"ok"`
	Result    T         `json:"result"`
	Error     string    `json:"error"`
	ErrorCode ErrorCode `json:"error_code"`
}

func decodeResponse[T any](resp *http.Response) (T, error) {
	var zero T
	defer resp.Body.Close()
	requestID := resp.Header.Get("X-Request-Id")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("reading response: %w", err)
	}

	var envelope apiEnvelope[T]
	if err = json.Unmarshal(body, &envelope); err != nil {
		return zero, &KappelaError{
			Message:   fmt.Sprintf("unexpected non-JSON response (HTTP %d)", resp.StatusCode),
			Code:      ErrCodeUpstreamError,
			Status:    resp.StatusCode,
			RequestID: requestID,
		}
	}
	if !envelope.OK {
		return zero, &KappelaError{
			Message:   envelope.Error,
			Code:      envelope.ErrorCode,
			Status:    resp.StatusCode,
			RequestID: requestID,
		}
	}
	return envelope.Result, nil
}

func httpGet[T any](ctx context.Context, c *httpClient, path string) (T, error) {
	resp, err := c.get(ctx, path)
	if err != nil {
		var zero T
		return zero, err
	}
	return decodeResponse[T](resp)
}

func httpPost[T any](ctx context.Context, c *httpClient, path string, body any) (T, error) {
	resp, err := c.postJSON(ctx, path, body)
	if err != nil {
		var zero T
		return zero, err
	}
	return decodeResponse[T](resp)
}

func httpPostForm[T any](ctx context.Context, c *httpClient, path string, ff formFile, fields map[string]string) (T, error) {
	resp, err := c.postForm(ctx, path, ff, fields)
	if err != nil {
		var zero T
		return zero, err
	}
	return decodeResponse[T](resp)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func int64Field(v int64) string    { return strconv.FormatInt(v, 10) }
func boolField(v bool) string      { return strconv.FormatBool(v) }
