# Permissions

---

Fns的权限服务，提供校验`authorization.Account`是否可以访问`fn`。

具体实现可见[RBAC](https://github.com/aacfactory/fns-contrib/tree/main/permissions/rbac)。

## 配置
```yaml
services:
  permissions:
    ...
```

## 添加依赖
在`modules/services.go`中的`dependencies`函数中添加。
```go
func dependencies() (v []services.Service) {
	v = []services.Service{
		// add dependencies here
		permissions.New(enforcer),
	}
	return
}
```

## 权限校验
校验上下文中`authorization.Account`对当前`fn`的权限。
```go
err := permissions.EnforceContext(ctx)
```

校验指定`authorization.Account`对指定`fn`的权限。
```go
param := permissions.EnforceParam{
    Account: authorization.Id("xx"),
	Endpoint: "",  // name of service 
	Fn: ""         // name of fn
}
ok, err := permissions.Enforce(ctx, param)
```