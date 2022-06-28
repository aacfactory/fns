```yaml
name: "project name"
# service engine
service:
  maxWorkers: 0
  workerMaxIdleSeconds: 10
  handleTimeoutSeconds: 10
# logger
log:
  level: info
  formatter: console
  color: true
# http server
server:
  port: 80
# openapi config
oas:
  title: "Project title"
  description: |
    Project description
  terms: https://terms.fns
  contact:
    name: hello
    url: https://hello.fns
    email: hello@fns
  license:
    name: license
    url: https://license.fns
  servers:
    - url: https://test.hello.fns
      description: test
    - url: https://hello.fns
      description: prod
# service config >>>
# authorizations service
authorizations:
  encoding:
    method: "RS512"
    publicKey: "path of public key"
    privateKey: "path of private key"
    issuer: ""
    audience:
      - foo
      - bar
    expirations: "720h0m0s"
  store: {}
examples:
  componentA: {}
# service config <<<
```