# 集群

---

## 固定成员

集群中只会发现指定的成员节点，不过节点之间会共享已发现的节点。

Config:

```yaml
cluster:
  devMode: false                    # 如果为真，则当前节点不会注册到集群中
  nodesProxyAddress: ""             # 在开发模式下，如果当前节点不能通过成员地址访问成员时，可以使用代理的方式访问
  kind: "members"                   # 集群成员发现者类型
  client:
    maxIdleConnSeconds: 10
    maxConnsPerHost: 512
    maxIdleConnsPerHost: 64
    requestTimeoutSeconds: 10
  options:
    members:
      - "192.168.11.1:8080"
      - "192.168.11.2:8080"
```

## DOCKER SWARM

阅读 [文档](https://github.com/aacfactory/fns-contrib/tree/main/cluster/swarm) 获取更多信息。

## KUBERNETES

阅读 [文档](https://github.com/aacfactory/fns-contrib/tree/main/cluster/kubernetes) 获取更多信息。