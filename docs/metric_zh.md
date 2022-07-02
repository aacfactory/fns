# 指标

------

请求中每个函数处理的指标状态，所以一个请求中会有多个指标结果。

## 模型

| Name      | Type     | Description |
| --------- | -------- |-------------|
| service   | string   | 服务名称        |
| fn        | string   | 函数名         |
| succeed   | bool     | 是否正确处理      |
| errorCode | int      | 错误代码        |
| errorName | string   | 错误名称        |
| latency   | duration | 耗时          |


## 组件
### 报告器 
她是一个接口，可以使用 `Prometheus` 进行实现。 

## 使用
在`modules/dependencies.go`中添加依赖。
```go
func dependencies() (services []service.Service) {
	services = append(
		services,
		stats.Service(&SomeReporter{}),
	)
	return
}
```
设置配置
```yaml
stats:
  reporter: {}
```