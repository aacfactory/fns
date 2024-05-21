Context

---

上下文在fns中是十分关键的。

可以从上下文里获取很多东西。

# 用户存储
用户存储在集群中是共享的。只能存储可被序列化的值对象。

可以通过泛型的方式快速获取。
```go
v, has, err := context.UserValue[T](ctx, key)
```

# 本地存储
本地存储不能在集群里共享。

可以通过泛型的方式快速获取。
```go
v, has := context.LocalValue[T](ctx, key)
```

# 运行时
```go
rt := runtime.Load(ctx)
```

# 服务组件
```go
// 获取当前服务的组件
component, has := services.GetComponent[T](ctx, componentName)
```

# 函数请求
```go
r := services.LoadRequest(ctx)
```

# 函数请求跟踪链
```go
trace, has := tracings.Load(ctx)
```

# 服务端口
```go
eps := runtime.Endpoints(ctx)
```

# 执行任务
```go
runtime.Execute(ctx, task)
```

# 共享锁
```go
locker, err := runtime.AcquireLocker(ctx, key, ttl)
```