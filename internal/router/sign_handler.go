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
	builder       *signer.TransactionBuilder
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
		builder:       signer.NewTransactionBuilder(),
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
	txParams, err := signer.ParseTransactionParams(request.Params)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to parse eth_signTransaction params")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Invalid transaction parameters: %v", err)), nil
	}

	h.logger.WithFields(logrus.Fields{
		"from": txParams.From,
		"to":   txParams.To,
	}).Info("Signing transaction")

	expectedAddress := h.signer.Address().String()
	if !strings.EqualFold(txParams.From, expectedAddress) {
		h.logger.WithFields(logrus.Fields{
			"expected": expectedAddress,
			"provided": txParams.From,
		}).Warn("From address mismatch in eth_signTransaction")
		return h.CreateInvalidParamsResponse(request.ID, "From address mismatch"), nil
	}

	tx, err := h.builder.BuildTransaction(*txParams)
	if err != nil {
		h.logger.WithError(err).Error("Failed to build transaction")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Failed to build transaction: %v", err)), nil
	}

	signedTx, err := h.signer.SignTransaction(tx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sign transaction")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to sign transaction", err.Error()), nil
	}

	h.logger.WithFields(logrus.Fields{
		"from": txParams.From,
		"to":   txParams.To,
	}).Info("Transaction signed successfully")
	return h.CreateSuccessResponse(request.ID, signedTx)
}

// handleEthSendTransaction 处理 eth_sendTransaction 方法
func (h *SignHandler) handleEthSendTransaction(ctx context.Context, request *internaljsonrpc.Request) (*internaljsonrpc.Response, error) {
	txParams, err := signer.ParseTransactionParams(request.Params)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to parse eth_sendTransaction params")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Invalid transaction parameters: %v", err)), nil
	}

	h.logger.WithFields(logrus.Fields{
		"from": txParams.From,
		"to":   txParams.To,
	}).Info("Sending transaction")

	expectedAddress := h.signer.Address().String()
	if !strings.EqualFold(txParams.From, expectedAddress) {
		h.logger.WithFields(logrus.Fields{
			"expected": expectedAddress,
			"provided": txParams.From,
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
	if txParams.Nonce == "" {
		txParams.Nonce = fmt.Sprintf("0x%x", nonce)
	}

	gasPrice, err := h.downstreamRPC.Eth().GasPrice()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get gasPrice from downstream")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to get gasPrice", err.Error()), nil
	}

	h.logger.WithField("gasPrice", gasPrice).Debug("Retrieved gasPrice from downstream")

	if txParams.GasPrice == "" || txParams.GasPrice == "0" {
		txParams.GasPrice = new(big.Int).SetUint64(gasPrice).String()
	}

	// 如果客户端没有提供 gas 或 gas 为 0，进行 gas 估算
	if txParams.Gas == "" || txParams.Gas == "0" {
		// 构建 CallMsg 用于 gas 估算
		callMsg := &ethgo.CallMsg{
			From:  h.signer.Address(),
			Value: new(big.Int),
		}

		if txParams.To != "" {
			to := ethgo.HexToAddress(txParams.To)
			callMsg.To = &to
		}

		if txParams.Value != "" {
			value := new(big.Int)
			if _, ok := value.SetString(txParams.Value, 0); ok {
				callMsg.Value = value
			}
		}

		if txParams.Data != "" {
			data, err := hexToBytes(txParams.Data)
			if err == nil {
				callMsg.Data = data
			}
		}

		estimatedGas, err := h.downstreamRPC.Eth().EstimateGas(callMsg)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to estimate gas, using default")
			// 使用默认 gas 限制
			txParams.Gas = "0x5208" // 21000 gas for simple transfer
		} else {
			// 增加 20% 作为安全边界
			estimatedGas = estimatedGas * 120 / 100
			txParams.Gas = fmt.Sprintf("0x%x", estimatedGas)
			h.logger.WithField("estimatedGas", estimatedGas).Debug("Estimated gas for transaction")
		}
	}

	tx, err := h.builder.BuildTransaction(*txParams)
	if err != nil {
		h.logger.WithError(err).Error("Failed to build transaction")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Failed to build transaction: %v", err)), nil
	}

	signedTx, err := h.signer.SignTransaction(tx)
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
		"from": txParams.From,
		"to":   txParams.To,
	}).Info("Transaction sent successfully")
	forwardResponse.ID = request.ID
	forwardResponse.JSONRPC = internaljsonrpc.JSONRPCVersion
	return forwardResponse, nil
}

// hexToBytes 将十六进制字符串转换为字节切片
func hexToBytes(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}

	// 移除 0x 前缀
	if len(s) >= 2 && s[0:2] == "0x" {
		s = s[2:]
	}

	// 如果长度为奇数，在前面补 0
	if len(s)%2 != 0 {
		s = "0" + s
	}

	return hex.DecodeString(s)
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
