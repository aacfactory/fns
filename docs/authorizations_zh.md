# 认证

---

Http 认证
HTTP授权请求标头可用于提供凭据，用于向服务器验证用户代理，从而允许访问受保护的资源。
阅读 [Authorization](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Authorization) 获取更多信息。

## 组件
### 编码器
`DEFAULT` 编码器是 `fns` 内置的, 配置如下 
```yaml
authorizations:
  encoding:
    expireMinutes: 1440
```
`JWT` 编码器是 `fns-contrib` 提供的, 阅读 [doc](https://github.com/aacfactory/fns-contrib/tree/main/authorizations/encoding/jwt) 获取更多信息。
### 存储器
`Discard` 不会存储用户令牌， 所以 `revoke` 接口是不会有响应的。

`Redis`, `Postgres` 和 `MYSQL` 是由 `fns-contrib` 提供，阅读 [doc](https://github.com/aacfactory/fns-contrib/tree/main/authorizations/store) 获取更多信息。

## 接口
编码是通过用户标识生成令牌。
```go
token, encodingErr := authorizations.Encode(ctx, "userId", userAttributes)
```
验证当前请求上下文，如 `@authorization`值为真，则会自动验证。
```go
verifyErr := authorizations.Verify(ctx)
```
吊销令牌。
```go
revokeErr := authorizations.Revoke(ctx, "tokenId")
```
吊销用户的所有令牌。
```go
revokeErr := authorizations.RevokeUserTokens(ctx, "userId")
```