# Fns 
[简体中文](https://github.com/aacfactory/fns/blob/main/README_zh.md)

---

Fn services for Golang. Simplify the development process by using standardized development scheme. Every thing is service.

## Features
* [x] Applicable to enterprise projects
  * [x] Standardization
  * [x] Environmental configuration
  * [x] Rapid development by code generations
  * [x] Built in authorizations service (schema is customizable)
  * [x] Built in RBAC permission service
  * [x] Built in metric service (schema is customizable)
  * [x] Built in tracing service (schema is customizable)
  * [x] Traceable and searchable error schema
  * [x] Searchable log content schema
* [x] Code generations
    * [x] Service of fn
    * [x] Proxy of fn
    * [x] Open api documents
    * [x] Request argument validation
    * [x] Authorizations validation
    * [x] Permissions validation
* [x] High concurrency
  * [x] Goroutines pool
  * [x] Fasthttp as default http server (Http3 is optionally)
  * [x] Request Barrier (Multiple identical requests will only process the first one, and others will share the results of the first one)
* [x] TLS
  * [x] Standard 
  * [x] SSC (Auto generate cert and key by self sign ca, it is useful for building http3 based cluster)
  * [x] ACMEs (Easy to use and supports auto-renew)
* [x] Cluster
  * [x] Designated members 
  * [x] DOCKER SWARM (Auto find members by container label)
  * [x] KUBERNETES (Auto find members by pod label)
* [x] Databases 
    * [x] SQL
      * [x] Serviceability (Internal service)
      * [x] Distributed SQL transaction (Non-intrusive and all databases are supported)
      * [x] Postgres ORM (High performance)
      * [x] Mysql ORM (To be tested)
      * [ ] GraphQL to SQL
    * [x] Redis (Internal service)
    * [ ] Dgraph
* [x] Message queue
    * [x] RabbitMQ
    * [x] Kafka
    * [x] Nats.IO
* [ ] DDD
* [ ] Extra listeners
  * [ ] MQTT
* [ ] Third party integration
  * [ ] Oauth
  * [ ] Payments
  * [ ] SMS
  * [ ] Notifications

## Quickstart
### Create project
First: install `fnc` commander.
```shell
go install github.com/aacfactory/fnc@latest
```
Second: use `fnc` create a fns project.
```shell
mkdir {your project path}
cd {your project path}
fnc create 
```
Third: look `main.go`, `config`, `modules` to understand [project structure](https://github.com/aacfactory/fns/blob/main/docs/structure.md). 

Happy coding. Read [usage](https://github.com/aacfactory/fns/blob/main/docs/usage.md) for more.
### Read openapi document
Setup `swagger-ui`, then open it.
```shell
docker run -d --rm --name swagger-ui \
 -p 80:8080 \
 -e SWAGGER_JSON_URL=http://ip:port/documents/oas \ 
 swaggerapi/swagger-ui 
```
### Send request to fn
```shell
curl -H "Content-Type: application/json" -X POST -d '{}' http://ip:port/service/fn
```

## Environmental configuration
Use `FNS-ACTIVE` environment to control which configuration to be merged into default configuration and used. 
The environment value could be `local`, `dev`, `qa`, and `prod`.
Read [configuration](https://github.com/aacfactory/fns/blob/main/docs/config.md) for more.

## Cluster
Fns use Gossip-like schema to manage a cluster. 
Just turn on the cluster mode in project, 
then the endpoints of fn service will be auto registered and discovered,
there is no difference of coding between cluster mode and standalone mode,
so it is very easy to use. Read [cluster](https://github.com/aacfactory/fns/blob/main/docs/cluster.md) for more.

## Benchmark
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
