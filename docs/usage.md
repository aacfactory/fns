# Usage

---

## Create service
1. Create service pkg under `modules`
2. Create `doc.go` under service pkg

Example:
```go
// Package users
// @service users
// @title Users
// @description User service
// @internal false
package users
```

## Create service components when required

Create `components` dir under created service dir,
then create `component` which is implement `services.Component`.  
Components are not required in most cases, unless the service needs to integrate third-party services.

Structure of component:

```text
|-- {service dir}
    |-- components
        |-- component.go
```

component.go

```go
// HelloComponent
// @component
type HelloComponent struct {
}

func (component *HelloComponent) Name() (name string) {
    return "hello"
}

func (component *HelloComponent) Construct(options services.Options) (err error) {
    return
}

func (component *HelloComponent) Shutdown(ctx context.Context) {
}

```

## Create fn

Create go file under created service dir, then create a private go func.
Finally use `fnc codes .` command to generate codes about register fn into service and fn proxy.



Note: the first argument type must be a `context.Context` and the second argument type must be a `struct`, the first
return value type must be a `pointer` or `slice`, the second return value type must be `errors.CodeError`.

Read [definition](https://github.com/aacfactory/fns/blob/main/docs/definition.md) for more.

For example:

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
// @permission obj:act,obj_a:act_a
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

## Register fn and generate fn proxy
Direct access between functions is not recommended. Instead, proxy access is used.

```shell
cd {project path}
fnc codes .
```
Or
```shell
cd {project path}
go generate
```

In the above case, `users.Get` fn proxy will be generated. 

## Get websocket endpoint from context
```go
socket, has := wss.GetWebsocketEndpoint(ctx)
```