# Permissions

---

RBAC permission schema. 
Simply, which role can access which function. 
It also supports which role can read or write to which resource.


## Components
### Store
`Postgres` and `MYSQL` are supplied by `fns-contrib`, read [doc](https://github.com/aacfactory/fns-contrib/tree/main/permissions/store) for more.

## API
### Policy
Verify
```go
verifyErr := permissions.Verify(ctx, roles...)
```
User bind roles
```go
bindErr := permissions.UserBindRoles(ctx, userId, roles...)
```
User unbind roles
```go
bindErr := permissions.UserUnbindRoles(ctx, userId, roles...)
```
Get user roles
```go
roles, getErr := permissions.GetUserRoles(ctx, userId)
```
User (current user in context) can read resource
```go
ok, err := CanReadResource(ctx, resource)
```
User (current user in context) can write resource
```go
ok, err := CanWriteResource(ctx, resource)
```
### Model
Get all roles (root role trees)
```go
roles, getErr := permissions.GetRoles(ctx)
```
Get role (current role tree)
```go
role, getErr := permissions.GetRole(ctx, name)
```
Save role (changing of children will not be saved), 
```go
saveErr := permissions.SaveRole(ctx, role)
```
