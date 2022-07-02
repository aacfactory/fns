# 跟踪

------

跟踪请求，其上报器只会上报来自外部的请求，即集群内的代理请求是不会被上报的，但是会被跟踪到。

## 模型
### 跟踪器
| Name | Type   | Description         |
|------|--------|---------------------|
| id   | string | id of tracer        |
| span | Span   | root span of tracer |

### 跨度

| Name       | Type     | Description                 |
|------------|----------|-----------------------------|
| id         | string   | id of span                  |
| service    | string   | service name                |
| fn         | string   | fn name                     |
| tracerId   | string   | tracer id                   |
| startAt    | time     | start time of fn handing    |
| finishedAt | time     | finished time of fn handled |
| children   | []Span   | sub spans                   |
| tags       | []string | tags                        |



## 组件
### 报告器
它是一个接口，可以使用 `opentracing` 进行实现。

## 使用
在 `modules/dependencies.go` 增加依赖。
```go
func dependencies() (services []service.Service) {
	services = append(
		services,
		tracings.Service(&SomeReporter{}),
	)
	return
}
```
设置配置
```yaml
tracings:
  reporter: {}
```