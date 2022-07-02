# 定义

------

## 服务

服务是一组`fn`。且服务是`fns`的核心组件, 诸如数据库、消息队列都是服务, 服务的注解如下:

| Annotation  | Type   | Required | Description                   |
|-------------|--------|----------|-------------------------------|
| service     | string | true     | 服务的名称                         |
| title       | string | false    | 用于OAS的TAG标题                   |
| description | string | false    | 用于OAS的TAG描述                   |
| internal    | bool   | false    | 如果为真，则来自公网的请求无法直接访问           |



## 函数

函数代表着业务, 第一个参数必须是 `context.Context`，第二个参数必须是 `struct`的值类型, 第一个返回值必须是 `pointer` 或者 `slice`,第二个返回值必须是 `errors.CodeError`, 函数的注解如下:

| Annotation    | Type              | Required | Description                 |
|---------------|-------------------|----------|-----------------------------|
| fn            | string            | true     | 函数名称                        |
| validate      | bool              | false    | 是否开启自动参数校验                  |
| authorization | bool              | false    | 是否开启自动认证校验                  |
| permission    | []roleName        | false    | 是否开启权限校验，值为角色名，即哪些角色可以访问该函数 |
| transactional | enum(sql, dgraph) | false    | 是否自动开启和管理数据库事务              |
| deprecated    | bool              | false    | 是否为弃用                       |
| title         | string            | false    | 用于OAS的请求标题                  |
| description   | string            | false    | 用于OAS的请求描述                  |
| internal      | bool              | false    | 与服务的效果一样                    |

## Context
请求的上下文是一棵树，每个函数的上下文都附着在这棵树上。

```text
|-- request context
    |-- fn context
    |-- fn context
        |-- fn context
```
上下文中由 `current fn log`, `current service components`, `application id`, `service endpoint discovery` 和 `tracer`。

举例:
```go
log := service.GetLog(ctx)

```

## 参数字段注解

| Tag      | Type   | Required | Description                                                        |
|----------|--------|----------|--------------------------------------------------------------------|
| json     | string | false    | json tag                                                           |
| validate | string | false    | 校验器名, 由 [validator](https://github.com/go-playground/validator) 实现 |
| message  | string | false    | 校验失败后的消息提示                                                         |

## Task
任务是协程的代理，由fns的协程池管理。