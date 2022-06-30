# Usage

---

## Create service

Use `fnc service create` command to create a service.

## Create service components if required

Create `{component name}` dir under created service dir,
then create `component` which is implement `service.Component`,
last create `components.go` under created service dir and use `@component` to register it into service,
Components are not required in most cases,
unless the service needs to integrate third-party services.

Structure of component:

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

## Create fn

Create go file under created service dir, then create a private go func.
Finally use `fnc code .` command to generate codes about register fn into service and fn proxy.

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
    //@enum F,M,N
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
// @authorization false
// @permission foo,bar
// @internal false
// @title Get user profile
// @description >>>
// Get user profile
// ----------
// errors:
// * user_get_failed
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
