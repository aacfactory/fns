# 配置

---

## Core
```yaml
name: "foo"                   # 项目名称
runtime:                      # 服务运行时
  maxWorkers: 0               # 最大 goroutines, 默认为 256 * 1024
  workerMaxIdleSeconds: 10    # 最大协程空闲秒数, 默认为 10
  handleTimeoutSeconds: 10    # 函数处理超时秒数, 默认为 10
```
## Logger
```yaml
log:
  level: info                 # 等级: debug, info, warn, error
  formatter: console          # 输出格式: console, json
  color: true                 # 是否开启色彩
```
## Http
```yaml
server:
  port: 80                    # 端口
  cors:                       # cors, 默认是全部允许
    allowedOrigins:
      - "*"
    allowedHeaders:
      - "X-Foo"
    exposedHeaders:
      - "X-Foo"
    allowCredentials: false
    maxAge: 10
  tls:                        # https, 默认为空
    kind: ""                  # tls 类型: DEFAULT, SSC, ACME 
    options: {}               # 类型配置
  websocket:
    readBufferSize: "4k"
    writeBufferSize: "4k"
    enableCompression: false
    maxConns: 0
  options: {}                 # 服务其它配置
  interceptors: {}            # 拦截器配置
```
### TLS
`DEFAULT` 类型配置: 
```yaml
options: 
  cert: "证书路径"
  key: "密钥路径"
```
`SSC` 类型配置:
```yaml
options:
  ca: "CA路径"
  caKey: "CA密钥路径"
```
`ACME` 类型配置没有预设, 不过可以阅读 [ACMES](https://github.com/aacfactory/acmes) 来获取更多信息。
```shell
go get github.com/aacfactory/acmes/client
```
创建ACME配置
```go
type AcmeConfig struct {
	CA string 
	Key string
	Endpoint string
	Domain string
}
```
注册ACME加载器
```go
ssl.RegisterLoader("ACME", func(options configuares.Config) (serverTLS *tls.Config, clientTLS *tls.Config, err error) {
	config := AcmeConfig{}
	configErr := options.As(&config)
	// handle configErr
	ca, _ := ioutil.ReadFile(config.CA)
	key, _ := ioutil.ReadFile(config.Key)
	acme, connErr := client.New(ca, key, config.Endpoint) 
	// handle connErr 
	serverTLS, _, err = acme.Obtain(context.TODO(), config.Domain) 
	// handle err
})
```
设置配置
```yaml
options:
  ca: "CA路径"
  caKey: "CA密钥路径"
  endpoint: "acme的服务端点"
  domain: "待申请域名"
```
### 服务配置
`Fasthttp` 配置:
```yaml
options:
  readTimeoutSeconds: 2
  maxWorkerIdleSeconds: 10
  maxRequestBody: "4MB"
  reduceMemoryUsage: true
```
### 拦截器配置
`pprof`
```yaml
interceptors:
  pprof:
    password: "bcrypt password"
```
## OpenAPI
```yaml
oas:
  title: "Project title"
  description: |
    Project description
  terms: "https://terms.fns"
  contact:
    name: "hello"
    url: "https://hello.fns"
    email: "hello@fns"
  license:
    name: "license"
    url: "https://license.fns"
  servers:
    - url: "https://test.hello.fns"
      description: "test"
    - url: "https://hello.fns"
      description: "prod"
```
## 集群配置
阅读 [doc](https://github.com/aacfactory/fns/blob/main/docs/cluster_zh.md) 获取更多信息。
```yaml
cluster:
  devMode: false                  # 开发模式，当开启后，本地开发的服务不会推送到集群中。
  nodesProxyAddress: "ip:port"    # 当开发模式开启后，如果需要访问集群但直接访问不行，则可以添加一个集群访问代理。
  kind: ""                        # 集群类型: members, swarm, kubernetes.
  client:
    maxIdleConnSeconds: 0
    maxConnsPerHost: 0
    maxIdleConnsPerHost: 0
    requestTimeoutSeconds: 0
  options: {}                     # 集群类型的配置
```
## 服务
根节点的名称为服务名，第二层节点一般是组件名(如果组件模式是被使用的).   
举例: `jwt authorizations`
```yaml
authorizations:
  encoding:
    method: "RS512"
    publicKey: "path of public key"
    privateKey: "path of private key"
    issuer: ""
    audience:
      - "foo"
      - "bar"
    expirations: "720h0m0s"
  store: {}
```
