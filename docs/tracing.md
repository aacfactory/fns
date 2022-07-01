# Tracing

------

Trace the `fn` used in the request. the tracer will be reported when origin request was finished.

## Model
### Tracer
| Name | Type   | Description         |
|------|--------|---------------------|
| id   | string | id of tracer        |
| span | Span   | root span of tracer |

### Span

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



## Component
### Reporter
It is an interface, so you can use `opentracing` to implement.

## Usage
Add service in `modules/dep.go`
```go
func dependencies() (services []service.Service) {
	services = append(
		services,
		tracings.Service(&SomeReporter{}),
	)
	return
}
```
Setup config
```yaml
tracings:
  reporter: {}
```