package tools

import (
	"context"
	"strconv"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// StreamablePusher 定义推送接口，避免 import cycle
// 需实现 SendEventToSession(sessionID string, event any) error 和 HasSessionContext(sessionID string) bool
type StreamablePusher interface {
	SendEventToSession(sessionID string, event any) error
	HasSessionContext(sessionID string) bool
}

// NewPushMessageTool 返回 push_message 工具及其 handler
func NewPushMessageTool(streamableServer StreamablePusher) (tool mcpgo.Tool, handler func(context.Context, mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error)) {
	tool = mcpgo.NewTool("push_message",
		mcpgo.WithDescription("向当前 session 推送一条消息"),
		mcpgo.WithString("message", mcpgo.Required(), mcpgo.Description("要推送的内容")),
	)
	handler = func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := req.GetArguments()
		msg, _ := args["message"].(string)
		sid := ""
		if session := server.ClientSessionFromContext(ctx); session != nil {
			sid = session.SessionID()
		}
		// 检查 context 是否存在
		if !streamableServer.HasSessionContext(sid) {
			return mcpgo.NewToolResultError("推送失败：用户未在线或 session 已失效"), nil
		}
		for i := 1; i <= 5; i++ {
			payload := map[string]any{
				"message":   msg,
				"index":     i,
				"timestamp": time.Now().Format(time.RFC3339),
			}
			err := streamableServer.SendEventToSession(sid, payload)
			if err != nil {
				return mcpgo.NewToolResultError("第" + strconv.Itoa(i) + "次推送失败: " + err.Error()), nil
			}
			time.Sleep(2 * time.Second)
		}
		return mcpgo.NewToolResultText("5次推送已完成，每2秒推送一次"), nil
	}
	return
}
