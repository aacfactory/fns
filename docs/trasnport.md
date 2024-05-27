# Transport

---

Http 服务与客户端。

## HTTP
默认为`fasthttp`。

基本配置：
```yaml
transport:
  port: 8080 
  tls:
    kind: ""        # tls的模式
    options: {}     # tls的相关配置
  options: {}       # 内核的相关配置
  middlewares: {}   # 中间件相关配置
  handlers: {}      # 处理器相关配置
```

如有调整，在`main.go`中设置。

```go
fns.New(
    fns.Transport(tr),  
)
```

### Fasthttp
传输器为`fast.Transport`，其相关配置见`fast.Config`。

### Fasthttp2
同`fast.Transport`，只需开启`fast.Config`中的`http2`配置。


### Standard
传输器为`standard.Transport`，其相关配置见`standard.Config`。

### Http3
详情见[HTTP3](https://github.com/aacfactory/fns-contrib/blob/main/transports/http3/README.md)。

## Middleware

* [Cors](https://github.com/aacfactory/fns/blob/main/docs/cors.md)
* [Compress](https://github.com/aacfactory/fns/blob/main/docs/compress.md)
* [Cache control](https://github.com/aacfactory/fns/blob/main/docs/cache-control.md)
* [Latency](https://github.com/aacfactory/fns/blob/main/docs/latency.md)

## Handler

* [Websocket](https://github.com/aacfactory/fns-contrib/blob/main/transports/handlers/websockets/readme.md)
* [Openapi](https://github.com/aacfactory/fns-contrib/tree/main/transports/handlers/documents)
* [Pprof](https://github.com/aacfactory/fns-contrib/tree/main/transports/handlers/pprof/README.md)

## TLS
安全传输。

### FILE
文件模式，kind为`DEFAULT`，支持国密。

配置：
```yaml
tls:
  kind: "DEFAULT"
  options:
    ca:
      - "ca.cert"
      - "ca2.cert"
    server:
      clientAuth: "" 
      keypair:
        - cert: "cert.pem"
          key: "key.pem"
          password: "when needed"
    client:
      insecureSkipVerify: false
      keypair:
        - cert: "cert.pem"
          key: "key.pem"
          password: "when needed"
```

ClientAuth 枚举值：
* no_client_cert
* request_client_cert
* require_any_client_cert
* verify_client_cert_if_given
* require_and_verify_client_cert

### SSC
自签模式，kind为`SSC`。只需给定一个根证书即可，其它子证书会根据根证书生成，建立根证书的有效时长偏长。

配置:
```yaml
tls:
  kind: "SSC"
  options:
    ca: "ca.cert"
    caKey: "ca.key"
    clientAuth: "require_and_verify_client_cert"
    serverName: "foo"
    insecureSkipVerify: false
```