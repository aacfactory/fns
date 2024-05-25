# FNS

---

Golang的函数式框架。标准化协议来简化开发过程。

## 特性
* [x] 标准化
  * [x] 万物皆[函数](https://github.com/aacfactory/fns/blob/main/docs/fn.md)
  * [x] 天然分布式[服务](https://github.com/aacfactory/fns/blob/main/docs/architecture.md#Service)
* [x] 适用于企业开发。
  * [x] [环境化配置](https://github.com/aacfactory/fns/blob/main/docs/config.md)
  * [x] 敏捷开发
  * [x] [内置身份校验](https://github.com/aacfactory/fns/blob/main/docs/authorizations.md) 
  * [x] [内置权限验证](https://github.com/aacfactory/fns/blob/main/docs/perissions.md) 
  * [x] [内置指标收集](https://github.com/aacfactory/fns/blob/main/docs/metric.md)
  * [x] [内置链路跟踪](https://github.com/aacfactory/fns/blob/main/docs/tracing.md)
  * [x] [支持分布式跟踪的错误](https://github.com/aacfactory/errors)
  * [x] [支持可检索的日志格式](https://github.com/aacfactory/fns/blob/main/docs/logs.md)
  * [x] [自动化分布式API文档](https://github.com/aacfactory/fns/blob/main/docs/openapi.md)
* [x] 高并发
  * [x] 协程池
  * [x] 支持 Fasthttp
  * [x] [栅栏](https://github.com/aacfactory/fns/blob/main/docs/barrier.md)
* [x] TLS
  * [x] [自动化自签证书](https://github.com/aacfactory/fns/blob/main/docs/trasnport.md#SSC)
* [x] 多版本HTTP
  * [x] [Fasthttp](https://github.com/aacfactory/fns/blob/main/docs/trasnport.md#Fasthttp)
  * [x] [基于Fasthttp的HTTP2](https://github.com/aacfactory/fns/blob/main/docs/trasnport.md#Fasthttp2)
  * [x] [标准库](https://github.com/aacfactory/fns/blob/main/docs/trasnport.md#Standard)
  * [x] [HTTP3](https://github.com/aacfactory/fns-contrib/blob/main/transports/http3/README.md)
  * [x] [通用的WEBSOCKET](https://github.com/aacfactory/fns-contrib/blob/main/transports/handlers/websockets/readme.md)
* [x] 集群
  * [x] [分布式共享](https://github.com/aacfactory/fns/blob/main/docs/cluster.md#Sharing)
  * [x] [Hazelcast](https://github.com/aacfactory/fns-contrib/blob/main/cluster/hazelcasts/README.md) 
  * [x] [Redis](https://github.com/aacfactory/fns-contrib/blob/main/databases/redis/README.md)
  * [x] [Kubernetes](https://github.com/aacfactory/fns/blob/main/docs/cluster.md#KUBERNETES)
* [x] 官方服务 
    * [x] 数据库
      * [x] [SQL](https://github.com/aacfactory/fns-contrib/blob/main/databases/sql/README.md)
        * [x] 分布式事务
        * [x] [Postgres ORM](https://github.com/aacfactory/fns-contrib/blob/main/databases/postgres/README.md) 
        * [x] [Mysql ORM](https://github.com/aacfactory/fns-contrib/blob/main/databases/mysql/readme.md)
      * [x] [Redis](https://github.com/aacfactory/fns-contrib/blob/main/databases/redis/README.md)
    * [ ] 消息
        * [ ] RabbitMQ
        * [x] [Kafka](https://github.com/aacfactory/fns-contrib/blob/main/message-queues/kafka/README.md)
        * [ ] RocketMQ
        * [ ] MQTT 
    * [ ] 第三方服务集成
      * [ ] Oauth
      * [ ] 支付
      * [ ] 短信
      * [ ] 通知

## 使用
### 创建项目
一、安装`fns`。
```shell
go install github.com/aacfactory/fns/cmd/fns@latest
```
二、使用`fns`创建项目。
```shell
mkdir {your project path}
cd {your project path}
fns init --mod={mod} --img={docker image name} --work={true} --version={go version} . 
## Example
# fns init --mod=foo.com/project --img={foo.com/project} --work=true --version=1.21.0 .
```

### 编写代码
一、理解[项目结构](https://github.com/aacfactory/fns/blob/main/docs/structure.md)

二、设置[配置](https://github.com/aacfactory/fns/blob/main/docs/config.md)与[依赖](https://github.com/aacfactory/fns/blob/main/docs/dependence.md)

三、创建[服务标识](https://github.com/aacfactory/fns/blob/main/docs/architecture.md#Service)。

四、创建[函数](https://github.com/aacfactory/fns/blob/main/docs/fn.md)。

五、运行`go generate`[生成代码](https://github.com/aacfactory/fns/blob/main/docs/generation.md)。

### 运行项目
设置环境变量激活[配置](https://github.com/aacfactory/fns/blob/main/docs/config.md)。

如`FNS-ACTIVE=local`，则运行时使用`fns-local.yaml`的配置。

### 测试与分析
* [单元测试](https://github.com/aacfactory/fns/blob/main/docs/testing.md)
* [pprof](https://github.com/aacfactory/fns-contrib/blob/main/transports/handlers/pprof/README.md)

### 发布API文档
开启[API文档](https://github.com/aacfactory/fns-contrib/blob/main/transports/handlers/documents/README.md)功能，通过浏览器或相关OPENAPI工具进行查阅。

## 集群
开启[集群](https://github.com/aacfactory/fns/blob/main/docs/architecture.md#Cluster)功能即可，无需其它改动。

当运行在`kubernetes`环境中时，请使用 [inject](https://kubernetes.io/zh-cn/docs/tasks/inject-data-application/environment-variable-expose-pod-information/) 把 POD IP 注入到`FNS-HOST`环境变量中，最后把配置中`cluster.hostRetriever`的值设置为`env`。

## 客制化HTTP服务
* [HTTP服务](https://github.com/aacfactory/fns/blob/main/docs/trasnport.md#Http)
* [中间件](https://github.com/aacfactory/fns/blob/main/docs/trasnport.md#Middleware)
* [处理器](https://github.com/aacfactory/fns/blob/main/docs/trasnport.md#Handler)

## 第三方服务集成
### 服务化 
[服务化](https://github.com/aacfactory/fns/blob/main/docs/architecture.md#Service)第三方服务的SDK，业务服务通过函数进行调用。

### 组件化
[组件化](https://github.com/aacfactory/fns/blob/main/docs/architecture.md#Component)第三方服务的SDK，然后注入到业务服务中。