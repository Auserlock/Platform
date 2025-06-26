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
	"sync"
	"time"
)

type HttpClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
	mu      sync.RWMutex
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

	// 设置合理的默认超时时间
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	headers := make(map[string]string)
	if config.Headers != nil {
		for k, v := range config.Headers {
			headers[k] = v
		}
	}

	return &HttpClient{
		client:  client,
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		headers: headers,
	}
}

func (c *HttpClient) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[key] = value
}

func (c *HttpClient) SetHeaders(headers map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range headers {
		c.headers[k] = v
	}
}

func (c *HttpClient) buildURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if c.baseURL == "" {
		return strings.TrimLeft(path, "/")
	}
	return c.baseURL + "/" + strings.TrimLeft(path, "/")
}

func (c *HttpClient) doRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*Response, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if method == "" {
		return nil, fmt.Errorf("HTTP method cannot be empty")
	}
	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 线程安全地应用全局 headers
	c.mu.RLock()
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	c.mu.RUnlock()

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	requestClient := &http.Client{
		Transport:     c.client.Transport,
		CheckRedirect: c.client.CheckRedirect,
		Jar:           c.client.Jar,
		Timeout:       0,
	}

	resp, err := requestClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

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
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	return c.GetWithContext(ctx, path, params, headers...)
}

func (c *HttpClient) GetWithContext(ctx context.Context, path string, params map[string]string, headers ...map[string]string) (*Response, error) {
	fullURL := c.buildURL(path)

	if len(params) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL '%s': %w", fullURL, err)
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
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	return c.PostWithContext(ctx, path, data, headers...)
}

func (c *HttpClient) PostWithContext(ctx context.Context, path string, data interface{}, headers ...map[string]string) (*Response, error) {
	return c.requestWithBody(ctx, http.MethodPost, path, data, headers...)
}

func (c *HttpClient) Put(path string, data interface{}, headers ...map[string]string) (*Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	return c.PutWithContext(ctx, path, data, headers...)
}

func (c *HttpClient) PutWithContext(ctx context.Context, path string, data interface{}, headers ...map[string]string) (*Response, error) {
	return c.requestWithBody(ctx, http.MethodPut, path, data, headers...)
}

func (c *HttpClient) Patch(path string, data interface{}, headers ...map[string]string) (*Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	return c.PatchWithContext(ctx, path, data, headers...)
}

func (c *HttpClient) PatchWithContext(ctx context.Context, path string, data interface{}, headers ...map[string]string) (*Response, error) {
	return c.requestWithBody(ctx, http.MethodPatch, path, data, headers...)
}

func (c *HttpClient) Delete(path string, headers ...map[string]string) (*Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	return c.DeleteWithContext(ctx, path, headers...)
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
			contentType = "text/plain; charset=utf-8"
		case []byte:
			body = bytes.NewReader(v)
			contentType = "application/octet-stream"
		case io.Reader:
			body = v
			contentType = "application/octet-stream" // 为 io.Reader 设置默认 Content-Type
		default:
			jsonData, err := json.Marshal(data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal JSON: %w", err)
			}
			body = bytes.NewReader(jsonData)
			contentType = "application/json; charset=utf-8"
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
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	return c.PostFormWithContext(ctx, path, data, headers...)
}

func (c *HttpClient) PostFormWithContext(ctx context.Context, path string, data map[string]string, headers ...map[string]string) (*Response, error) {
	if data == nil {
		return nil, fmt.Errorf("form data cannot be nil")
	}

	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}

	body := strings.NewReader(form.Encode())
	reqHeaders := make(map[string]string)
	reqHeaders["Content-Type"] = "application/x-www-form-urlencoded"

	if len(headers) > 0 && headers[0] != nil {
		for k, v := range headers[0] {
			reqHeaders[k] = v
		}
	}

	fullURL := c.buildURL(path)
	return c.doRequest(ctx, http.MethodPost, fullURL, body, reqHeaders)
}

// PostMultipart: 将文件读入内存后上传（适用于小文件）
func (c *HttpClient) PostMultipart(path string, data map[string]string, files map[string]string) (*Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	return c.PostMultipartWithContext(ctx, path, data, files)
}

