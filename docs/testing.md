# Testing

---

单元测试。启动相应配置后测试服务函数。

## 配置环境
```go
err := tests.Setup(service) // service 是对应实例。
```
环境选项：

| 选项               | 说明   |
|------------------|------|
| WithConfigActive | 选择配置 |
| WithConfig       | 使用配置 |
| WithDependence   | 添加依赖 |
| WithTransport    | 替换传输 |


## 测试案例
```go
// 获取上下行
ctx := tests.TODO()

// 测试服务函数，调用生成的函数代理
r, err := foo.Do(ctx, param)

// 最后关闭
tests.Teardown()
```