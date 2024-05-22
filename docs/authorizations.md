# Authorizations

---
Fns的身份验证服务。

验证 HTTP 头 [`Authorization`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Authorization) 的值。

当前提供`hmac`和[`jwt`](https://github.com/aacfactory/fns-contrib/tree/main/authorizations/encoding/jwt)两种实现。

令牌默认存储在共享器中，如由需要，可以代替。

## 配置
Hmac
```yaml
services:
  authorizations:
    expireTTL: "24h"
    autoRefresh: true
    autoRefreshWindow: "12h"
    encoder: 
      key: "some sk"
```

## 校验
在函数上打上`@authorization`注解即可。

手动校验：
```go
// 校验指定令牌，当成功后，身份不会自动注入上下文中。
validated, validErr := authorizations.Validate(ctx, token)
// 校验当前上下文中的令牌，当成功后，身份会自动注入上下文中。
validated, validErr := authorizations.ValidateContext(ctx)
```

## 获取身份
```go
authorization, has, err := authorizations.Load(ctx)
```

## 创建令牌
```go
token, err := authorizations.Create(ctx, param)
```
成功创建后，身份会自动注入上下文中。

