# Metric

------

Fns的函数运行指标收集服务。

## 函数指标属性

| Name      | Type     | Description |
|-----------|----------|-------------|
| endpoint  | string   | 服务端口名       |
| fn        | string   | 函数名         |
| latency   | duration | 耗时（毫秒）      |
| succeed   | bool     | 函数处理是否正确    |
| errorCode | int      | 错误代码        |
| errorName | string   | 错误名称        |
| deviceId  | string   | 访问端设备标识     |
| deviceIp  | string   | 访问端设备IP     |


## Reporter
上报器，实现`metrics.Reporter`，可以使用`Prometheus`进行实现。

## 开启服务
在 `modules/dependencies.go` 添加依赖
```go
func dependencies() (v []services.Service) {
    v = []services.Service{
        // add dependencies here
		metrics.New(reporter),
    }
	return
}
```
设置配置
```yaml
metrics:
  reporter: {}  
```

## 自动收集
在函数上增加`@metric`注解。

## 手动收集
```go
metrics.Begin(ctx)
// do something
metrics.End(ctx)
```