package mcp

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/relaxyabc/k8s-helper/common"
)

type SSEServer struct {
	sseServer *server.SSEServer
}

func NewSSEServer(mcpServer *MCPServer, sm *HTTPSessionManager, opts ...server.SSEOption) *SSEServer {
	baseOpts := []server.SSEOption{
		server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			sid := ""
			mcpId := r.URL.Query().Get(common.McpIDParam)
			log.Printf("[MCP-SSE] new sse request, mcpId=%s", mcpId)
			if mcpId != "" {
				mcpId = strings.TrimSpace(mcpId)
				mcpId, _ = url.QueryUnescape(mcpId)
				mcpId = strings.ReplaceAll(mcpId, " ", "+")
				userId, userRole := ParseUserIDAndRoleFromSID(mcpId)
				log.Printf("[MCP-SSE] Parsed mcpId from url: userId=%s, userRole=%s", userId, userRole)

				if userId != "" {
					ses := sm.CreateSession(userId)
					ses.Data["role"] = userRole
					sid = ses.ID
					sessionUserInfoMapMutex.Lock()
					sessionUserInfoMap[ses.ID] = struct{ UserID, Role string }{userId, userRole}
					sessionUserInfoMapMutex.Unlock()
				}
			}
			if sid == "" {
				log.Printf("[MCP-SSE] invalid mcpId, create empty session")
				ses := sm.CreateSession("")
				sid = ses.ID
			}

			return context.WithValue(ctx, common.ContextKeyMcpSession, sid)
		}),
	}
	allOpts := append(baseOpts, opts...)
	sse := server.NewSSEServer(mcpServer.server, allOpts...)
	return &SSEServer{sseServer: sse}
}

func (s *SSEServer) Start(addr string) {
	log.Printf("[MCP] SSE server listening on %s", addr)
	if err := s.sseServer.Start(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
