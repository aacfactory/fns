# Fns
[English](https://github.com/aacfactory/fns/blob/main/README.md)

---

Golang 的类函数式服务，使用标准化进行简化的开发方案。关键信息是 Fns 中所有事情都是服务化的。

## 特性
* [x] 适用于企业应用开发
    * [x] 标准化
    * [x] 环境化配置
    * [x] 通过代码生成来急速开发
    * [x] 内置认证服务
    * [x] 内置RBAC权限服务
    * [x] 内置指标服务
    * [x] 内置访问跟踪访问
    * [x] 可跟踪且可查询的错误方案
    * [x] 可查询的日志内容
* [x] 代码生成器
    * [x] 函数服务
    * [x] 函数代理
    * [x] OAS文档
    * [x] 请求参数校验
    * [x] 认证校验
    * [x] 权限校验
* [x] 高并发
    * [x] Goroutines 池
    * [x] 默认使用 Fasthttp （Http3是可选的）
    * [x] 请求栅栏（一个请求处理时间内，其余接收到的相同请求不会执行函数，而是等待并共享第一个请求执行的结果）
* [x] TLS
    * [x] 支持标准TLS
    * [x] SSC (根据给到的CA自动生成两端的自签证书和密钥，适用于基于Http3的集群)
    * [x] ACMEs (简化使用并支持自动续签)
* [x] 集群
    * [x] 固定成员
    * [x] DOCKER SWARM (根据容器标签自动发现成员)
    * [x] KUBERNETES (根据POD标签自动发现成员)
* [x] 数据库
    * [x] SQL
        * [x] 服务化的
        * [x] 分布式事务 (不依赖数据库服务类型)
        * [x] Postgres ORM (高性能的)
        * [x] Mysql ORM (待经过测试)
        * [ ] GraphQL 映射 SQL
    * [x] Redis 
    * [ ] Dgraph
* [x] 消息队列
    * [x] RabbitMQ
    * [x] Kafka
    * [x] Nats.IO
* [ ] DDD
* [ ] 额外的服务监听
    * [ ] MQTT
* [ ] 第三方集成
    * [ ] Oauth
    * [ ] 支付
    * [ ] 短信
    * [ ] 通知

## 快速开始
### 创建项目
第一步: 安装 `fnc` 命令.
```shell
go install github.com/aacfactory/fnc@latest
```
第二步: 使用 `fnc`创建项目。
```shell
mkdir {your project path}
cd {your project path}
fnc create 
```
第三步: 查看 `main.go`, `config`, `modules` 去理解 [项目解构](https://github.com/aacfactory/fns/blob/main/docs/structure_zh.md).

享受开发. 查阅 [使用](https://github.com/aacfactory/fns/blob/main/docs/usage_zh.md) 获取更多信息.

### 查询接口文档
搭建 `swagger-ui`。
```shell
docker run -d --rm --name swagger-ui \
 -p 80:8080 \
 -e SWAGGER_JSON_URL=http://ip:port/documents/oas \ 
 swaggerapi/swagger-ui 
```

### 发送请求
```shell
curl -H "Content-Type: application/json" -H "X-Fns-Device-Id: client-uuid" -X POST -d '{}' http://ip:port/service/fn
```

## 环境化配置
使用 `FNS-ACTIVE` 系统环境去控制哪个配置合并到默认配置并运用。
系统环境值可为 `local`, `dev`, `qa`, 和 `prod`.
阅读 [配置](https://github.com/aacfactory/fns/blob/main/docs/config_zh.md) 获取更多信息。

## 集群
Fns 使用类Gossip的方案进行集群管理。
使用时只要开启集成即可，之后函数服务的端点将会自动注入集群和被发现。
集群和单机在开发中没有什么区别。所有集群用起来会非常的简单。 
阅读 [集群](https://github.com/aacfactory/fns/blob/main/docs/cluster_zh.md) 获取更多信息。

## 压力测试
CPU: AMD 3950X;   
MEM: 64G;   
RPS: 118850.991763/s;
```shell

          /\      |‾‾| /‾‾/   /‾‾/
     /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
   /          \   |  |\  \ |  (‾)  |
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: ./test.js
     output: -

  scenarios: (100.00%) 1 scenario, 50 max VUs, 1m0s max duration (incl. graceful stop):
           * default: 50 looping VUs for 30s (gracefulStop: 30s)


running (0m30.0s), 00/50 VUs, 3565697 complete and 0 interrupted iterations
default ✓ [======================================] 50 VUs  30s

     ✓ status was 200

     checks.........................: 100.00% ✓ 3565697       ✗ 0
     data_received..................: 564 MB  19 MB/s
     data_sent......................: 521 MB  17 MB/s
     http_req_blocked...............: avg=1.58µs   min=0s med=0s   max=5.57ms   p(90)=0s      p(95)=0s
     http_req_connecting............: avg=24ns     min=0s med=0s   max=2.31ms   p(90)=0s      p(95)=0s
     http_req_duration..............: avg=261.31µs min=0s med=0s   max=12.92ms  p(90)=844.7µs p(95)=1ms
       { expected_response:true }...: avg=261.31µs min=0s med=0s   max=12.92ms  p(90)=844.7µs p(95)=1ms
     http_req_failed................: 0.00%   ✓ 0             ✗ 3565697
     http_req_receiving.............: avg=26.91µs  min=0s med=0s   max=8.65ms   p(90)=0s      p(95)=29.3µs
     http_req_sending...............: avg=10.53µs  min=0s med=0s   max=7.7ms    p(90)=0s      p(95)=0s
     http_req_tls_handshaking.......: avg=0s       min=0s med=0s   max=0s       p(90)=0s      p(95)=0s
     http_req_waiting...............: avg=223.86µs min=0s med=0s   max=12.43ms  p(90)=641.9µs p(95)=1ms
     http_reqs......................: 3565697 118850.991763/s
     iteration_duration.............: avg=412.23µs min=0s med=92µs max=118.81ms p(90)=1ms     p(95)=1ms
     iterations.....................: 3565697 118850.991763/s
     vus............................: 50      min=50          max=50
     vus_max........................: 50      min=50          max=50

```

