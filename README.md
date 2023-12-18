# Fns

---

Fn services for Golang. Simplify the development process by using standardized development scheme. Every thing is function.

## Features
* [x] Applicable to enterprise projects
  * [x] Standardization
  * [x] Environmental configuration
  * [x] Rapid development by code generations
  * [x] Built in authorizations service (schema is customizable)
  * [x] Built in permission service
  * [x] Built in metric
  * [x] Built in tracing
  * [x] Traceable and searchable error schema
  * [x] Searchable log content schema
* [x] High concurrency
  * [x] Goroutines pool
  * [x] Fasthttp as default http server ([Http3]() is optionally)
  * [x] Request Barrier (Used for sharing result and idempotent)
* [x] TLS
  * [x] Standard 
  * [x] SSC (Auto generate cert and key by self sign ca)
  * [x] ACMEs (Easy to use and supports auto-renew)
* [x] Cluster
  * [x] [Hazelcast]() 
  * [x] [Redis]()
* [x] Official services 
    * [x] [SQL]()
      * [x] Distributed SQL transaction 
      * [x] [Postgres ORM]() 
      * [x] [Mysql ORM]()
    * [x] [Redis]()
      * [x] Shared store
      * [x] Shared lockers
      * [x] Shared barrier
    * [x] Message queue
        * [ ] RabbitMQ
        * [x] [Kafka]()
        * [ ] RocketMQ
    * [ ] Third party integration
      * [ ] Oauth
      * [ ] Payments
      * [ ] SMS
      * [ ] Notifications
* [x] Documents
  * [x] Openapi
  * [ ] Official

## Quickstart
### Creation
First: install `fns` commander.
```shell
go install github.com/aacfactory/fns/cmd/fns@latest
```
Second: use `fns` create a fns project.
```shell
mkdir {your project path}
cd {your project path}
fns init --mod={mod_name} . 
```
Third: look `main.go`, `configs`, `modules` to understand [project structure](https://github.com/aacfactory/fns/blob/main/docs/structure.md). 

Last: add [dependencies]() and setup [config]().

### Coding
1. First: create a [service]() ident.  
2. Second: create a [function]().  
3. Last: run `go generate` to generate source codes.
4. Happy coding. 

### Running
Setup `FNS-ACTIVE` env, such as `FNS-ACTIVE=local`, that the `fns-local.yaml` is used.

### View documents
Setup [documents](), then done, so easy.

### Testing
Just write test units, read [tests]() for more details.

## Cluster
Use [hazelcast]() or [redis](), and create a [proxy]() server as gateway, then done, so easy.   
When used in `kubernetes`, then [inject](https://kubernetes.io/zh-cn/docs/tasks/inject-data-application/environment-variable-expose-pod-information/) pod ip into `FNS-HOST`, and set `env` into config field `cluster.hostRetriever`.