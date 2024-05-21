# FN

---
## 定义
业务领域的函数。 

函数的标准为：
* 必须是私有的。
* 第一个参数必须是`github.com/aacfactory/fns/context.Context`（[关键](https://github.com/aacfactory/fns/blob/main/docs/context.md)）
* 当有入参时，则为第二个参数，且为值对象，不能是指针引用。
* 当有返回值时，则为第一个返回值，且为值对象，不能是指针引用。
* 最后一个返回值必须是`error`
* 函数注释中必须要有`@fn`的标识注解。

## 注解
| 注解             | 值      | 必要 | 含义                                                                               |
|----------------|--------|----|----------------------------------------------------------------------------------|
| @fn            | string | 是  | 函数名，必须是英文的，用于程序中寻址。                                                              |
| @validation    | 无      | 否  | 是否开启参数校验。                                                                        |
| @readonly      | 无      | 否  | 是否为只读，当开启时，HTTP的METHOD为GET，反之为POST。                                              |
| @internal      | 无      | 否  | 是否为内部函数，当开启时，该函数不可被外部端口访问。                                                       |
| @deprecated    | 无      | 否  | 是否为废弃函数，只适用于API文档。                                                               |
| @authorization | 无      | 否  | 是否开启身份校验。                                                                        |
| @permission    | 无      | 否  | 是否开启权限校验。                                                                        |
| @metric        | 无      | 否  | 是否开启指标功能。                                                                        |
| @barrier       | 无      | 否  | 是否开启栅栏，建议只用于`@readonly`函数。                                                       |
| @cache         | 多参     | 否  | 具体见[缓存](https://github.com/aacfactory/fns/blob/main/docs/fn.md#Cache)。           |
| @cache-control | 多参     | 否  | 具体见[缓存控制](https://github.com/aacfactory/fns/blob/main/docs/fn.md#Cache-control)。 |
| @errors        | string | 否  | 错误信息，用于[API文档](https://github.com/aacfactory/fns/blob/main/docs/openapi.md)。     |
| @title         | string | 否  | 标题，用于[API文档](https://github.com/aacfactory/fns/blob/main/docs/openapi.md)。       |
| @description   | string | 否  | 描述，用于[API文档](https://github.com/aacfactory/fns/blob/main/docs/openapi.md)。       |

以上是内置的注解，如需要扩展，请阅读[代码生成器](https://github.com/aacfactory/fns/blob/main/docs/generation.md)。

## 参数校验
通过[validator](https://github.com/go-playground/validator)实现，通过`validate` tag标记属性的校验模式，`validate-message` tag描述错误信息。

`validate`值为`validator`的tag值，并支持以下扩展校验。

| 模式        | 功能                                   |
|-----------|--------------------------------------|
| not_blank | 非空文本。                                |
| not_empty | 非空切片。                                |
| regexp    | 正则表达式，参数为表达式。                        |
| uid       | 是否为[xid](https://github.com/rs/xid)。 |

如虚增加校验扩展，在`init.go`中注入。
```go
import (
	"github.com/fns/services/validators"
)

func init() {
    validators.AddValidateRegister(register) 
}
```

## Cache
函数的缓存操作，其函数必须有入参，且入参必须实现`github.com/aacfactory/fns/caches.KeyParam`。

`KeyParam`为提供缓存的`key`。

| 参数      | 说明                                                           |
|---------|--------------------------------------------------------------|
| get     | 获取缓存                                                         |
| set     | 设置缓存，后跟秒数（如 `@cache set 60`）。                                |
| get-set | 先去缓存，命中直接返回，未命中走函数，函数正确则把返回值加入缓存。后跟秒数，如 `@cache get-set 60`。 |
| remove  | 函数处理后且正确的情况下，删除缓存。                                           |



## Cache-control
Http的`Cache-Control`，当且仅当`@readonly`开启且非`@internal`时有效。

其参数如下：

| 参数               | 说明                                                                 |
|------------------|--------------------------------------------------------------------|
| max-age={sec}    | sec为秒数。设置缓存存储的最大周期，超过这个时间缓存被认为过期 (单位秒)。                            |
| public={bool}    | bool为true或false。 表明响应可以被任何对象（包括：发送请求的客户端，代理服务器，等等）缓存，即使是通常不可缓存的内容。 |
| must-revalidate  | 一旦资源过期（比如已经超过max-age），在成功向原始服务器验证之前，缓存不能用该资源响应后续请求。                |
| proxy-revalidate | 与 must-revalidate 作用相同，但它仅适用于共享缓存（例如代理），并被私有缓存忽略。                  |

## 案例
```go
// add
// @fn add
// @authorization
// @validation
// @cache set
// @title add
// @description >>>
// add user
// <<<
// @errors >>>
// user_not_found
// zh: zh_message
// en: en_message
// <<<
func add(ctx context.Context, param AddParam) (v User, err error) {
    // todo 
	return
}
```
```go
// AddParam
// @title add param
// @description add user param
type AddParam struct {
	// Id
	// @title user id
	// @description user id
	Id string `json:"id" validate:"required" validate-message:"invalid id"`
	// Name
	// @title name
	// @description name
	Name string `json:"name" validate:"required" validate-message:"invalid name"`
	// Age
	// @title age
	// @description age
	Age int `json:"age"`
	// Birthday
	// @title birthday
	// @description birthday
	Birthday times.Time `json:"birthday"`
}
```
```go
type User struct {
	Id       string    `json:"id"`
	Name     string    `json:"name"`
	Age      string    `json:"age"`
	Birthday time.Time `json:"birthday"`
}
```