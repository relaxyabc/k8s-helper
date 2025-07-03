package mcp

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s.io/klog/v2"
)

// session context map
var (
	streamSessionCtxMap      = make(map[string]context.Context)
	streamSessionCtxMapMutex = sync.RWMutex{}
)

// GetSessionContextBySessionID 获取指定 sessionID 的 context
func GetSessionContextBySessionID(sid string) context.Context {
	streamSessionCtxMapMutex.RLock()
	defer streamSessionCtxMapMutex.RUnlock()
	return streamSessionCtxMap[sid]
}

// SetSessionContext 设置 sessionID 的 context
func SetSessionContext(sid string, ctx context.Context) {
	streamSessionCtxMapMutex.Lock()
	defer streamSessionCtxMapMutex.Unlock()
	streamSessionCtxMap[sid] = ctx
}

// DeleteSessionContext 删除 sessionID 的 context
func DeleteSessionContext(sid string) {
	streamSessionCtxMapMutex.Lock()
	defer streamSessionCtxMapMutex.Unlock()
	delete(streamSessionCtxMap, sid)
}

// StreamableServer 兼容 streamable/http 的推送与会话专属工具
// 只持有 MCPServer 实例
type StreamableServer struct {
	mcpServer *MCPServer
}

func NewStreamableServer(mcpServer *MCPServer) *StreamableServer {
	return &StreamableServer{mcpServer: mcpServer}
}

// SendEventToSession 通过 MCPServer 的推送接口实现
func (s *StreamableServer) SendEventToSession(sessionID string, event any) error {
	ctx := GetSessionContextBySessionID(sessionID)
	if ctx == nil {
		return errors.New("session context not found")
	}
	klog.Infof("[Streamable] 推送事件到 session: %s, event=%v", sessionID, event)
	// 类型断言，保证推送类型正确
	payload, ok := event.(map[string]any)
	if !ok {
		return errors.New("event 必须为 map[string]any 类型")
	}
	return s.mcpServer.server.SendNotificationToSpecificClient(sessionID, "notification", payload)
}

// RegisterPerSessionTool 为指定 sessionID 注册专属工具
func (s *StreamableServer) RegisterPerSessionTool(sessionID string) {
	tool := mcp.NewTool("my_custom_tool", mcp.WithDescription("会话专属工具"))
	s.mcpServer.server.AddSessionTool(sessionID, tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("你访问了专属工具，sessionID: " + sessionID), nil
	})
}

// HasSessionContext 判断 sessionID 是否有 context
func (s *StreamableServer) HasSessionContext(sessionID string) bool {
	return GetSessionContextBySessionID(sessionID) != nil
}

// StartSessionContextDebugPrint 每秒输出一次所有 sessionID
func StartSessionContextDebugPrint() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			streamSessionCtxMapMutex.RLock()
			sessions := make([]string, 0, len(streamSessionCtxMap))
			for sid := range streamSessionCtxMap {
				sessions = append(sessions, sid)
			}
			streamSessionCtxMapMutex.RUnlock()
			if len(sessions) == 0 {
				klog.Infof("[Streamable] 当前无活跃 session context")
			} else {
				klog.Infof("[Streamable] 活跃 session context: %v", sessions)
			}
		}
	}()
}
