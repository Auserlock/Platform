package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type HttpClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
}

type Config struct {
	BaseURL string
	Timeout time.Duration
	Headers map[string]string
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	RawResp    *http.Response
}

func NewClient(config *Config) *HttpClient {
	if config == nil {
		config = &Config{}
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	headers := make(map[string]string)
	if config.Headers != nil {
		for k, v := range config.Headers {
			headers[k] = v
		}
	}

	return &HttpClient{
		client:  client,
		baseURL: config.BaseURL,
		headers: headers,
	}
}

func (c *HttpClient) SetHeader(key, value string) {
	c.headers[key] = value
}

func (c *HttpClient) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		c.headers[k] = v
	}
}

func (c *HttpClient) buildURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.baseURL + "/" + strings.TrimLeft(path, "/")
}

func (c *HttpClient) doRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       bodyBytes,
		RawResp:    resp,
	}, nil
}

func (c *HttpClient) Get(path string, params map[string]string, headers ...map[string]string) (*Response, error) {
	return c.GetWithContext(context.Background(), path, params, headers...)
}

func (c *HttpClient) GetWithContext(ctx context.Context, path string, params map[string]string, headers ...map[string]string) (*Response, error) {
	fullURL := c.buildURL(path)

	if len(params) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse url: %w", err)
		}
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}

	var reqHeaders map[string]string
	if len(headers) > 0 {
		reqHeaders = headers[0]
	}

	return c.doRequest(ctx, http.MethodGet, fullURL, nil, reqHeaders)
}

func (c *HttpClient) Post(path string, data interface{}, headers ...map[string]string) (*Response, error) {
	return c.PostWithContext(context.Background(), path, data, headers...)
}

func (c *HttpClient) PostWithContext(ctx context.Context, path string, data interface{}, headers ...map[string]string) (*Response, error) {
	return c.requestWithBody(ctx, http.MethodPost, path, data, headers...)
}

func (c *HttpClient) Put(path string, data interface{}, headers ...map[string]string) (*Response, error) {
	return c.PutWithContext(context.Background(), path, data, headers...)
}

func (c *HttpClient) PutWithContext(ctx context.Context, path string, data interface{}, headers ...map[string]string) (*Response, error) {
	return c.requestWithBody(ctx, http.MethodPut, path, data, headers...)
}

func (c *HttpClient) Patch(path string, data interface{}, headers ...map[string]string) (*Response, error) {
	return c.PatchWithContext(context.Background(), path, data, headers...)
}

func (c *HttpClient) PatchWithContext(ctx context.Context, path string, data interface{}, headers ...map[string]string) (*Response, error) {
	return c.requestWithBody(ctx, http.MethodPatch, path, data, headers...)
}

func (c *HttpClient) Delete(path string, headers ...map[string]string) (*Response, error) {
	return c.DeleteWithContext(context.Background(), path, headers...)
}

func (c *HttpClient) DeleteWithContext(ctx context.Context, path string, headers ...map[string]string) (*Response, error) {
	fullURL := c.buildURL(path)

	var reqHeaders map[string]string
	if len(headers) > 0 {
		reqHeaders = headers[0]
	}

	return c.doRequest(ctx, http.MethodDelete, fullURL, nil, reqHeaders)
}

func (c *HttpClient) requestWithBody(ctx context.Context, method, path string, data interface{}, headers ...map[string]string) (*Response, error) {
	fullURL := c.buildURL(path)

	var body io.Reader
	var contentType string

	if data != nil {
		switch v := data.(type) {
		case string:
			body = strings.NewReader(v)
			contentType = "text/plain"
		case []byte:
			body = bytes.NewReader(v)
			contentType = "application/octet-stream"
		case io.Reader:
			body = v
		default:
			jsonData, err := json.Marshal(data)
			if err != nil {
				return nil, fmt.Errorf("failed marshal json: %w", err)
			}
			body = bytes.NewReader(jsonData)
			contentType = "application/json"
		}
	}

	reqHeaders := make(map[string]string)
	if len(headers) > 0 && headers[0] != nil {
		for k, v := range headers[0] {
			reqHeaders[k] = v
		}
	}

	if contentType != "" && reqHeaders["Content-Type"] == "" {
		reqHeaders["Content-Type"] = contentType
	}

	return c.doRequest(ctx, method, fullURL, body, reqHeaders)
}

func (c *HttpClient) PostForm(path string, data map[string]string, headers ...map[string]string) (*Response, error) {
	return c.PostFormWithContext(context.Background(), path, data, headers...)
}

func (c *HttpClient) PostFormWithContext(ctx context.Context, path string, data map[string]string, headers ...map[string]string) (*Response, error) {
	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}

	reqHeaders := make(map[string]string)
	reqHeaders["Content-Type"] = "application/x-www-form-urlencoded"

	if len(headers) > 0 && headers[0] != nil {
		for k, v := range headers[0] {
			reqHeaders[k] = v
		}
	}

	return c.requestWithBody(ctx, http.MethodPost, path, strings.NewReader(form.Encode()), reqHeaders)
}

func (c *HttpClient) PostMultipart(path string, data map[string]string, files map[string]string) (*Response, error) {
	return c.PostMultipartWithContext(context.Background(), path, data, files)
}

func (c *HttpClient) PostMultipartWithContext(ctx context.Context, path string, data map[string]string, files map[string]string) (*Response, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 写入文本字段
	for key, val := range data {
		if err := writer.WriteField(key, val); err != nil {
			return nil, fmt.Errorf("写入表单字段 %s 失败: %w", key, err)
		}
	}

	// 写入文件字段
	for key, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("打开文件 %s 失败: %w", filePath, err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile(key, filepath.Base(filePath))
		if err != nil {
			return nil, fmt.Errorf("为文件 %s 创建 form-data part 失败: %w", filePath, err)
		}

		if _, err = io.Copy(part, file); err != nil {
			return nil, fmt.Errorf("拷贝文件 %s 内容失败: %w", filePath, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭 multipart writer 失败: %w", err)
	}

	reqHeaders := make(map[string]string)
	reqHeaders["Content-Type"] = writer.FormDataContentType()

	fullURL := c.buildURL(path)
	return c.doRequest(ctx, http.MethodPost, fullURL, body, reqHeaders)
}

func (r *Response) String() string {
	return string(r.Body)
}

func (r *Response) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}
