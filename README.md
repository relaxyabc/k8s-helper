# k8s-helper

k8s-helper 是一个基于 MCP 协议的 Kubernetes 多集群管理与运维工具，支持通过 HTTP/STDIO/SSE 等多种方式与客户端交互，便于自动化和平台集成。

## 主要功能
- 多集群统一管理（PostgreSQL 存储集群信息）
- 查询集群、命名空间、Pod、Deployment、DaemonSet 等资源
- 支持滚动重启 Deployment/DaemonSet
- 查询集群 Kubernetes 版本
- 支持 SOCKS5 代理和跳过 TLS 校验
- 基于 session 的用户权限与会话管理
- MCP 工具接口自动注册与权限过滤

## 快速开始
### 依赖
- Go 1.24+
- PostgreSQL 数据库
- 依赖见 `go.mod`

### 构建
```shell
# 拉取依赖
go mod tidy
# 构建可执行文件
go build -o k8s-helper main.go
```

### 运行
```shell
# 以 stdio 模式运行（适合 MCP 客户端集成）
./k8s-helper -t stdio -dbhost <host> -dbport <port> -dbname <db> -dbuser <user> -dbpass <pass> [-proxy <socks5>""]

# 以 HTTP 模式运行（适合 REST API 调用）
./k8s-helper -t http -dbhost <host> -dbport <port> -dbname <db> -dbuser <user> -dbpass <pass> [-proxy <socks5>""]

# 以 SSE 模式运行（Server-Sent Events）
./k8s-helper -t sse -dbhost <host> -dbport <port> -dbname <db> -dbuser <user> -dbpass <pass> [-proxy <socks5>""]
```

## 常用接口说明（HTTP Tool 风格）
- `GET  /clusters` 查询所有集群
- `GET  /namespaces?cluster_name=xxx` 查询指定集群的 namespace
- `GET  /pods?cluster_name=xxx&namespace=xxx` 查询指定命名空间下的 Pod
- `GET  /deployments?cluster_name=xxx&namespace=xxx` 查询 Deployment
- `GET  /daemonsets?cluster_name=xxx&namespace=xxx` 查询 DaemonSet
- `POST /rollout_restart_deployment?cluster_name=xxx&namespace=xxx&name=xxx` 滚动重启 Deployment
- `POST /rollout_restart_daemonset?cluster_name=xxx&namespace=xxx&name=xxx` 滚动重启 DaemonSet
- `GET  /k8s_version?cluster_name=xxx` 查询集群 Kubernetes 版本

## 数据库表结构

本项目仅依赖 `clusters` 表，所有 namespace、pod、deployment、daemonset 等资源均通过实时调用 Kubernetes API 获取，无需落库。

### clusters 表结构
| 字段名         | 类型    | 说明           |
| -------------- | ------- | -------------- |
| cluster_name   | text    | 集群名称       |
| ip             | text    | 集群 IP 地址   |
| kube_config    | text    | kubeconfig 内容|

> 说明：`clusters` 表用于存储所有可管理的 Kubernetes 集群信息。主键字段请根据实际数据库表结构设置，`cluster_name` 仅为业务字段。

## 测试
```shell
go test ./tools
```

## 许可证
Apache License 2.0

## 使用帮助

## VS Code 集成使用指南（mcp.json 配置）

本项目支持通过 MCP 协议（stdio/http）与 VS Code 的 MCP 客户端插件集成。你可以通过配置 `.vscode/mcp.json` 文件，直接在 VS Code 侧边栏调用 k8s-helper。

### 1. stdio 模式
- mcp.json 推荐配置示例：
  ```json
  {
    "servers": {
      "mcp-k8s-stdio": {
        "type": "stdio",
        "command": "k8s-helper",
        "args": [
          "-t=stdio","-proxy=10.11.12.13:1080","-dbhost=20.21.22.23","-dbport=5432","-dbname=dbname","-dbuser=username","-dbpass=pwd"
        ]
      }
    }
  }
  ```
- 启动命令（与上面 args 保持一致）：
  ```shell
  ./k8s-helper -t stdio -proxy=10.11.12.13:1080 -dbhost=20.21.22.23 -dbport=5432 -dbname=dbname -dbuser=username -dbpass=xxxx
  ```

### 2. http 模式
- 启动命令：
  ```shell
  ./k8s-helper -t http -dbhost <host> -dbport <port> -dbname <db> -dbuser <user> -dbpass <pass>
  ```
- mcp.json 配置示例：
  ```json
  {
    "servers": {
      "k8s-helper-http": {
        "url": "http://localhost:8080/mcp"
      }
    }
  }
  ```

### 3. 使用方法
- 在 `.vscode` 目录下新建或编辑 `mcp.json`，填入上述配置。
- 选择对应的 server，即可在侧边栏直接调用 k8s-helper 的能力。

> 注意：如需使用 http 模式，需保证 k8s-helper 服务已启动并监听 8080 端口。


### 4. mcp-inspector 使用
``` bash
npx @modelcontextprotocol/inspector
```
