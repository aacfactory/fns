# 配置

---

RBAC的权限模式。 
简单的说，哪些角色可以访问函数，当然也支持哪些角色可以读写哪些资源。 


## 组件
### 存储器
`Postgres` 和 `MYSQL` 由 `fns-contrib`提供, 阅读 [doc](https://github.com/aacfactory/fns-contrib/tree/main/permissions/store) 获取更多信息。

## 接口
### 策略
验证
```go
verifyErr := permissions.Verify(ctx, roles...)
```
用户绑定角色
```go
bindErr := permissions.UserBindRoles(ctx, userId, roles...)
```
用户解绑角色
```go
bindErr := permissions.UserUnbindRoles(ctx, userId, roles...)
```
获取用户的角色
```go
roles, getErr := permissions.GetUserRoles(ctx, userId)
```
请求上下文中的用户是否可以读资源
```go
ok, err := CanReadResource(ctx, resource)
```
请求上下文中的用户是否可以写资源
```go
ok, err := CanWriteResource(ctx, resource)
```
### 模型
获取所有角色树
```go
roles, getErr := permissions.GetRoles(ctx)
```
获取上下文中用户的指定角色树。
```go
role, getErr := permissions.GetRole(ctx, name)
```
保存角色，不会对其子节点角色进行保存。
```go
saveErr := permissions.SaveRole(ctx, role)
```
