package router

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/mowind/web3signer-go/internal/signer"
	"github.com/sirupsen/logrus"
)

// SignHandler 处理签名相关的 JSON-RPC 方法
type SignHandler struct {
	*BaseHandler
	signer  *signer.MPCKMSSigner
	builder *signer.TransactionBuilder
}

// NewSignHandler 创建签名处理器
func NewSignHandler(mpcSigner *signer.MPCKMSSigner, logger *logrus.Logger) *SignHandler {
	return &SignHandler{
		BaseHandler: NewBaseHandler("sign", logger),
		signer:      mpcSigner,
		builder:     signer.NewTransactionBuilder(),
	}
}

// handleEthAccounts 处理 eth_accounts 方法
func (h *SignHandler) handleEthAccounts(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	// 返回KMS管理的地址
	kmsAddress := h.signer.Address().String()

	h.logger.WithField("address", kmsAddress).Debug("Returning KMS managed address for eth_accounts")

	return h.CreateSuccessResponse(request.ID, []string{kmsAddress})
}

// Method 返回处理器支持的方法名
func (h *SignHandler) Method() string {
	return "sign_handler" // 这个处理器处理多个方法
}

// Handle 处理 JSON-RPC 请求
func (h *SignHandler) Handle(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	h.LogRequest(request)

	// 根据方法名分发到具体的处理函数
	switch request.Method {
	case "eth_accounts":
		return h.handleEthAccounts(ctx, request)
	case "eth_sign":
		return h.handleEthSign(ctx, request)
	case "eth_signTransaction":
		return h.handleEthSignTransaction(ctx, request)
	case "eth_sendTransaction":
		return h.handleEthSendTransaction(ctx, request)
	default:
		// 不支持的签名方法
		return h.CreateErrorResponse(request.ID, jsonrpc.CodeMethodNotFound,
			"Method not supported by sign handler", nil), nil
	}
}

// handleEthSign 处理 eth_sign 方法
func (h *SignHandler) handleEthSign(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	// 解析参数
	address, data, err := signer.ParseSignParams(request.Params)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to parse eth_sign params")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	// 验证地址匹配（转换为小写比较）
	expectedAddress := h.signer.Address().String()
	if strings.ToLower(address) != strings.ToLower(expectedAddress) {
		h.logger.WithFields(logrus.Fields{
			"expected": expectedAddress,
			"provided": address,
		}).Warn("Address mismatch in eth_sign")
		return h.CreateInvalidParamsResponse(request.ID, "Address mismatch"), nil
	}

	h.logger.WithField("data_length", len(data)).Debug("Processing eth_sign request")

	// 使用 MPC-KMS 进行签名
	signatureHex, err := h.signer.Sign(data)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sign data")
		return h.CreateErrorResponse(request.ID, jsonrpc.CodeInternalError,
			"Failed to sign data", err.Error()), nil
	}

	// 将签名转换为十六进制字符串
	signature := hex.EncodeToString(signatureHex)

	h.logger.Debug("eth_sign completed successfully")
	return h.CreateSuccessResponse(request.ID, signature)
}

// handleEthSignTransaction 处理 eth_signTransaction 方法
func (h *SignHandler) handleEthSignTransaction(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	// 解析交易参数
	txParams, err := signer.ParseTransactionParams(request.Params)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to parse eth_signTransaction params")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Invalid transaction parameters: %v", err)), nil
	}

	h.logger.WithField("from", txParams.From).Debug("Processing eth_signTransaction request")

	// 验证 from 地址匹配（转换为小写比较）
	expectedAddress := h.signer.Address().String()
	if strings.ToLower(txParams.From) != strings.ToLower(expectedAddress) {
		h.logger.WithFields(logrus.Fields{
			"expected": expectedAddress,
			"provided": txParams.From,
		}).Warn("From address mismatch in eth_signTransaction")
		return h.CreateInvalidParamsResponse(request.ID, "From address mismatch"), nil
	}

	// 构建交易
	tx, err := h.builder.BuildTransaction(*txParams)
	if err != nil {
		h.logger.WithError(err).Error("Failed to build transaction")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Failed to build transaction: %v", err)), nil
	}

	// 对交易进行签名
	signedTx, err := h.signer.SignTransaction(tx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sign transaction")
		return h.CreateErrorResponse(request.ID, jsonrpc.CodeInternalError,
			"Failed to sign transaction", err.Error()), nil
	}

	h.logger.Debug("eth_signTransaction completed successfully")
	return h.CreateSuccessResponse(request.ID, signedTx)
}

// handleEthSendTransaction 处理 eth_sendTransaction 方法
func (h *SignHandler) handleEthSendTransaction(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	// 首先执行签名逻辑（与 eth_signTransaction 相同）
	signResponse, err := h.handleEthSignTransaction(ctx, request)
	if err != nil {
		return nil, err
	}

	// 如果签名失败，直接返回错误
	if signResponse.Error != nil {
		return signResponse, nil
	}

	// 注意：实际的 eth_sendTransaction 需要转发到下游服务
	// 这里我们只处理签名部分，转发由专门的 ForwardHandler 处理
	h.logger.Debug("eth_sendTransaction signing completed, forwarding required")

	// 返回签名后的交易数据，由上层路由决定是否转发
	return signResponse, nil
}

// IsSignMethod 检查是否为签名方法
func IsSignMethod(method string) bool {
	switch method {
	case "eth_accounts", "eth_sign", "eth_signTransaction", "eth_sendTransaction":
		return true
	default:
		return false
	}
}
