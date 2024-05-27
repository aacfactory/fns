# Compress

---

Http 请求与响应的压缩中间件。

## 开启
```go
fns.New(
	fns.Middleware(compress.New()),  
)
```

## 配置
```yaml
transport:
  middlewares:
    compress:
      enable: true
      default: ""  # 默认方法，枚举值：deflate, br,  gzip。
      gzipLevel: 6  
      deflateLevel: 4 
      brotliLevel: 4
```
压缩等级详见`fasthttp`。
