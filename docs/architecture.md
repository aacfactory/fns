# architecture

---

# Cluster
![Cluster](https://github.com/aacfactory/fns/blob/main/docs/cluster.png "Cluster")

注册表形式的[集群](https://github.com/aacfactory/fns/blob/main/docs/cluster.md)。
# Service
![Service](https://github.com/aacfactory/fns/blob/main/docs/service.png "Service")

服务为函数包。一个服务代表一个业务领域，[函数](https://github.com/aacfactory/fns/blob/main/docs/fn.md)是业务功能。

在代码中，服务通过标识自动生成。

在`doc.go`中进行标识。
```go
// Package foo
// @service foo
// @title Foo
// @description Foo service
package foo
```
| 注解           | 值      | 必要 | 含义                  |
|--------------|--------|----|---------------------|
| @service     | string | 是  | 服务名，必须是英文的，用于程序中寻址。 |
| @title       | string | 否  | 标题，用于API文档。         |
| @description | string | 否  | 描述，用于API文档。         |

## Listenable
监听服务，在`Service`上增加了`Listen`函数。 

在应用启动后，开启服务监听。一般适用于消息队列服务，监听事件除非函数。

## Component
服务组件，一般用于向服务注入第三方SDK。

创建组件：
1. 一般会在服务包下创建一个子包，名为`components`。
2. 在包里实现`services.Component`。
3. 在结构体上打上`@component`进行组件注入。
4. 配置组件，在服务配置中添加组件名的属性作为组件配置。

函数中获取当前服务的组件：
```go
component, has := services.GetComponent[T](ctx, componentName)
```