// PostMultipartWithContext: 将文件读入内存后上传（适用于小文件）
func (c *HttpClient) PostMultipartWithContext(ctx context.Context, path string, data map[string]string, files map[string]string) (*Response, error) {
	if data == nil && files == nil {
		return nil, fmt.Errorf("both data and files cannot be nil")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			// 记录关闭错误
			_ = closeErr
		}
	}()

	// 写入文本字段
	for key, val := range data {
		if err := writer.WriteField(key, val); err != nil {
			return nil, fmt.Errorf("failed to write form field '%s': %w", key, err)
		}
	}

	// 处理文件字段
	for key, filePath := range files {
		if err := c.addFileToMultipart(writer, key, filePath); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	reqHeaders := make(map[string]string)
	reqHeaders["Content-Type"] = writer.FormDataContentType()

	fullURL := c.buildURL(path)
	return c.doRequest(ctx, http.MethodPost, fullURL, body, reqHeaders)
}

func (c *HttpClient) addFileToMultipart(writer *multipart.Writer, fieldName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// 记录文件关闭错误
			_ = closeErr
		}
	}()

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file for '%s': %w", filePath, err)
	}

	if _, err = io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content for '%s': %w", filePath, err)
	}

	return nil
}

// PostMultipartStream: 使用流式方法上传文件，内置10分钟超时（适用于大文件）
func (c *HttpClient) PostMultipartStream(path string, data map[string]string, files map[string]string) (*Response, error) {
	// 为大文件上传设置更长的超时时间
	uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return c.PostMultipartStreamWithContext(uploadCtx, path, data, files)
}

// PostMultipartStreamWithContext: 允许自定义上下文的流式上传
func (c *HttpClient) PostMultipartStreamWithContext(ctx context.Context, path string, data map[string]string, files map[string]string) (*Response, error) {
	if data == nil && files == nil {
		return nil, fmt.Errorf("both data and files cannot be nil")
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// 在单独的 goroutine 中写入数据
	go func() {
		defer func() {
			// 确保管道写入端总是被关闭
			if closeErr := pw.Close(); closeErr != nil {
				// 记录关闭错误
				_ = closeErr
			}
		}()

		if err := c.writeMultipartData(writer, data, files); err != nil {
			// 将错误传递给管道读取端
			pw.CloseWithError(err)
			return
		}

		// 成功完成数据写入
		if err := writer.Close(); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to close multipart writer: %w", err))
			return
		}
	}()

	reqHeaders := make(map[string]string)
	reqHeaders["Content-Type"] = writer.FormDataContentType()

	fullURL := c.buildURL(path)
	return c.doRequest(ctx, http.MethodPost, fullURL, pr, reqHeaders)
}

// writeMultipartData: 辅助方法，用于写入 multipart 数据
func (c *HttpClient) writeMultipartData(writer *multipart.Writer, data map[string]string, files map[string]string) error {
	// 写入文本字段
	for key, val := range data {
		if err := writer.WriteField(key, val); err != nil {
			return fmt.Errorf("failed to write form field '%s': %w", key, err)
		}
	}

	// 写入文件字段
	for key, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file '%s': %w", filePath, err)
		}

		part, err := writer.CreateFormFile(key, filepath.Base(filePath))
		if err != nil {
			file.Close() // 立即关闭文件
			return fmt.Errorf("failed to create form file for '%s': %w", filePath, err)
		}

		_, copyErr := io.Copy(part, file)
		closeErr := file.Close() // 立即关闭文件

		if copyErr != nil {
			return fmt.Errorf("failed to copy file content for '%s': %w", filePath, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("failed to close file '%s': %w", filePath, closeErr)
		}
	}

	return nil
}


func (r *Response) String() string {
	return string(r.Body)
}

func (r *Response) JSON(v interface{}) error {
	if v == nil {
		return fmt.Errorf("destination cannot be nil")
	}
	if len(r.Body) == 0 {
		return fmt.Errorf("response body is empty")
	}
	if err := json.Unmarshal(r.Body, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}
