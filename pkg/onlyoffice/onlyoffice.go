package onlyoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

const (
	DefaultDirPerm fs.FileMode = 0o755
)

type Client struct {
	URL string
}

type UploadedData struct {
	FilePath string //路径和文件名
	Reader   io.Reader
}

type UploadResults struct {
	FileName     string `json:"filename"`     //文件名，包括后缀，接口返回
	DocumentType string `json:"documentType"` //文件类型，word/slide/pdf/cell
}

type ConvertData struct {
	Async      bool   `json:"async"`      // 异步转换开关(需要key)
	Filetype   string `json:"filetype"`   // 输入格式
	Key        string `json:"key"`        //
	OutPutType string `json:"outputtype"` // 输出格式
	Title      string `json:"title"`      // 文件名
	Url        string `json:"url"`        // 文件链接
}

type ConvertResults struct {
	FileName   string `json:"filename"`   //文件名，包括后缀，接口返回
	FileUrl    string `json:"fileUrl"`    // 文件下载地址
	Percent    int    `json:"percent"`    // 完成进度
	EndConvert bool   `json:"endConvert"` //是否结束转换
}

type DownloadData struct {
	FileName string //文件名，通过上传获取
}

// fileTitle:附件名（不包含路径和后缀名）suffix:后缀名
func (o Client) UploadedFile(ctx context.Context,
	upload UploadedData) (*UploadResults, error) {
	// 使用 io.Pipe 实现边写边读
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer func() {
			if closer, ok := upload.Reader.(io.Closer); ok {
				closer.Close()
			}
		}()

		// 创建 multipart 的 file part
		part, err := writer.CreateFormFile("uploadedFile", upload.FilePath)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		// 将 MinIO 的对象内容复制到 multipart 部分
		if _, err := io.Copy(part, upload.Reader); err != nil {
			pw.CloseWithError(err)
			return
		}

		// 关闭 multipart writer，写入结束
		if err := writer.Close(); err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	// go o.writeMultipartFile(pw, writer, upload.FilePath, upload.Reader)

	// 构造 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.URL+"example/upload", pr)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Accept", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	var data UploadResults
	// 打印响应
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("上传结果：%s\n响应内容：%s\n", resp.Status, string(respBody))

	if resp.StatusCode == http.StatusOK {
		// 解析 resp.Body  { "filename": "测试.doc", "documentType": "word" }
		if err := json.Unmarshal(respBody, &data); err != nil {
			return nil, fmt.Errorf("解析相映失败: %w", err)
		}
	}
	return &data, nil
}

// 发起转换请求
func (o Client) ConvertFile(ctx context.Context,
	convert ConvertData) (*ConvertResults, error) {
	// 构造 JSON 请求体
	reqBody := map[string]any{
		"async":      convert.Async,
		"filetype":   convert.Filetype,   // 源文件类型，如 docx
		"key":        convert.Key,        // 可用文件名做唯一标识
		"outputtype": convert.OutPutType, // 目标类型，可根据需要传参
		"title":      convert.Title,      // 文件标题
		"url":        convert.Url,        // 这里应传文件下载地址，需根据实际参数调整
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.URL+"/converter", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("创建转换请求失败: %w", err)
	}
	httpReq.Header.Set("Pragma", "no-cache")
	httpReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送转换请求失败: %w", err)
	}
	defer resp.Body.Close()

	var data ConvertResults
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("转换结果：%s\n响应内容：%s\n", resp.Status, string(respBody))
	if resp.StatusCode == http.StatusOK {
		// 解析 resp.Body  { "filename": "测试.doc", "documentType": "word" }
		if err := json.Unmarshal(respBody, &data); err != nil {
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
	}
	return &data, nil
}

// 下载文件 使用时需要Close流
func (o Client) DownloadFile(ctx context.Context, download DownloadData) (io.Reader, error) {
	// 创建 GET 请求
	baseURL := o.URL + "example/download"
	encoded := url.QueryEscape(download.FileName)
	reqURL := fmt.Sprintf("%s?fileName=%s", baseURL, encoded)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %v", err)
	}

	httpReq.Header.Set("Pragma", "no-cache")
	httpReq.Header.Set("Upgrade-Insecure-Requests", "1")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求发送失败: %v", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close() // 这里只能提前关闭错误响应体
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求失败，状态码: %d,响应：%s", resp.StatusCode, string(bodyBytes))
	}
	return resp.Body, nil
}
