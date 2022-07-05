# Definition

------

## Service

Service is a group of `fn`. and it is core unit, everything are service, such as database and message queues, annotations of service are:

| Annotation  | Type   | Required | Description                                                                        |
|-------------|--------|----------|------------------------------------------------------------------------------------|
| service     | string | true     | name of service                                                                    |
| title       | string | false    | used for tag name of oas                                                           |
| description | string | false    | used for tag description of oas                                                    |
| internal    | bool   | false    | it value is true, then fn in this service cannot be accessed by the public network |



## Fn

Fn is a business function, the first argument type must be a `context.Context` and the second argument type must be a `struct`, the first return value type must be a `pointer` or `slice`, the second return value type must be `errors.CodeError`, annotations of fn are:

| Annotation    | Type                               | Required | Description                                                                        |
|---------------|------------------------------------|----------|------------------------------------------------------------------------------------|
| fn            | string                             | true     | name of fn                                                                         |
| validate      | bool                               | false    | whether parameter verification is required                                         |
| authorization | bool                               | false    | whether authorization verification is required                                     |
| permission    | []roleName                         | false    | whether permission verification is required and what roles can access the fn       |
| transactional | enum(sql, postgres, mysql, dgraph) | false    | whether to start database transaction                                              |
| deprecated    | bool                               | false    | deprecated                                                                         |
| title         | string                             | false    | used for request title of oas                                                      |
| description   | string                             | false    | used for request description of oas                                                |
| internal      | bool                               | false    | it value is true, then fn in this service cannot be accessed by the public network |

## Context
The context of request is a tree, the context of each function is on the request context tree.

```text
|-- request context
    |-- fn context
    |-- fn context
        |-- fn context
```
There are `current fn log`, `current service components`, `application id`, `service endpoint discovery` and `tracer` in the context.

For example:
```go
log := service.GetLog(ctx)

```

## Argument field tag

| Tag      | Type   | Required | Description                                                                          |
|----------|--------|----------|--------------------------------------------------------------------------------------|
| json     | string | false    | json tag                                                                             |
| validate | string | false    | validator name, implement by [validator](https://github.com/go-playground/validator) |
| message  | string | false    | validate failed message                                                              |

## Task
The task is a goroutine proxy, and it is managed by fns goroutine pool.