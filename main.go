package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/relaxyabc/k8s-helper/dao"
	"github.com/relaxyabc/k8s-helper/mcp"
)

var (
	httpSessionMgr = mcp.NewHTTPSessionManager(30 * time.Minute)
)

func main() {
	var transport string
	var dbhost, dbport, dbname, dbuser, dbpass, proxy string
	var aesKeyFlag string
	flag.StringVar(&transport, "t", "", "Transport type (stdio, http, or sse)")
	flag.StringVar(&transport, "transport", "", "Transport type (stdio, http, or sse)")
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

	s := mcp.NewMCPServer()

	switch transport {
	case "stdio":
		log.Println("[MCP] Starting in stdio mode, waiting for client to connect...")
		if err := s.ServeStdio(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case "http":
		log.Println("[MCP] Starting in HTTP mode, using MCPServer as handler...")
		mux := http.NewServeMux()
		mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("SESSIONID")
			if err == nil {
				httpSessionMgr.DeleteSession(c.Value)
				http.SetCookie(w, &http.Cookie{Name: "SESSIONID", Value: "", Path: "/", MaxAge: -1})
			}
			w.Write([]byte("logout success"))
		})
		// 自定义 /mcp handler，显式处理 sid、用户、会话注册
		mux.Handle("/mcp", s.ServeHTTP())
		handler := mcp.SessionMiddleware(httpSessionMgr, mux)
		addr := ":8080"
		log.Printf("[MCP] HTTP server listening on %s (via MCPServer)", addr)
		if err := http.ListenAndServe(addr, handler); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case "sse":
		log.Printf("[MCP] Starting SSE server on :8080")
		sseServer := mcp.NewSSEServer(s, httpSessionMgr,
			server.WithStaticBasePath("/mcp"),
			server.WithKeepAliveInterval(30*time.Second),
			server.WithBaseURL("http://localhost:8080"),
		)
		sseServer.Start(":8080")
	default:
		log.Fatalf("Invalid transport type: %s. Must be 'stdio', 'http' or 'sse'", transport)
	}
}
