# 集群

---

## 固定成员

集群中只会发现指定的成员节点，不过节点之间会共享已发现的节点。

Config:

```yaml
cluster:
  options:
    members:
      - "192.168.11.1:8080"
      - "192.168.11.2:8080"
```

## DOCKER SWARM

阅读 [doc](https://github.com/aacfactory/fns-contrib/tree/main/cluster/swarm) 获取更多信息。

## KUBERNETES

阅读 [doc](https://github.com/aacfactory/fns-contrib/tree/main/cluster/kubernetes) 获取更多信息。