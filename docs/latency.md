# Latency

---

记录请求耗时，并以Header名为`X-Fns-Handle-Latency`的值返回。

## 开启
```go
fns.New(
	fns.Middleware(latency.New()),  
)
```

## 配置
```yaml
transport:
  middlewares:
    latency:
      enable: true
```
