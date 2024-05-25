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
传输器为`fast.Transport`。

配置：
```yaml

```

### Fasthttp2
传输器为`fast.Transport`。只需开启即可。注意：它是不稳定的。

配置：
```yaml

```

### Standard
传输器为`standard.Transport`。

配置：
```yaml

```

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
文件模式，kind为`DEFAULT`。

配置：
```yaml
tls:
  kind: "DEFAULT"
  options: 
    
```

### SSC