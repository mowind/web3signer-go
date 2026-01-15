# web3signer-go 需求

web3signer-go 灵感来源于 [web3signer](https://github.com/Consensys/web3signer)，与 web3signer 不同，web3signer-go 只支持通过 MPC-KMS ，通过 MPC-KMS 进行签名。

web3signer-go 对外提供 JSON-RPC 服务，支持 [ETH JSON-RPC](https://ethereum.org/zh/developers/docs/apis/json-rpc/) 接口。web3signer-go 实现以下接口，其他接口透穿给指定的 `downstream` 服务：
- eth_accounts
- eth_sign
- eth_signTransaction
- eth_sendTransaction

`eth_accounts` MPC-KMS没有提供接口获取access key对应的地址列表，暂时空实现。

`eth_sign` `eth_signTransaction` `eth_sendTransaction` 通过调用MPC-KMS的签名接口对数据进行签名。MPC-KMS文档在这里 [KMS RPC](./gw-http-api.md)

## Usage

web3signer-go 通过以下命令启动

```shell
web3signer --http-host localhost \
           --http-port 9000 \
           --kms-endpoint <MPC-KMS endpoint> \
           --kms-access-key-id <access key id> \
           --kms-secret-key <secret key> \
           --downstream-http-host <downstream http host> \
           --downstream-http-port <downstream http port> \
           --downstream-http-path <downstream http path>
```

## 技术栈

- [gin](https://github.com/gin-gonic/gin)
- [viper](https://github.com/spf13/viper)
- [cobra](https://github.com/spf13/cobra)
- [logrus](https://github.com/sirupsen/logrus)