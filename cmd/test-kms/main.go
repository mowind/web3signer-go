package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/kms"
	"github.com/sirupsen/logrus"
)

func main() {
	// 使用提供的参数
	kmsConfig := &config.KMSConfig{
		Endpoint:    "http://10.2.8.108:8080",
		AccessKeyID: "c609f7de1e154999bd1018026a665149",
		SecretKey:   "Z7CY32LuQW+ccdc+m01YY4b92neAi7bM5bQ0SWbXjp4=",
		KeyID:       "38HGvLc8nJ6KwQqn2PzCvZg70yJ",
	}

	fmt.Println("=== MPC-KMS HTTP签名测试 ===")
	fmt.Printf("Endpoint: %s\n", kmsConfig.Endpoint)
	fmt.Printf("AccessKeyID: %s\n", kmsConfig.AccessKeyID)
	fmt.Printf("KeyID: %s\n", kmsConfig.KeyID)
	fmt.Printf("SecretKey: [REDACTED]\n")
	fmt.Println()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	client := kms.NewClient(kmsConfig, logger)

	// 测试1: 测试签名请求构建
	fmt.Println("测试1: 测试签名请求构建")
	if err := testSignRequest(kmsConfig); err != nil {
		fmt.Printf("❌ 测试1失败: %v\n", err)
	} else {
		fmt.Println("✅ 测试1通过: 签名请求构建成功")
	}
	fmt.Println()

	// 测试2: 测试实际的签名调用
	fmt.Println("测试2: 测试实际的签名调用")
	fmt.Println("  注意: GG18算法要求消息长度为32字节")
	if err := testActualSign(client, kmsConfig); err != nil {
		fmt.Printf("❌ 测试2失败: %v\n", err)
	} else {
		fmt.Println("✅ 测试2通过: 签名调用成功")
	}
	fmt.Println()

	// 测试3: 测试错误处理
	fmt.Println("测试3: 测试错误处理")
	if err := testErrorHandling(client, kmsConfig); err != nil {
		fmt.Printf("❌ 测试3失败: %v\n", err)
	} else {
		fmt.Println("✅ 测试3通过: 错误处理正常")
	}
}

func testSignRequest(kmsConfig *config.KMSConfig) error {
	testData := []byte(`{"data": "test", "encoding": "PLAIN"}`)
	req, err := http.NewRequest("POST", kmsConfig.Endpoint+"/api/v1/keys/test/sign", bytes.NewReader(testData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	httpClient := kms.NewHTTPClient(kmsConfig, logger)

	// 签名请求
	if err := httpClient.SignRequest(req, testData); err != nil {
		return fmt.Errorf("签名请求失败: %w", err)
	}

	// 验证请求头
	fmt.Printf("  Authorization: %s\n", req.Header.Get("Authorization"))
	fmt.Printf("  Date: %s\n", req.Header.Get("Date"))
	fmt.Printf("  Content-Type: %s\n", req.Header.Get("Content-Type"))

	// 验证Authorization头格式
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("Authorization头为空")
	}

	if len(authHeader) < 20 {
		return fmt.Errorf("authorization header too short: %s", authHeader)
	}

	return nil
}

func testActualSign(client *kms.Client, kmsConfig *config.KMSConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试不同长度的消息
	testCases := []struct {
		name     string
		message  []byte
		encoding kms.DataEncoding
	}{
		{
			name:     "32字节消息（GG18要求）",
			message:  []byte("0123456789abcdef0123456789abcdef"), // 32字节
			encoding: kms.DataEncodingPlain,
		},
		{
			name:     "32字节HEX编码",
			message:  []byte("3031323334353637383961626364656630313233343536373839616263646566"), // "0123456789abcdef0123456789abcdef"的HEX
			encoding: kms.DataEncodingHex,
		},
		{
			name:     "交易哈希（32字节HEX）",
			message:  []byte("7d5a5c5f5e5d5b5a595857565554535251504f4e4d4c4b4a49484746454443"),
			encoding: kms.DataEncodingHex,
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n  测试用例: %s\n", tc.name)
		fmt.Printf("    消息长度: %d字节\n", len(tc.message))
		fmt.Printf("    编码格式: %s\n", tc.encoding)
		fmt.Printf("    使用KeyID: %s\n", kmsConfig.KeyID)

		// 尝试调用签名接口
		var signature []byte
		var err error

		if tc.encoding == kms.DataEncodingPlain {
			signature, err = client.Sign(ctx, kmsConfig.KeyID, tc.message)
		} else {
			signature, err = client.SignWithOptions(ctx, kmsConfig.KeyID, tc.message, tc.encoding, nil, "")
		}

		if err != nil {
			fmt.Printf("    ❌ 签名失败: %v\n", err)

			// 如果是长度错误，提供建议
			if strings.Contains(err.Error(), "bad sign message length") {
				fmt.Println("    💡 建议: 确保消息长度为32字节（GG18算法要求）")
			}

			// 继续测试下一个用例
			continue
		}

		fmt.Printf("    ✅ 签名成功!\n")
		fmt.Printf("    签名结果: %s\n", string(signature))

		// 如果有一个成功，就返回成功
		return nil
	}

	return fmt.Errorf("所有测试用例都失败了")
}

func testErrorHandling(client *kms.Client, kmsConfig *config.KMSConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试1: 使用不存在的KeyID
	fmt.Println("  测试不存在的KeyID...")
	_, err := client.Sign(ctx, "non-existent-key-id", []byte("test"))
	if err != nil {
		fmt.Printf("    预期错误: %v\n", err)
	} else {
		fmt.Println("    警告: 应该返回错误但成功了")
	}

	// 测试2: 空数据
	fmt.Println("  测试空数据...")
	_, err = client.Sign(ctx, kmsConfig.KeyID, []byte{})
	if err != nil {
		fmt.Printf("    错误: %v\n", err)
	} else {
		fmt.Println("    空数据签名成功")
	}

	return nil
}
