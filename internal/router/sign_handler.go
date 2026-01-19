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
	if strings.ToLower(address) != strings.ToLower(expectedAddress) {
		h.logger.WithFields(logrus.Fields{
			"expected": expectedAddress,
			"provided": address,
		}).Warn("Address mismatch in eth_sign")
		return h.CreateInvalidParamsResponse(request.ID, "Address mismatch"), nil
	}

	h.logger.WithField("data_length", len(data)).Debug("Processing eth_sign request")

	signatureHex, err := h.signer.Sign(data)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sign data")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to sign data", err.Error()), nil
	}

	signature := hex.EncodeToString(signatureHex)

	h.logger.Debug("eth_sign completed successfully")
	return h.CreateSuccessResponse(request.ID, signature)
}

// handleEthSignTransaction 处理 eth_signTransaction 方法
func (h *SignHandler) handleEthSignTransaction(ctx context.Context, request *internaljsonrpc.Request) (*internaljsonrpc.Response, error) {
	txParams, err := signer.ParseTransactionParams(request.Params)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to parse eth_signTransaction params")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Invalid transaction parameters: %v", err)), nil
	}

	h.logger.WithField("from", txParams.From).Debug("Processing eth_signTransaction request")

	expectedAddress := h.signer.Address().String()
	if strings.ToLower(txParams.From) != strings.ToLower(expectedAddress) {
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

	h.logger.Debug("eth_signTransaction completed successfully")
	return h.CreateSuccessResponse(request.ID, signedTx)
}

// handleEthSendTransaction 处理 eth_sendTransaction 方法
func (h *SignHandler) handleEthSendTransaction(ctx context.Context, request *internaljsonrpc.Request) (*internaljsonrpc.Response, error) {
	txParams, err := signer.ParseTransactionParams(request.Params)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to parse eth_sendTransaction params")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Invalid transaction parameters: %v", err)), nil
	}

	h.logger.WithField("from", txParams.From).Debug("Processing eth_sendTransaction request")

	expectedAddress := h.signer.Address().String()
	if strings.ToLower(txParams.From) != strings.ToLower(expectedAddress) {
		h.logger.WithFields(logrus.Fields{
			"expected": expectedAddress,
			"provided": txParams.From,
		}).Warn("From address mismatch in eth_sendTransaction")
		return h.CreateInvalidParamsResponse(request.ID, "From address mismatch"), nil
	}

	chainID, err := h.downstreamRPC.Eth().ChainID()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get chainId from downstream")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to get chainId", err.Error()), nil
	}

	h.logger.WithField("chainId", chainID).Debug("Retrieved chainId from downstream")

	gasPrice, err := h.downstreamRPC.Eth().GasPrice()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get gasPrice from downstream")
		return h.CreateErrorResponse(request.ID, internaljsonrpc.CodeInternalError,
			"Failed to get gasPrice", err.Error()), nil
	}

	h.logger.WithField("gasPrice", gasPrice).Debug("Retrieved gasPrice from downstream")

	txParams.ChainID = chainID.String()
	if txParams.GasPrice == "" || txParams.GasPrice == "0" {
		txParams.GasPrice = new(big.Int).SetUint64(gasPrice).String()
	}

	tx, err := h.builder.BuildTransaction(*txParams)
	if err != nil {
		h.logger.WithError(err).Error("Failed to build transaction")
		return h.CreateInvalidParamsResponse(request.ID, fmt.Sprintf("Failed to build transaction: %v", err)), nil
	}

	tx.ChainID = chainID

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

	h.logger.Debug("eth_sendTransaction completed successfully")
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
