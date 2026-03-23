package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	pkgErrors "jusha/mcp/pkg/errors"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/model"

	httptransport "github.com/go-kit/kit/transport/http"
)

const (
	// JavaScript Number.MAX_SAFE_INTEGER
	jsMaxSafeInteger = 1<<53 - 1    // 9007199254740991
	jsMinSafeInteger = -(1<<53 - 1) // -9007199254740991

	// requestStartTimeKey 是用于存储请求开始时间的 context 键
	requestStartTimeKey contextKey = "requestStartTime"
)

func NewSuccessResponse(data any) model.ApiResponse {
	return model.ApiResponse{
		Code:    pkgErrors.SUCCESS.Code(),
		Message: "Success",
		Data:    data,
	}
}

func NewErrorResponse(code int, message string) model.ApiResponse {
	return model.ApiResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

// NewErrorResponseFromError 从错误创建响应，自动提取错误码
func NewErrorResponseFromError(err error) model.ApiResponse {
	if err == nil {
		return NewSuccessResponse(nil)
	}

	// 获取错误码
	errorCode := pkgErrors.GetErrorCode(err)

	return model.ApiResponse{
		Code:    errorCode.Code(),
		Message: err.Error(),
		Data:    nil,
	}
}

// convertInt64ToString 递归转换响应中超出前端精度的int64为string
func convertInt64ToString(data any) any {
	if data == nil {
		return nil
	}

	// 特殊处理 time.Time 类型，保持不变
	if _, ok := data.(time.Time); ok {
		return data
	}

	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.Int64:
		val := v.Int()
		// 只有超出 JavaScript Number 安全范围的才转换为string
		if val > jsMaxSafeInteger || val < jsMinSafeInteger {
			return strconv.FormatInt(val, 10)
		}
		return val
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return convertInt64ToString(v.Elem().Interface())
	case reflect.Slice, reflect.Array:
		result := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = convertInt64ToString(v.Index(i).Interface())
		}
		return result
	case reflect.Map:
		result := make(map[string]any)
		for _, key := range v.MapKeys() {
			keyStr := key.String()
			result[keyStr] = convertInt64ToString(v.MapIndex(key).Interface())
		}
		return result
	case reflect.Struct:
		// 特殊处理 time.Time 结构体
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return data
		}
		result := make(map[string]any)
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.IsExported() {
				fieldName := field.Name
				if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
					fieldName = strings.Split(jsonTag, ",")[0]
				}
				result[fieldName] = convertInt64ToString(v.Field(i).Interface())
			}
		}
		return result
	default:
		return data
	}
}

func EncodeHTTPResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	// 转换int64为string
	convertedResponse := convertInt64ToString(response)
	kgResp := NewSuccessResponse(convertedResponse)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(kgResp)
}

func EncodeHTTPError(logger logging.ILogger) httptransport.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		if err == nil {
			logger.ErrorContext(ctx, "encodeHTTPError called with nil error")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(NewErrorResponse(http.StatusInternalServerError, "未知错误"))
			return
		}

		logger.ErrorContext(ctx, "HTTP transport error", "error", err)

		// 使用统一的错误处理
		httpStatusCode := pkgErrors.GetHTTPStatusCode(err)
		errorCode := pkgErrors.GetErrorCode(err)

		kgResp := model.ApiResponse{
			Code:    errorCode.Code(),
			Message: err.Error(),
			Data:    nil,
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(httpStatusCode)
		json.NewEncoder(w).Encode(kgResp)
	}
}

func SendHTTPRequest(ctx context.Context, method, url string, headers map[string]string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
	} else {
		req.Body = http.NoBody
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, pkgErrors.NewHTTPError(resp.StatusCode, "HTTP request failed")
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBytes, nil
}

func GetRequestStartTime(ctx context.Context) (startTime time.Time, hasStartTime bool) {
	startTime, ok := ctx.Value(requestStartTimeKey).(time.Time)
	if !ok {
		return time.Time{}, false
	}
	return startTime, true
}
func SetRequestStartTime(ctx context.Context, startTime time.Time) context.Context {
	return context.WithValue(ctx, requestStartTimeKey, startTime)
}

func IsUriInWhitelist(uri string, whitelist []string) bool {
	for _, whitelistUri := range whitelist {
		if uri == whitelistUri {
			return true
		}
	}
	return false
}

func IsUriOpen(uri string) bool {
	return strings.Contains(uri, "/open")
}

func IsUriForbid(uri string) bool {
	// 检查环境变量，如果设置为 disabled 则不拦截
	if os.Getenv("REFUSE_FORBID_URI") == "disabled" {
		return false
	}
	return strings.Contains(uri, "/forbid")
}
