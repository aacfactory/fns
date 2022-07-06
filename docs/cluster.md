# Cluster

---

## members 
For a fixed number of clusters, existing nodes in other members will be automatically added, but other members will not be found by themselves.

Config:
```yaml
cluster:
  devMode: false                    # if true, current node will not be pushed into cluster
  nodesProxyAddress: ""             # used for dev mode and current node can not access members by member's address
  kind: "members"                   # cluster member finder type
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
Read [doc](https://github.com/aacfactory/fns-contrib/tree/main/cluster/swarm) for more.

## KUBERNETES
Read [doc](https://github.com/aacfactory/fns-contrib/tree/main/cluster/kubernetes) for more.