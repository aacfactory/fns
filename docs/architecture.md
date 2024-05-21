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