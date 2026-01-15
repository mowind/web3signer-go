# web3signer-go

web3signer-go is inspired by [web3signer]("https://github.com/Consensys/web3signer"), but only support MPC-KMS signing.

## Usage

``` shell
web3signer --http-host localhost \
           --http-port 9000 \
           --kms-endpoint <MPC-KMS endpoint> \
           --kms-access-key-id <access key id> \
           --kms-secret-key <secret key> \
           --downstream-http-host <downstream http host> \
           --downstream-http-port <downstream http port> \
           --downstream-http-path <downstream http path>
```
