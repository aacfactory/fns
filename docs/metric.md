# Metric

------

Measurement each fn handling result stats in request. so there are more than one stats of one request.

## Model

| Name      | Type     | Description            |
| --------- | -------- | ---------------------- |
| service   | string   | name of service        |
| fn        | string   | name of fn             |
| succeed   | bool     | handled succeed or not |
| errorCode | int      | code of error          |
| errorName | string   | name or error          |
| latency   | duration | time cost              |


## Component
### Reporter 
It is an interface, so you can use `Prometheus` to implement. 

## Usage
Add service in `modules/dependencies.go`
```go
func dependencies() (services []service.Service) {
	services = append(
		services,
		stats.Service(&SomeReporter{}),
	)
	return
}
```
Setup config
```yaml
stats:
  reporter: {}
```