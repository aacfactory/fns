# websocket
增加一个内置的websocket service，PATH为/_websocket_/send。  
函数逻辑为寻找本地链接并发送。   
websocket代理改成链接表（id：Conn），Conn是channel或http client proxy。

# services
大幅重构。   
增加一个manager，提供服务注册与发现，原本的discovery修改成只能发现，没有注册，由manager提供discovery。   
handle中预处理处理auth，如果有auth token，则校验。   
配置中增加concurrency，移到build中处理。   
删除discovery配置，配置中增加cluster配置，在build中构建manager与barrier。   
cluster为etcd或redis，不再有docker和K8S。   
barrier不再在service的handle中处理，迁移至services的handle中或者request中，request改名为dispatch？   


# application
配合services调整


# context
user增加passed属性以及只读函数，当user id被set后，passed改为true。   
app中增加locker，getLocker(KEY)，releaseLocker(locker)。   
app中增加websocket代理。
