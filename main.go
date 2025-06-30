package main

import (
	"flag"
	"net/http"
	"time"

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
	flag.StringVar(&transport, "t", "", "Transport type (stdio, http, or sse)")
	flag.StringVar(&transport, "transport", "", "Transport type (stdio, http, or sse)")
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
		httpSessionMgr := mcp.NewHTTPSessionManager(30*time.Minute, s)
		klog.Info("[MCP] Starting in HTTP mode, using MCPServer as handler...")
		mux := http.NewServeMux()
		mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("SESSIONID")
			if err == nil {
				httpSessionMgr.DeleteSession(c.Value, s)
				http.SetCookie(w, &http.Cookie{Name: "SESSIONID", Value: "", Path: "/", MaxAge: -1})
			}
			w.Write([]byte("logout success"))
		})
		// 自定义 /mcp handler，显式处理 sid、用户、会话注册
		mux.Handle("/mcp", s.ServeHTTP())
		handler := mcp.SessionMiddleware(httpSessionMgr, s, mux)
		listenAddr := ":" + addr
		klog.Infof("[MCP] HTTP server listening on %s (via MCPServer)", listenAddr)
		if err := http.ListenAndServe(listenAddr, handler); err != nil {
			klog.Fatalf("Server error: %v", err)
		}
	case "sse":

		// Create the MCP server with the hooks.
		s := mcp.NewMCPServer()
		httpSessionMgr := mcp.NewHTTPSessionManager(30*time.Minute, s)

		listenAddr := ":" + addr
		klog.Infof("[MCP] Starting SSE server on %s", listenAddr)

		baseURL := "http://localhost:" + addr

		sseServer := mcp.NewSSEServer(s, httpSessionMgr,
			server.WithStaticBasePath("/mcp"),
			server.WithKeepAliveInterval(3*time.Minute),
			server.WithBaseURL(baseURL),
		)

		// 注册 SSE 推送工具，传递 sseServer
		s.RegisterSSEPushTool(sseServer)

		mux := http.NewServeMux()

		// Handle the base /mcp path for handshakes.
		mcpHandler := s.ServeHTTP()
		mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
			klog.Infof("[ROUTING_DEBUG] Path: %s -> /mcp handler", r.URL.Path)
			if r.URL.Path != "/mcp" {
				http.NotFound(w, r)
				return
			}
			mcpHandler.ServeHTTP(w, r)
		})

		// Handle the SSE connection path.
		sseHandler := sseServer.SSEHandler()
		mux.Handle("/mcp/sse", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			klog.Infof("[ROUTING_DEBUG] Path: %s -> /mcp/sse handler", r.URL.Path)
			sseHandler.ServeHTTP(w, r)
		}))

		// Handle the SSE message path.
		messageHandler := sseServer.MessageHandler()
		mux.Handle("/mcp/message", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			klog.Infof("[ROUTING_DEBUG] Path: %s -> /mcp/message handler", r.URL.Path)
			messageHandler.ServeHTTP(w, r)
		}))

		handler := mcp.SessionMiddleware(httpSessionMgr, s, mux)
		klog.Infof("SSE server listening on %s", listenAddr)
		if err := http.ListenAndServe(listenAddr, handler); err != nil {
			klog.Fatalf("Server error: %v", err)
		}
	default:
		klog.Fatalf("Invalid transport type: %s. Must be 'stdio', 'http' or 'sse'", transport)
	}
}
