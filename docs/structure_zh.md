# 项目解构

---


```
|-- main.go                             # main
|-- config/                             # 配置目录
     |-- fns.yaml                       # 默认配置
     |-- fns-local.yaml                 # FNS-ACTIVE=local
     |-- fns-dev.yaml                   # FNS-ACTIVE=dev
     |-- fns-qa.yaml                    # FNS-ACTIVE=qa
     |-- fns-prod.yaml                  # FNS-ACTIVE=prod
|-- module/                             # 业务模块
     |-- services.go                    # 所有的业务函数服务聚合，由`fnc`生成，无须要手动修改
     |-- dependencies.go                # 依赖的服务
     |-- foo/                           # 业务服务
          |-- doc.go                    # 服务定义
          |-- fns.go                    # 函数服务与代理 (它由`fnc`生成，无须要手动修改)
          |-- some_fn.go                # 函数
|-- repository/                         # 数据库访问对象包
     |-- some_db_model.go               # 数据库访问对象

```