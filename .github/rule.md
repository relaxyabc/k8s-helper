# Copilot Instructions for Go Project

## 代码风格

- 遵循 [Effective Go](https://golang.org/doc/effective_go.html) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)。
- 使用 gofmt 格式化所有 Go 代码。
- 变量、函数、类型命名采用驼峰式（CamelCase），包名使用小写短名词。
- 每个包应有简明的 package 注释。
- 使用 `go doc` 查看包文档，确保文档清晰易懂。
- 禁止重复代码，重复逻辑应提取为函数或方法。

## 目录结构

```
.
├── .github/        # 项目配置与协作规范（如 Copilot 规则等）
├── .vscode/        # VSCode 编辑器相关配置
├── crypto/         # 加解密相关工具包
├── dao/            # 数据库访问与数据持久化逻辑
├── mcp/            # 主业务逻辑与服务实现
├── test/           # 所有测试代码，含各包测试用例
├── tools/          # Kubernetes 及相关工具方法
├── LICENSE         # 许可证
├── main.go         # 程序入口
└── README.md       # 项目说明文档
```

## 结构与约定

- 每个功能模块单独放在对应的包目录下。
- 涉及重构时  a.go 从 包 A 迁移到 包 B 时，包A 中的 a.go 文件应直接删除,且已有代码引用 包A 的需要更新为引用 包B。
- main.go 仅用于程序入口和启动逻辑，业务逻辑应放在其他包中。
- 测试文件以 `_test.go` 结尾，全部放在 test 目录下。
- 测试代码需要将关键结果和日志输出到控制台，便于调试。

## 错误处理

- 错误优先返回，避免 panic，除非遇到不可恢复的错误。
- 错误信息应简洁明了，必要时加上调用栈或上下文。

## 依赖管理

- 使用 Go Modules（go.mod/go.sum）管理依赖。
- 新增依赖请使用 `go get` 并及时更新 go.mod/go.sum。

## 注释与文档

- 导出函数、类型、接口必须有注释，说明其用途和用法。
- 复杂逻辑应有必要的行内注释。

## 性能与安全

- 优先使用内置类型和库，避免不必要的第三方依赖。
- 注意并发安全，合理使用 goroutine、channel、sync 包。

## 其他

- 所有提交前请确保通过 `go test ./...`。
- 遵循本项目已有的目录结构和代码风格。
