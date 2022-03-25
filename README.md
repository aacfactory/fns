# 项目简介
FNS 是一个类 FaaS 的 http 服务框架。   
其主要目标是提供快速构建一个 FaaS 项目的模式与非侵入式生态。  
其宗旨为亲和性标准化去构建一个可持续发展的 http 项目。

## 特性列表
* [x] 标准化
* [x] 环境化配置
* [x] 代码生成
  * [x] fns service
  * [x] open api documents 
  * [x] 参数验证
* [x] 高并发 
* [ ] 分布式
  * [ ] DOCKER SWARM
  * [ ] KUBERNETES
  * [ ] ETCD 
* [ ] authorizations
  * [x] JWT
* [x] 重复提交拦截
  * [x] 本地
  * [x] redis
* [ ] 数据库
  * [x] sql 
  * [x] 分布式 sql 事务 
  * [x] postgres orm
  * [ ] mysql orm
  * [ ] dgraph
  * [ ] graphQL to sql
  * [x] redis
* [ ] 消息中间件
  * [ ] RabbitMQ
  * [ ] Kafka
  * [ ] RocketMQ
* [ ] DDD
* [ ] OAUTH CLIENT
  * [ ] Wechat
  * [ ] Apple
  * [ ] Alipay
* [ ] OAUTH SERVER
* [ ] 第三方支付
  * [ ] Wechat
  * [ ] Alipay
   
## 使用说明
使用要求：  
* go1.17 或更高
* go mod 项目环境

下载代码生成器（具体版本见fnc项目）
```bash
go install github.com/aacfactory/fnc@v1.6.1
```

获取库
```bash
go get github.com/aacfactory/fns
```

建议的项目结构
```
|-- main.go
|-- config/
     |-- app.yaml
     |-- app-dev.yaml
     |-- app-prod.yaml
|-- module/
     |-- foo/
          |-- doc.go
          |-- some_fn.go
|-- repository/
     |-- some_db_model.go
```

