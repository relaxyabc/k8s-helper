package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/relaxyabc/k8s-helper/dao"
	"github.com/relaxyabc/k8s-helper/mcp"
	"k8s.io/klog/v2"
)

func main() {
	var transport string
	var dbhost, dbport, dbname, dbuser, dbpass, proxy string
	var aesKeyFlag string
	var addr string
	flag.StringVar(&transport, "t", "", "Transport type (stdio, http, or streamable)")
	flag.StringVar(&transport, "transport", "", "Transport type (stdio, http, or streamable)")
	flag.StringVar(&addr, "addr", "8080", "服务监听地址端口")
	flag.StringVar(&dbhost, "dbhost", "localhost", "数据库地址")
	flag.StringVar(&dbport, "dbport", "5432", "数据库端口")
	flag.StringVar(&dbname, "dbname", "postgres", "数据库名")
	flag.StringVar(&dbuser, "dbuser", "postgres", "数据库用户名")
	flag.StringVar(&dbpass, "dbpass", "", "数据库密码")
	flag.StringVar(&proxy, "proxy", "", "代理地址")
	flag.StringVar(&aesKeyFlag, "aeskey", "k8s-mcp-client", "AES加密key")
	flag.Parse()

	if transport == "" {
		transport = "stdio"
	}

	dao.InitDBByArgs(dbhost, dbport, dbname, dbuser, dbpass)
	mcp.Init(proxy, aesKeyFlag, transport)

	switch transport {
	case "stdio":
		s := mcp.NewMCPServer()
		klog.Info("[MCP] Starting in stdio mode, waiting for client to connect...")
		if err := s.ServeStdio(); err != nil {
			klog.Fatalf("Server error: %v", err)
		}
	case "http":
		s := mcp.NewMCPServer()
		klog.Info("[MCP] Starting in HTTP mode, using MCPServer as handler...")
		mux := http.NewServeMux()
		mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
			_, err := r.Cookie("SESSIONID")
			if err == nil {
				http.SetCookie(w, &http.Cookie{Name: "SESSIONID", Value: "", Path: "/", MaxAge: -1})
			}
			w.Write([]byte("logout success"))
		})
		// 自定义 /mcp handler，显式处理 sid、用户、会话注册
		mux.Handle("/mcp", s.ServeHTTP())
		listenAddr := ":" + addr
		klog.Infof("[MCP] HTTP server listening on %s (via MCPServer)", listenAddr)
		if err := http.ListenAndServe(listenAddr, mux); err != nil {
			klog.Fatalf("Server error: %v", err)
		}
	case "streamable":
		// 1. 创建一个新的 MCP 服务器实例
		mcpServer := server.NewMCPServer(
			"k8s-helper",          // 服务器名称
			"1.0.0",               // 服务器版本
			server.WithRecovery(), // 启用崩溃恢复
		)

		// 2. 添加一个用于演示服务器端推送的工具
		streamTool := mcpgo.NewTool(
			"start_stream",
			mcpgo.WithDescription("Starts a simulated long-running process and streams progress updates."),
			mcpgo.WithNumber("duration_seconds", mcpgo.Required(), mcpgo.Description("Duration of the simulated process in seconds.")),
		)

		// 3. 为工具添加处理函数
		mcpServer.AddTool(streamTool, func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
			duration, err := request.RequireInt("duration_seconds")
			if err != nil {
				return mcpgo.NewToolResultError(fmt.Sprintf("Invalid duration: %v", err)), nil
			}

			// 从上下文中获取 MCP 服务器和会话
			mcpServer := server.ServerFromContext(ctx)
			if mcpServer == nil {
				klog.Infof("Could not get MCPServer from context. This tool requires an active session for streaming.")
				return mcpgo.NewToolResultError("Server session not found for streaming."), nil
			}

			session := server.ClientSessionFromContext(ctx)
			if session == nil {
				klog.Infof("Could not get ClientSession from context. This tool requires an active session for streaming.")
				return mcpgo.NewToolResultError("Server session not found for streaming."), nil
			}

			klog.Infof("Starting simulated stream for %d seconds for session %s", duration, session.SessionID())

			// 模拟长时间运行的任务，并发送进度通知
			for i := 0; i < duration; i++ {
				select {
				case <-ctx.Done(): // 检查请求上下文是否被取消（例如客户端断开连接）
					klog.Infof("Stream for session %s cancelled.", session.SessionID())
					return mcpgo.NewToolResultText("Stream cancelled."), nil
				case <-time.After(1 * time.Second):
					progress := float64(i+1) / float64(duration) * 100
					message := fmt.Sprintf("Progress: %.2f%% (%d/%d seconds)", progress, i+1, duration)
					klog.Infof("Session %s: Sending progress: %s", session.SessionID(), message)

					// 发送通知
					msg := map[string]interface{}{
						"message":   message,
						"progress":  progress,
						"timestamp": time.Now().Unix(),
					}
					// err := mcpServer.SendNotificationToClient(ctx, "notifications/progress", msg)
					err := mcpServer.SendNotificationToClient(ctx, "window/progress", msg)
					// err := mcpServer.SendNotificationToClient(ctx, "window/showMessage", msg)
					if err != nil {
						klog.Errorf("Failed to send notification to session %s: %v", session.SessionID(), err)
						// 通常这里会根据错误类型决定是否继续，但为演示目的，我们只是记录并继续
					}
				}
			}

			klog.Infof("Simulated stream finished for session %s.", session.SessionID())
			return mcpgo.NewToolResultText(fmt.Sprintf("Simulated process finished after %d seconds.", duration)), nil
		})

		// 4. 初始化 StreamableHTTPServer
		// 这是处理 HTTP 传输细节的关键组件，包括管理 SSE 流。
		httpServer := server.NewStreamableHTTPServer(
			mcpServer,
			server.WithEndpointPath("/mcp"), // 设置 MCP 服务器的统一 HTTP 端点路径
			server.WithHeartbeatInterval(3*time.Second), // 设置心跳间隔，保持 SSE 连接活跃
		)

		// 5. 将 StreamableHTTPServer 注册到 Go 的标准 HTTP 路由器
		//http.Handle("/mcp", httpServer)
		// 使用自定义处理程序包装MCP服务器的ServeHTTP方法
		http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
			httpServer.ServeHTTP(w, r)
			// 可以根据crw.statusCode进行额外的逻辑处理
			fmt.Printf("Request to %s, Path: %s\n", r.Method, r.URL.Path)
		})

		// 6. 启动 HTTP 服务器
		listenAddr := ":" + addr
		klog.Infof("Starting MCP Streamable HTTP server on %s", listenAddr)
		klog.Fatal(http.ListenAndServe(listenAddr, nil))
	default:
		klog.Fatalf("Invalid transport type: %s. Must be 'stdio', 'http' or 'streamable'", transport)
	}
}
