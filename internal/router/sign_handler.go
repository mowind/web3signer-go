package router

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/mowind/web3signer-go/internal/downstream"
	internaljsonrpc "github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/mowind/web3signer-go/internal/signer"
	"github.com/sirupsen/logrus"
	"github.com/umbracle/ethgo"
	ethgojsonrpc "github.com/umbracle/ethgo/jsonrpc"
)

// SignHandler 处理签名相关的 JSON-RPC 方法
type SignHandler struct {
	*BaseHandler
	signer        *signer.MPCKMSSigner
	client        downstream.ClientInterface
	downstreamRPC *ethgojsonrpc.Client
}

// NewSignHandler 创建签名处理器
func NewSignHandler(mpcSigner *signer.MPCKMSSigner, client downstream.ClientInterface, downstreamEndpoint string, logger *logrus.Logger) (*SignHandler, error) {
	rpcClient, err := ethgojsonrpc.NewClient(downstreamEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create downstream RPC client: %v", err)
	}

	return &SignHandler{
		BaseHandler:   NewBaseHandler("sign", logger),
		signer:        mpcSigner,
		client:        client,
		downstreamRPC: rpcClient,
	}, nil
}

// handleEthAccounts 处理 eth_accounts 方法
func (h *SignHandler) handleEthAccounts(ctx context.Context, request *internaljsonrpc.Request) (*internaljsonrpc.Response, error) {
	kmsAddress := h.signer.Address().String()

	h.logger.WithField("address", kmsAddress).Debug("Returning KMS managed address for eth_accounts")

	return h.CreateSuccessResponse(request.ID, []string{kmsAddress})
}

// Method 返回处理器支持的方法名
func (h *SignHandler) Method() string {
	return "sign_handler"
}

// Handle 处理 JSON-RPC 请求
func (h *SignHandler) Handle(ctx context.Context, request *internaljsonrpc.Request) (*internaljsonrpc.Response, error) {
	h.LogRequest(request)

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
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeMethodNotFound,
			"Method not supported by sign handler", nil), nil
	}
}

// handleEthSign 处理 eth_sign 方法
func (h *SignHandler) handleEthSign(ctx context.Context, request *internaljsonrpc.Request) (*internaljsonrpc.Response, error) {
	address, data, err := signer.ParseSignParams(request.Params)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to parse eth_sign params")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	expectedAddress := h.signer.Address().String()
	if !strings.EqualFold(address, expectedAddress) {
		h.logger.WithFields(logrus.Fields{
			"expected": expectedAddress,
			"provided": address,
		}).Warn("Address mismatch in eth_sign")
		return h.CreateInvalidParamsResponse(request.ID, "Address mismatch"), nil
	}

	h.logger.WithFields(logrus.Fields{
		"data_length": len(data),
	}).Info("Signing data")

	signatureHex, err := h.signer.Sign(data)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sign data")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to sign data", err.Error()), nil
	}

	signature := hex.EncodeToString(signatureHex)

	h.logger.WithFields(logrus.Fields{
		"address": h.signer.Address().String(),
	}).Info("Data signed successfully")
	return h.CreateSuccessResponse(request.ID, signature)
}

// handleEthSignTransaction 处理 eth_signTransaction 方法
func (h *SignHandler) handleEthSignTransaction(ctx context.Context, request *internaljsonrpc.Request) (*internaljsonrpc.Response, error) {
	tx, err := signer.ParseJSONRPCTransaction(request.Params)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to parse eth_signTransaction params")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Invalid transaction parameters: %v", err)), nil
	}

	h.logger.WithFields(logrus.Fields{
		"from": tx.From.String(),
		"to":   tx.To,
	}).Info("Signing transaction")

	expectedAddress := h.signer.Address().String()
	if tx.From.String() != "" && !strings.EqualFold(tx.From.String(), expectedAddress) {
		h.logger.WithFields(logrus.Fields{
			"expected": expectedAddress,
			"provided": tx.From.String(),
		}).Warn("From address mismatch in eth_signTransaction")
		return h.CreateInvalidParamsResponse(request.ID, "From address mismatch"), nil
	}

	signedTx, err := h.signer.SignTransaction(&tx.Transaction)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sign transaction")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to sign transaction", err.Error()), nil
	}

	h.logger.WithFields(logrus.Fields{
		"from": tx.From.String(),
		"to":   tx.To,
	}).Info("Transaction signed successfully")
	return h.CreateSuccessResponse(request.ID, signedTx)
}

