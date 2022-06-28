# Fns
Fn services for Golang. Simplify the development process by using standardized development scheme. Every thing is service.

## Features
* [x] Applicable to enterprise projects
  * [x] Standardization
  * [x] Environmental configuration
  * [x] Rapid development by code generations
  * [x] Built in RBAC permission scheme
* [x] Code generations
    * [x] Service of fn
    * [x] Proxy of fn
    * [x] Open api documents
    * [x] Request argument validation
    * [x] Authorizations validation
    * [x] Permissions validation
* [x] High concurrency
  * [x] Goroutines pool
  * [x] Fasthttp (Default)
  * [x] Http3 (Optional)
  * [x] Request Barrier (Multiple identical requests will only process the first one, and others will share the results of the first one)
* [x] TLS
  * [x] Standard 
  * [x] SSC (Auto generate cert and key by self sign ca)
  * [x] ACMEs
* [x] Cluster
  * [x] Designated members 
  * [x] DOCKER SWARM (Auto find members by container label)
  * [x] KUBERNETES (Auto find members by pod label)
* [x] Authorizations
  * [x] Fns default token
  * [x] Json web token
* [x] Permissions
  * [x] RBAC
* [x] Monitoring standards
  * [x] Metric
  * [x] Tracing
* [x] Databases
    * [x] SQL
      * [x] Serviceability (Internal service)
      * [x] Distributed SQL transaction
    * [x] Postgres ORM
    * [x] Mysql ORM
    * [ ] Dgraph
    * [ ] GraphQL to SQL
    * [x] Redis
* [x] Message queue
    * [x] RabbitMQ
    * [x] Kafka
    * [x] Nats.IO
* [ ] DDD
* [ ] Extra listeners
  * [ ] Websockets
  * [ ] MQTT
* [ ] Third party integration
  * [ ] Oauth
  * [ ] Payments
  * [ ] SMS
  * [ ] Notifications

## Quickstart
Read [usage]() for more.
### Create project
First: install `fnc` commander.
```shell
go install github.com/aacfactory/fnc
```
Second: use `fnc` create a fns project.
```shell
mkdir {your project path}
cd {your project path}
fnc create .
```
Third: look `main.go`, `config`, `modules` to understand [project structure](). 
### Create fn
Happy coding by `FNS`.
### Send request to fn
```shell
curl -H "Content-Type: application/json" -X POST -d '{}' http://ip:port/service/fn
```

## Environmental configuration


## Coding
### Service
Create service package under `modules`, such as `modules/users`.  
Create service `doc.go`, such as `modules/users/doc.go`.
```go
// Package users
// @service users
// @title User
// @description User service
// @internal false
package users
```
`@service` value is service name;  
`@title` value is service title, used as openapi tag;  
`@description` value is service title, used as openapi tag description;   
`@internal` value is mark the service can not be accessed from public network request;
### Fn
Create `fn` file, such as `get.go`. 
```go
// GetArgument
// @title Get User Argument
// @description Get User Argument
type GetArgument struct {
	// Id
	// @title id
	// @description id
	Id int64 `json:"id" validate:"required" message:"id is invalid"`
}

// User
// @title User
// @description User profile
type User struct {
    // Id
    // @title id
    // @description id
    Id string `json:"id"`
    // Mobile
    // @title Mobile
    // @description Mobile
    Mobile string `json:"mobile"`
    // Name
    // @title Name
    // @description Name
    Name string `json:"name"`
    // Gender
    // @title Gender 
	// @enum M,F,N
    // @description Gender
    Gender string `json:"gender"`
    // Age
    // @title Age
    // @description Age
    Age int `json:"age"`
    // Avatar
    // @title Avatar
    // @description Avatar
    Avatar string `json:"avatar"`
}


// get
// @fn get
// @validate true
// @authorization false
// @permission false
// @internal false
// @title Get user info
// @description >>>
// Get user info function
// ----------
// errors:
// * user_get_failed
// <<<
func get(ctx context.Context, argument GetArgument) (v *User, err errors.CodeError) {
	v = &User{
		Id:     fmt.Sprintf("%v", argument.Id),
		Mobile: "000",
		Name:   "foo",
		Gender: "F",
		Age:    10,
		Avatar: "bar",
	}
	return
}
```
### Code generation
```shell
cd {your project home}
fnc codes .
```
Or you can use `go generate` and `//go:generate fnc codes .`
### Deploy service
Add service in `main.go`.
```go
app.Deploy(users.Service())
```

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