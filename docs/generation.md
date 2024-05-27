# Generation

---
Fns的代码生成器。

## 使用
一般在`internal/generator/`包下。

`main.go` 中构建生成器。

```go
func main() {
	g := generates.New()
	if err := g.Execute(context.Background(), os.Args...); err != nil {
		fmt.Println(fmt.Sprintf("%+v", err))
	}
}

```

## 调整
`generates.New()` 中可以添加选项来调整。

一般会用到`WithAnnotations`多一点，比如增加`sql`的注解支持。

| 选项               | 说明       |
|------------------|----------|
| WithName         | 设置bin的名字 |
| WithModulesDir   | 设置mod的目录 |
| WithAnnotations  | 添加新的注解支持 |
| WithBuiltinTypes | 添加新的内置类型 |
| WithGenerator    | 添加额外的生成器 |
