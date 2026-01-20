package kms

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
)

// SignRequest 表示 MPC-KMS 签名请求
type SignRequest struct {
	Data         string       `json:"data"`
	DataEncoding string       `json:"data_encoding,omitempty"`
	Summary      *SignSummary `json:"summary,omitempty"`
	CallbackURL  string       `json:"callback_url,omitempty"`
}

// SignSummary 表示签名数据摘要
type SignSummary struct {
	Type   string `json:"type"`
	From   string `json:"from"`
	To     string `json:"to"`
	Amount string `json:"amount"`
	Remark string `json:"remark,omitempty"`
	Token  string `json:"token"`
}

// SignResponse 表示 MPC-KMS 签名响应
type SignResponse struct {
	Signature string `json:"signature"`
}

// TaskResponse 表示 MPC-KMS 任务响应（需要审批时返回）
type TaskResponse struct {
	TaskID string `json:"task_id"`
}

// TaskStatus 表示任务状态
type TaskStatus string

const (
	TaskStatusPendingApproval TaskStatus = "PENDING_APPROVAL"
	TaskStatusApproved        TaskStatus = "APPROVED"
	TaskStatusRejected        TaskStatus = "REJECTED"
	TaskStatusDone            TaskStatus = "DONE"
	TaskStatusFailed          TaskStatus = "FAILED"
)

// TaskResult 表示任务结果
type TaskResult struct {
	Status   TaskStatus `json:"status"`
	Message  string     `json:"msg,omitempty"`
	Response string     `json:"response,omitempty"`
}

// ErrorResponse 表示 MPC-KMS 错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// DataEncoding 表示数据编码格式
type DataEncoding string

const (
	DataEncodingPlain  DataEncoding = "PLAIN"
	DataEncodingBase64 DataEncoding = "BASE64"
	DataEncodingHex    DataEncoding = "HEX"
)

// SummaryType 表示摘要类型
type SummaryType string

const (
	SummaryTypeTransfer SummaryType = "TRANSFER"
)

// NewSignRequest 创建新的签名请求
func NewSignRequest(data []byte, encoding DataEncoding) *SignRequest {
	var dataStr string
	switch encoding {
	case DataEncodingBase64:
		dataStr = base64.StdEncoding.EncodeToString(data)
	case DataEncodingHex:
		dataStr = hex.EncodeToString(data)
	default: // DataEncodingPlain
		dataStr = string(data)
	}

	return &SignRequest{
		Data:         dataStr,
		DataEncoding: string(encoding),
	}
}

// WithSummary 为签名请求添加摘要
func (r *SignRequest) WithSummary(summary *SignSummary) *SignRequest {
	r.Summary = summary
	return r
}

// WithCallbackURL 为签名请求添加回调URL
func (r *SignRequest) WithCallbackURL(url string) *SignRequest {
	r.CallbackURL = url
	return r
}

// NewTransferSummary 创建转账摘要
func NewTransferSummary(from, to, amount, token, remark string) *SignSummary {
	return &SignSummary{
		Type:   string(SummaryTypeTransfer),
		From:   from,
		To:     to,
		Amount: amount,
		Token:  token,
		Remark: remark,
	}
}

// Marshal 序列化签名请求
func (r *SignRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// UnmarshalSignResponse 反序列化签名响应
func UnmarshalSignResponse(data []byte) (*SignResponse, error) {
	var resp SignResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UnmarshalTaskResponse 反序列化任务响应
func UnmarshalTaskResponse(data []byte) (*TaskResponse, error) {
	var resp TaskResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UnmarshalTaskResult 反序列化任务结果
func UnmarshalTaskResult(data []byte) (*TaskResult, error) {
	var result TaskResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UnmarshalErrorResponse 反序列化错误响应
func UnmarshalErrorResponse(data []byte) (*ErrorResponse, error) {
	var errResp ErrorResponse
	if err := json.Unmarshal(data, &errResp); err != nil {
		return nil, err
	}
	return &errResp, nil
}
