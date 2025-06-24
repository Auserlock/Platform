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

	// 为常规请求设置一个默认超时
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
	return strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (c *HttpClient) doRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 应用客户端的全局 headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// 应用本次请求特定的 headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 为了让 context 完全控制请求的生命周期（特别是超时），
	// 我们克隆原始客户端以继承其所有设置（例如 Transport），然后禁用其固有的 Timeout。
	// 这可以防止 http.Client.Timeout 与 context 的超时发生冲突。
	requestClient := *c.client
	requestClient.Timeout = 0

	resp, err := requestClient.Do(req)
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
	// 为常规请求应用默认的短超时
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	return c.GetWithContext(ctx, path, params, headers...)
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
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	return c.PostFormWithContext(ctx, path, data, headers...)
}

func (c *HttpClient) PostFormWithContext(ctx context.Context, path string, data map[string]string, headers ...map[string]string) (*Response, error) {
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

// PostMultipart (旧方法): 将文件读入内存后上传
func (c *HttpClient) PostMultipart(path string, data map[string]string, files map[string]string) (*Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	return c.PostMultipartWithContext(ctx, path, data, files)
}

// PostMultipartWithContext (旧方法): 将文件读入内存后上传
func (c *HttpClient) PostMultipartWithContext(ctx context.Context, path string, data map[string]string, files map[string]string) (*Response, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, val := range data {
		_ = writer.WriteField(key, val)
	}

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

// PostMultipartStream: 使用流式方法上传文件，内置了10分钟的超时时间。
func (c *HttpClient) PostMultipartStream(path string, data map[string]string, files map[string]string) (*Response, error) {
	// 为流式上传创建一个带有10分钟超时的专用上下文。
	uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	// 确保在函数返回后（无论成功还是失败）都能释放上下文相关资源。
	defer cancel()

	// 使用带有长超时的上下文来调用实际的执行函数。
	return c.PostMultipartStreamWithContext(uploadCtx, path, data, files)
}

// PostMultipartStreamWithContext: 允许调用者传入自定义的上下文，以实现更精细的超时控制或取消操作。
func (c *HttpClient) PostMultipartStreamWithContext(ctx context.Context, path string, data map[string]string, files map[string]string) (*Response, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		// 【核心修改】将写入逻辑包装在一个函数中，以便更清晰地处理错误和资源关闭。
		// 在 goroutine 退出时，总是关闭管道写入器，并传递遇到的任何错误。
		var err error
		defer func() {
			pw.CloseWithError(err)
		}()

		// 写入文本字段
		for key, val := range data {
			if err = writer.WriteField(key, val); err != nil {
				err = fmt.Errorf("写入表单字段 %s 失败: %w", key, err)
				return
			}
		}

		// 写入文件字段
		for key, filePath := range files {
			var file *os.File
			file, err = os.Open(filePath)
			if err != nil {
				err = fmt.Errorf("打开文件 %s 失败: %w", filePath, err)
				return
			}
			defer file.Close()

			var part io.Writer
			part, err = writer.CreateFormFile(key, filepath.Base(filePath))
			if err != nil {
				err = fmt.Errorf("为文件 %s 创建 form-data part 失败: %w", filePath, err)
				return
			}

			if _, err = io.Copy(part, file); err != nil {
				err = fmt.Errorf("拷贝文件 %s 内容失败: %w", filePath, err)
				return
			}
		}

		// 至关重要：必须在所有部分写入后关闭 multipart writer，
		// 这会写入最后的边界标记。
		err = writer.Close()

	}()

	reqHeaders := make(map[string]string)
	reqHeaders["Content-Type"] = writer.FormDataContentType()

	fullURL := c.buildURL(path)
	return c.doRequest(ctx, http.MethodPost, fullURL, pr, reqHeaders)
}

// --- Response 方法 ---

func (r *Response) String() string {
	return string(r.Body)
}

func (r *Response) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}
