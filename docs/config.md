# Configuration

---

## Core
```yaml
name: "foo"                   # project name
runtime:                      # service runtime
  maxWorkers: 0               # max goroutines, default is 256 * 1024
  workerMaxIdleSeconds: 10    # max idle goroutine second, default is 10
  handleTimeoutSeconds: 10    # fn request handle timeout, default is 10
```
## Logger
```yaml
log:
  level: info                 # level: debug, info, warn, error
  formatter: console          # output formatter: console, json
  color: true                 # enable or disable colorful console output
```
## Http
```yaml
server:
  port: 80                    # port
  cors:                       # cors, default is allow all
    allowedOrigins:
      - "*"
    allowedHeaders:
      - "X-Foo"
    exposedHeaders:
      - "X-Foo"
    allowCredentials: false
    maxAge: 10
  tls:                        # https, default is nil
    kind: ""                  # tls kind: DEFAULT, SSC, ACME 
    options: {}               # options of kind
  options: {}                 # http server options
  interceptors: {}            # http interceptors config
```
### TLS
`DEFAULT` kind options: 
```yaml
options: 
  cert: "path of cert pem file"
  key: "path of private key pem file"
```
`SSC` kind options:
```yaml
options:
  ca: "path of ca pem file"
  caKey: "path of private key pem file"
```
`ACME` kind options is not defined, but you can use [ACMES](https://github.com/aacfactory/acmes)
```shell
go get github.com/aacfactory/acmes/client
```
Make options type
```go
type AcmeConfig struct {
	CA string 
	Key string
	Endpoint string
	Domain string
}
```
Register ACME Loader
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
set options
```yaml
options:
  ca: "path of ca pem file"
  caKey: "path of private key pem file"
  endpoint: "endpoint of acmes server"
  domain: "which domain to be obtained"
```
### Server Options
`Fasthttp` options:
```yaml
options:
  readTimeoutSeconds: 2
  maxWorkerIdleSeconds: 10
  maxRequestBody: "4MB"
  reduceMemoryUsage: true
```
### Interceptors
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
## Cluster
cluster config, read [doc](https://github.com/aacfactory/fns/blob/main/docs/cluster.md) for more.
```yaml
cluster:
  devMode: false                  # when dev mode is true, current node will not be pushed to other members
  nodesProxyAddress: "ip:port"    # when dev mode is true, and current node can not use member address to access them, then use a proxy to access members. 
  kind: ""                        # cluster kind: members, swarm, kubernetes.
  client:
    maxIdleConnSeconds: 0
    maxConnsPerHost: 0
    maxIdleConnsPerHost: 0
    requestTimeoutSeconds: 0
  options: {}                     # options of kind
```
## Service
service config, name of root node is service name, second layer nodes are components (if components are used).   
For example: `jwt authorizations`
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
