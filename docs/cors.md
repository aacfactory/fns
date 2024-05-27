# Cors

---

跨域中间件。

## 开启
```go
fns.New(
	fns.Middleware(cors.New()),  
)
```

## 配置
```yaml
transport:
  middlewares:
    cors:
      maxAge: 6000
      allowPrivateNetwork: true
      allowCredentials: true
      allowedOrigins:
        - ""
      allowedHeaders:
        - ""
      exposedHeaders:
        - ""
```

