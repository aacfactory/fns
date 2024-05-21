# Logs

---

Fns的异步日志系统。支持`text`和`json`两种输出格式。支持多个记载者。

## 配置
```yaml
log:
  level: ""               # debug info warn error（可选）
  formatter: ""           # text text_colorful json（可选）
  console: ""             # stdout stderr stdout_stderr（可选）
  disableConsole: false   # 是否禁止控制台输出（可选）
  consumes: 4             # 事件消费者（可选）
  buffer: 1024            # 队列大小（可选）
  sendTimeout: "10s"      # 发送超时（可选），在队列满的情况下，超时多少后则抛弃。
  shutdownTimeout: "3s"   # 关闭超时（可选）
  writers:                # 记载者（可选），支持多个。一般用于发送到Kafka或者指定文件。
    - name: ""            # 记载者的名称（必选）
      options: {}         # 记载者的相关选项配置
```

## 获取
```go
log := logs.Load(ctx)
```
