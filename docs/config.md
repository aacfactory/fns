# 配置

---

Fns是以`yaml`格式的多环境方式进行配置。

配置内容是`fns.yaml`合并`fns-{active}.yaml`组成。`fns-{active}.yaml`由环境变量`FNS-ACTIVE`的值决定。

## 基本配置项

### Runtime
配置协程池、容器内`GOMAXPROCS`等。
```yaml
runtime:
  procs:
    min: 1
  workers:
    max: 64
    maxIdleSeconds: 5
```

### Services
服务配置。
```yaml
services:
  {服务名}:
    ...
```
如[SQL](https://github.com/aacfactory/fns-contrib/tree/main/databases/sql)服务的配置：
```yaml
services:
  sql:
    kind: "standalone"
    isolation: 2
    transactionMaxAge: 10
    debugLog: true
```

### Hook
[钩子](https://github.com/aacfactory/fns/blob/main/docs/hooks.md)配置：
```yaml
hooks:
  {钩子名}:
    ...
```