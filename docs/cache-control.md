# Cache-Control

---

`Cache-Control` http middleware.

## 开启
```go
fns.New(
	fns.Middleware(cachecontrol.New()),  
)
```

## 配置
```yaml
transport:
  middlewares:
    cachecontrol:
      enable: true        # 是否起效。
      maxAge: 60          # 默认最大缓存秒数。
```

## 使用

使用时打上`@cache-control`，当且仅当`@readonly`开启且非`@internal`时有效。

其参数如下：

| 参数               | 说明                                                                 |
|------------------|--------------------------------------------------------------------|
| max-age={sec}    | sec为秒数。设置缓存存储的最大周期，超过这个时间缓存被认为过期 (单位秒)。                            |
| public={bool}    | bool为true或false。 表明响应可以被任何对象（包括：发送请求的客户端，代理服务器，等等）缓存，即使是通常不可缓存的内容。 |
| must-revalidate  | 一旦资源过期（比如已经超过max-age），在成功向原始服务器验证之前，缓存不能用该资源响应后续请求。                |
| proxy-revalidate | 与 must-revalidate 作用相同，但它仅适用于共享缓存（例如代理），并被私有缓存忽略。                  |
