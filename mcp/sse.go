package mcp

import (
	"context"
	"net/http"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/relaxyabc/k8s-helper/common"
	"k8s.io/klog/v2"
)

var (
	// sseSessionCtxMap stores the long-lived context for each SSE session.
	sseSessionCtxMap = make(map[string]context.Context)
	// sseSessionCtxMapMutex protects sseSessionCtxMap.
	sseSessionCtxMapMutex = sync.RWMutex{}
)

// GetSSEContextBySessionID retrieves the long-lived context for a given session ID.
func GetSSEContextBySessionID(sid string) context.Context {
	sseSessionCtxMapMutex.RLock()
	defer sseSessionCtxMapMutex.RUnlock()
	return sseSessionCtxMap[sid]
}

// AppSessionIDKey is a custom key for storing our application-specific session ID in the context.
// This is used to pass our session ID into the mcp-go library's context chain.
type AppSessionIDKey struct{}

type SSEServer struct {
	sseServer *server.SSEServer
}

func NewSSEServer(mcpServer *MCPServer, sm *HTTPSessionManager, opts ...server.SSEOption) *SSEServer {
	baseOpts := []server.SSEOption{
		// This function is called by the mcp-go library when a new SSE connection is established.
		// We use this hook to pass our application's session ID into the context that will be
		// available later in the OnRegisterSession hook.
		server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			// The SessionMiddleware has already placed our session ID ("mcp-session-xxx")
			// into the request's context.
			appSessionID, ok := r.Context().Value(common.ContextKeyMcpSession).(string)
			if !ok || appSessionID == "" {
				klog.Warningf("[MCP-SSE] ContextFunc: Could not find application session ID in request context. This is unexpected.")
				return ctx
			}

			// We add our session ID to the context under a unique key. This context is then
			// passed along the chain inside the mcp-go library.
			klog.Infof("[MCP-SSE] ContextFunc: Found app session ID '%s'. Placing it into the SSE context.", appSessionID)
			return context.WithValue(ctx, AppSessionIDKey{}, appSessionID)
		}),
	}

	allOpts := append(baseOpts, opts...)
	sse := server.NewSSEServer(mcpServer.server, allOpts...)
	return &SSEServer{sseServer: sse}
}

func (s *SSEServer) SSEHandler() http.Handler {
	return s.sseServer.SSEHandler()
}

func (s *SSEServer) MessageHandler() http.Handler {
	return s.sseServer.MessageHandler()
}

func (s *SSEServer) SendEventToSession(sessionID string, event any) error {
	return s.sseServer.SendEventToSession(sessionID, event)
}

// RegisterPerSessionTool 为指定 sessionID 注册专属工具
func (s *SSEServer) RegisterPerSessionTool(mcpServer *MCPServer, sessionID string) {
	tool := mcp.NewTool("my_custom_tool", mcp.WithDescription("会话专属工具"))
	mcpServer.server.AddSessionTool(sessionID, tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("你访问了专属工具，sessionID: " + sessionID), nil
	})
}
