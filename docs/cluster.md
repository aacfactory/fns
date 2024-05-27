# Cluster

---

## 开启配置
```yaml
cluster:
  name: ""                      # 如 hazelcast
  proxy: false                  # 是否开启代理功能，一般用于开发环境中，当开启时，则作为本地开发所链接的地址。
  secret: ""                    # 用于集群内部访问的签名校验
  hostRetriever: ""             # 地址获取器，适用于Kubernetes，详情见Kubernetes。
  option:                       # 选项，具体见注册表的相关配置。
```

## 本地开发 
本地配置
```yaml
cluster:            
  name: "dev"
  option:
    proxyAddr: "192.168.100.101:18082"  # 远程地址
```
本地链接的远程服务配置，一般配置在Gateway中，也可以配置在一个可被访问的业务服务上。
```yaml
cluster:            
  proxy: true                                            
```

## KUBERNETES
当运行在`kubernetes`环境中时，请使用 [inject](https://kubernetes.io/zh-cn/docs/tasks/inject-data-application/environment-variable-expose-pod-information/) 把 POD IP 注入到`FNS-HOST`环境变量中，最后把配置中`cluster.hostRetriever`的值设置为`env`。

## Sharing
分布式共享，主要提供 `Lockers` 和 `Store`。

## Proxy
请求自动代理。解析请求，转发到对应服务。

当开启代理后，建议把`cors`、`websocket`、`compress`、`cache-control`等都放在代理。

代理是开放的，可以被丰富成网关。

开启代理：
```go
func main() {
	// set system environment to make config be active, e.g.: export FNS-ACTIVE=local
	fns.
		New(
			// 注入代理
			fns.Proxy(
				proxies.Transport(fast.New()),
				proxies.Middleware(cors.New()),
				proxies.Middleware(compress.New()),
				proxies.Handler(documents.New()),
			),
		)
	return
}
```

配置代理(配置内容为传输的配置)：
```yaml
proxy:
  port: 18080
```