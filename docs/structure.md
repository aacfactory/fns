# Project structure

---


```
|-- main.go                             # main
|-- config/                             # config files and tls files .etc
     |-- fns.yaml                       # default config
     |-- fns-local.yaml                 # FNS-ACTIVE=local
     |-- fns-dev.yaml                   # FNS-ACTIVE=dev
     |-- fns-qa.yaml                    # FNS-ACTIVE=qa
     |-- fns-prod.yaml                  # FNS-ACTIVE=prod
|-- module/                             # biz modules
     |-- services.go                    # all services (it will be auto regenerated after invoking `fnc service create` command)
     |-- dependencies.go                # dependency services 
     |-- foo/                           # biz service
          |-- doc.go                    # definition of service
          |-- fns.go                    # fn service (it will be auto regenerated after invoking `fnc codes .` command)
          |-- some_fn.go                # fn
|-- repository/                         # database access objects
     |-- some_db_model.go               # database access object

```