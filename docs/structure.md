# Project structure

---


```
|-- main.go                             # main
|-- configs/                            # config files and tls files .etc
     |-- fns.yaml                       # default config
     |-- fns-local.yaml                 # set FNS-ACTIVE=local, which is used for local deveplment.
     |-- fns-dev.yaml                   # FNS-ACTIVE=dev
     |-- fns-test.yaml                  # FNS-ACTIVE=test
     |-- fns-prod.yaml                  # FNS-ACTIVE=prod
|-- hooks/                              # hooks
|-- internal/
     |-- generator/
          |-- main.go                   # code generator bin
|-- modules/                            # biz modules
     |-- services.go                    # all services (it will be auto regenerated after invoking `go generate` command)
     |-- dependencies.go                # dependency services 
     |-- foo/                           # biz service
          |-- doc.go                    # definition of service
          |-- fns.go                    # fn service (it will be auto regenerated after invoking `go generate` command)
          |-- some_fn.go                # fn
|-- repositories/                       # database access objects
     |-- some_db_model.go               # database access object

```