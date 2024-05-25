# Tracing

------

跟踪`fn`请求链。它是一个Http的Middleware。如需使用，请添加[Middleware](https://github.com/aacfactory/fns/blob/main/docs/trasnport.md#Middleware)。

## 添加中间件
在`main.go`中添加，并实现一个`tracings.Reporter`加入到Middleware中。
```go
fns.New(
	fns.Middleware(tracings.Middleware(reporter)),  // reporter是上报器
)
```

## 配置
```yaml
transport:
  middlewares:
    tracings:
      enable: true        # 是否起效。
      batchSize: 4        # 并行数，默认4。
      channelSize: 4096   # channel 大小，默认4096。
      reporter: {}        # 上报器的相关配置
```

## 数据模型
### Tracer
| 属性   | 类型     | 描述  |
|------|--------|-----|
| id   | string | 标识  |
| span | Span   | 根跨度 |

### Span

| 属性       | 类型                | 描述    |
|----------|-------------------|-------|
| id       | string            | 跨度标识  |
| endpoint | string            | 服务端口名 |
| fn       | string            | 函数名   |
| begin    | string            | 开始时间  |
| waited   | time              | 等待时间  |
| end      | time              | 结束时间  |
| tags     | map[string]string | 标签    |
| children | []Span            | 子跨度   |

## 获取上下文中的跟踪器
如需要手动记录，可在上下文中获取。
```go
tracer, has := tracings.Load(ctx)
```
