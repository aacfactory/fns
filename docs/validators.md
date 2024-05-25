# Validators

--- 

函数参数校验。

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