编辑 config/app.yaml
```yaml
name: project name
description: |
  project description.
terms: project terms
log:
  level: info
  formatter: console
  color: true
http:
  host: 0.0.0.0
  port: 80
  publicHost: 127.0.0.1
  publicPort: 80
  keepAlive: true
  readBufferSize: 1MB
  writeBufferSize: 4MB
  cors:
    enable: true
    allowedOrigins:
      - '*'
    allowedMethods:
      - GET
      - POST
services:
  handleTimeoutSecond: 10
  authorization:
    enable: true
    kind: jwt
    config:
      method: RS512
      publicKey: ./config/jwt.public.pem
      privateKey: ./config/jwt.private.pem
      issuer: FNS
      audience:
        - web
        - ios
        - android
        - wechat-miniapp
      expirations: 360h0m0s
```
创建内网传送密码 config/sk.txt
```
YOUR_PASSWORD
```
编辑 main.go
```go

var (
	AppVersion = "v0.0.1"
)

//go:generate fnc -p .
func main() {
	app, appErr := fns.New(
          // 在这之前请创建配置环境的环境变量，变量名没有约束。
		fns.ConfigRetriever("./config", "YAML", fns.ConfigActiveFromENV("YOUR_PROJECT_ACTIVE"), "app", '-'),
		fns.SecretKeyFile("./config/sk.txt"),
		fns.Version(AppVersion),
	)

	if appErr != nil {
		panic(fmt.Sprintf("%+v\n", appErr))
		return
	}

	deployErr := app.Deploy(
		// TODO: ADD SERVERS
	)

	if deployErr != nil {
		app.Log().Error().Cause(deployErr).Caller().Message("app deploy service failed")
		return
	}

	runErr := app.Run(context.TODO())

	if runErr != nil {
		app.Log().Error().Cause(runErr).Caller().Message("app run failed")
		return
	}

	if app.Log().DebugEnabled() {
		app.Log().Debug().Caller().Message("running...")
	}

	app.Sync()

	if app.Log().DebugEnabled() {
		app.Log().Debug().Message("stopped!!!")
	}

}

```
创建 fn service，编辑foo/doc.go
```go
// Package foo
// @service foo
// @title open tag title
// @description open tag title
// @internal false
package foo
```
创建 fn
```go
// CountNumParam
// @title open api title
// @description open api description
type CountNumParam struct {
	// BrandName
	// @title open api title
	// @description open api description
	BrandName string `json:"brandName" validate:"required,not_blank" message:"brandName is invalid"`
	// GroupName
	// @title open api title
	// @description open api description
	GroupName string `json:"groupName"`
	// Kind
	// @title open api title
	// @description open api description
	Kind string `json:"kind"`
	// UserId
	// @title open api title
	// @description open api description
	UserId int64 `json:"userId"`
}

// CountNumResult
// @title open api title
// @description open api description
type CountNumResult struct {
	Num int `json:"num"`
}

// countNum
// @fn count_num
// @validate true
// @authorization false
// @permission false
// @internal false
// @title open api title
// @description >>>
// open api description
// ----------
// 支持 markdown
// <<<
func countNum(ctx fns.Context, param CountNumParam) (v *CountNumResult, err errors.CodeError) {
	// TODO
     // 参数必须两个：第一个固定fns.Context，第二个必须是value struct
     // 返回值必须两个：第一个是返回的内容（支持PTR STRUCT、ARRAY、MAP），第二个必须是errors.CodeError
	return
}

```
使用 go:generate 命令生成 service 
```bash
go generate
```
将生成的 service 布到 main 程序中。
```go
deployErr := app.Deploy(
	foo.Service(),
)
```
### JWT
```go
claims := jwt.NewUserClaims()
claims.SetIntUserId(userId)
claims.SetSub("some sub")
token, tokenErr := ctx.App().Authorizations().Encode(ctx, claims)
```
### 数据库
导入对应的驱动，目前支持 `github.com/lib/pq`。  
在repository创建model
```go
type UserRow struct {
	Id               int64                 `col:"ID,incrPk" json:"ID"`
	Version          int64                 `col:"VERSION,aol" json:"VERSION"`
	UID              string                `col:"UID" json:"UID"`
	RegisterTime     time.Time             `col:"REGISTER_TIME" json:"REGISTER_TIME"`
	Nickname         string                `col:"NICKNAME" json:"NICKNAME"`
	Certified        bool                  `col:"CERTIFIED" json:"CERTIFIED"`
	Mobile           string                `col:"MOBILE" json:"MOBILE"`
	Gender           string                `col:"GENDER" json:"GENDER"`
     Avatar           *File                 `col:"AVATAR,json" json:"AVATAR_ROW" copy:"AVATAR"`
}

func (r UserRow) TableName() (string, string) {
	return "scheme", "table_name"
}

// File
// @title 文件
// @description 文件
type File struct {
	// Id
	// @title 标识
	// @description 标识
	Id int64 `json:"id"`
	// UID
	// @title unique id
	// @description unique id
	UID string `json:"uid"`
	// Schema
	// @title http schema
	// @description http schema
	Schema string `json:"schema"`
	// Domain
	// @title 域名
	// @description 域名
	Domain string `json:"domain"`
	// Path
	// @title 路径
	// @description 路径
	Path string `json:"path"`
	// MimeType
	// @title 文件类型
	// @description 文件类型
	MimeType string `json:"mimeType"`
	// Bucket
	// @title 文件所在桶
	// @description 文件所在桶
	Bucket string `json:"bucket"`
	// URL
	// @title URL地址
	// @description URL地址
	URL string `json:"url"`
}

```
在 fn 中使用（postgres）
```go
// QUERY ONE
row := &repository.UserRow{}
fetched, getErr := postgres.QueryOne(ctx, postgres.NewConditions(postgres.Eq("ID", param.Id)), row)
// QUERY list
rows := make([]*repository.UserRow, 0, 1)
cond := postgres.NewConditions(postgres.Eq("NICKNAME", param.Nickname))
sort := postgres.NewOrders().Asc("ID")
rng := postgres.NewRange(param.Offset, param.Length)
has, queryErr := postgres.QueryWithRange(ctx, cond, sort, rng, &rows)
// INSERT
insertErr := postgres.Insert(ctx, row)
// MODIFY
row.Gender = gender
modErr := postgres.Modify(ctx, row)
// DELETE
deleteErr := postgres.Delete(ctx, row)
```

## 压力测试
在AMD 3950X 64G内存的单机上 K6 压力测试结果（50 VUS 30s）。

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