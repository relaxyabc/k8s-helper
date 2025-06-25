package mcp

import (
	"log"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

type SSEServer struct {
	server *server.MCPServer
}

func NewSSEServer(s *MCPServer) *SSEServer {
	return &SSEServer{server: s.server}
}

func (s *SSEServer) Start(addr string) {
	http.HandleFunc("/events", s.handleSSE)
	log.Printf("SSE server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("SSE server error: %v", err)
	}
}

func (s *SSEServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 简单示例：推送一条欢迎消息
	_, _ = w.Write([]byte("event: message\ndata: Welcome to SSE!\n\n"))
	w.(http.Flusher).Flush()

	// 这里可根据实际业务推送更多事件
}
