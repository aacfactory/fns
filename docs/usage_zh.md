# 使用

---

## 创建访问
1. 在`modules`下创建服务包
2. 在服务包下创建 `doc.go`

举例：
```go
// Package users
// @service users
// @title Users
// @description User service
// @internal false
package users
```

## 创建服务组件（如果需要的话）

在服务包下创建 `{component name}` 包。
然后在`component`包下创建实现了 `service.Component`的组件，
最后创建 `components.go` 并打上 `@component`来把它注册到服务中。
组件在大多数情况下是不需要的，除非服务中集成了第三方服务。


组件解构:

```text
|-- {service dir}
    |-- {component name}
        |-- component.go
    |-- components.go
```

component.go

```go
type Foo struct {
}

func (c *Foo) Name() (name string) {
    name = "component_name"
    return
}

func (c *Foo) Build(options service.ComponentOptions) (err error) {
    // build with config
    return
}

func (c *Foo) Close() {
    return
}
```

components.go, value of `@component` is keypair, key is component_name, value is component position.

```go
// fooLoader
// @component {component_name}:{package}.Foo
func fooLoader() service.Component {
return &foo.Foo{}
}
```

## 创建函数

在服务包下创建一个私有函数。

阅读 [definition](https://github.com/aacfactory/fns/blob/main/docs/definition_zh.md) 获取更多信息。

举例:

```go
// GetArgument
// @title Get user argument
// @description Get user argument
type GetArgument struct {
    // Id
    // @title User Id
    // @description User Id
    Id int64 `json:"id" validate:"required" message:"id is invalid"`
}

// User
// @title User
// @description User
type User struct {
    // Id
    // @title User Id
    // @description User Id
    Id string `json:"id"`
    // Mobile
    // @title Mobile
    // @description Mobile
    Mobile string `json:"mobile"`
    // Name
    // @title Name
    // @description Name
    Name string `json:"name"`
    // Gender
    // @title Gender 
    // @enum F,M,N
    // @description Gender
    Gender string `json:"gender"`
    // Age
    // @title Age
    // @description Age
    Age int `json:"age"`
    // Avatar
    // @title Avatar
    // @description Avatar address
    Avatar string `json:"avatar"`
    // Active
    // @title Active
    // @description Active
    Active bool `json:"active"`
}


// get
// @fn get
// @validate true
// @authorization true
// @permission foo,bar
// @internal false
// @title Get user profile
// @description >>>
// Get user profile
// ----------
// errors:
// | Name                     | Code    | Description                   |
// |--------------------------|---------|-------------------------------|
// | users_get_failed         | 500     | get user failed               |
// <<<
func get(ctx context.Context, argument GetArgument) (v *User, err errors.CodeError) {
    v = &User{
        Id:     fmt.Sprintf("%v", argument.Id),
        Mobile: "000",
        Name:   "foo",
        Gender: "F",
        Age:    10,
        Avatar: "https://foo.com/u/1/default.jpg",
        Active: true,
    }
    return
}

```

## 注册函数和创建函数代理
直接访问函数是不建议的，那会失去很多特性，所以请使用函数代理的访问访问。

```shell
cd {project path}
fnc codes .
```
Or
```shell
cd {project path}
go generate
```

在上述例子中， `users.Get` 函数代理会被生成。
