# 项目解构

---


```
|-- main.go                             # 主文件
|-- configs/                            # 配置文件夹
     |-- fns.yaml                       # 默认配置
     |-- fns-local.yaml                 # set FNS-ACTIVE=local, 本地开发配置.
     |-- fns-dev.yaml                   # FNS-ACTIVE=dev
     |-- fns-test.yaml                  # FNS-ACTIVE=test
     |-- fns-prod.yaml                  # FNS-ACTIVE=prod
|-- hooks/                              # 回调函数
|-- internal/
     |-- generator/
          |-- main.go                   # 代码生成器
|-- modules/                            # 业务模块
     |-- services.go                    # 业务服务，它将在`go generate`后自动生成.
     |-- dependencies.go                # 依赖服务
     |-- foo/                           # 服务
          |-- doc.go                    # 服务定义
          |-- fns.go                    # 服务实体，它将在`go generate`后自动生成.
          |-- some_fn.go                # 函数
|-- repositories/                       # 数据
     |-- some_db_model.go               # 模型

```