// handleEthSendTransaction 处理 eth_sendTransaction 方法
func (h *SignHandler) handleEthSendTransaction(ctx context.Context, request *internaljsonrpc.Request) (*internaljsonrpc.Response, error) {
	tx, err := signer.ParseJSONRPCTransaction(request.Params)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to parse eth_sendTransaction params")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Invalid transaction parameters: %v", err)), nil
	}

	h.logger.WithFields(logrus.Fields{
		"from": tx.From.String(),
		"to":   tx.To,
	}).Info("Sending transaction")

	expectedAddress := h.signer.Address().String()
	if tx.From.String() != "" && !strings.EqualFold(tx.From.String(), expectedAddress) {
		h.logger.WithFields(logrus.Fields{
			"expected": expectedAddress,
			"provided": tx.From.String(),
		}).Warn("From address mismatch in eth_sendTransaction")
		return h.CreateInvalidParamsResponse(request.ID, "From address mismatch"), nil
	}

	// 获取账户 nonce
	nonce, err := h.downstreamRPC.Eth().GetNonce(h.signer.Address(), ethgo.Latest)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get nonce from downstream")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to get nonce", err.Error()), nil
	}

	h.logger.WithField("nonce", nonce).Debug("Retrieved nonce from downstream")

	// 如果客户端没有提供 nonce，使用获取到的 nonce
	if tx.Nonce == 0 {
		tx.Nonce = nonce
	}

	gasPrice, err := h.downstreamRPC.Eth().GasPrice()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get gasPrice from downstream")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to get gasPrice", err.Error()), nil
	}

	h.logger.WithField("gasPrice", gasPrice).Debug("Retrieved gasPrice from downstream")

	// 根据交易类型填充 gasPrice 或 maxFeePerGas/maxPriorityFeePerGas
	switch tx.Type {
	case ethgo.TransactionDynamicFee:
		// EIP-1559: 如果未提供，使用 gasPrice 作为 maxFeePerGas 和 maxPriorityFeePerGas
		if tx.MaxFeePerGas == nil || tx.MaxFeePerGas.Uint64() == 0 {
			tx.MaxFeePerGas = new(big.Int).SetUint64(gasPrice)
		}
		if tx.MaxPriorityFeePerGas == nil || tx.MaxPriorityFeePerGas.Uint64() == 0 {
			tx.MaxPriorityFeePerGas = new(big.Int).SetUint64(gasPrice)
		}
	case ethgo.TransactionLegacy, ethgo.TransactionAccessList:
		// Legacy 或 EIP-2930: 如果未提供 gasPrice，使用获取到的值
		if tx.GasPrice == 0 {
			tx.GasPrice = gasPrice
		}
	}

	// 如果客户端没有提供 gas 或 gas 为 0，进行 gas 估算
	if tx.Gas == 0 {
		// 构建 CallMsg 用于 gas 估算
		callMsg := &ethgo.CallMsg{
			From:  h.signer.Address(),
			Value: new(big.Int),
		}

		if tx.To != nil {
			callMsg.To = tx.To
		}

		if tx.Value != nil {
			callMsg.Value = tx.Value
		}

		if len(tx.Input) > 0 {
			callMsg.Data = tx.Input
		}

		estimatedGas, err := h.downstreamRPC.Eth().EstimateGas(callMsg)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to estimate gas, using default")
			// 使用默认 gas 限制
			tx.Gas = 21000 // 21000 gas for simple transfer
		} else {
			// 增加 20% 作为安全边界
			estimatedGas = estimatedGas * 120 / 100
			tx.Gas = estimatedGas
			h.logger.WithField("estimatedGas", estimatedGas).Debug("Estimated gas for transaction")
		}
	}

	signedTx, err := h.signer.SignTransaction(&tx.Transaction)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sign transaction")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to sign transaction", err.Error()), nil
	}

	rlpBytes := make([]byte, 0)
	rlpBytes, err = signedTx.MarshalRLPTo(rlpBytes)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal transaction to RLP")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to marshal transaction", err.Error()), nil
	}

	rawTxHex := "0x" + hex.EncodeToString(rlpBytes)

	paramsBytes, err := json.Marshal([]interface{}{rawTxHex})
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal eth_sendRawTransaction params")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to create forward request", err.Error()), nil
	}

	forwardRequest := &internaljsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "eth_sendRawTransaction",
		Params:  paramsBytes,
		ID:      request.ID,
	}

	forwardResponse, err := h.client.ForwardRequest(ctx, forwardRequest)
	if err != nil {
		h.logger.WithError(err).Error("Failed to forward eth_sendRawTransaction to downstream")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to forward transaction", err.Error()), nil
	}

	if forwardResponse.Error != nil {
		h.logger.WithField("error", forwardResponse.Error.Message).Error("Downstream returned error")
		return h.CreateErrorResponse(request.ID, forwardResponse.Error.Code,
			forwardResponse.Error.Message, forwardResponse.Error.Data), nil
	}

	h.logger.WithFields(logrus.Fields{
		"from": tx.From.String(),
		"to":   tx.To,
	}).Info("Transaction sent successfully")
	forwardResponse.ID = request.ID
	forwardResponse.JSONRPC = internaljsonrpc.JSONRPCVersion
	return forwardResponse, nil
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
