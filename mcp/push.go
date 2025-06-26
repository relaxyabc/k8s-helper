package mcp

import (
	"context"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// StartPushNotifications starts a goroutine that periodically sends notifications to the client.
func StartPushNotifications(ctx context.Context, mcpServer *server.MCPServer) {
	sid, _ := ctx.Value("mcp-session").(string)

	go func() {
		for i := 0; i < 5; i++ {
			// Check if the client is still connected by checking the context's Done channel.
			select {
			case <-ctx.Done():
				log.Printf("[MCP-SSE] Client disconnected, stopping push for session: %s", sid)
				return
			default:
				// continue execution
			}

			time.Sleep(3 * time.Second)

			payload := map[string]interface{}{
				"message":   "This is a push from the server.",
				"timestamp": time.Now().Format(time.RFC3339),
				"count":     i + 1,
			}

			log.Printf("[MCP-SSE] Pushing message to session: %s, payload: %v", sid, payload)
			err := mcpServer.SendNotificationToClient(ctx, "server_push", payload)
			if err != nil {
				log.Printf("[MCP-SSE] Failed to send notification to session %s: %v", sid, err)
				// If sending fails, the client might have disconnected, so we stop.
				return
			}
		}
		log.Printf("[MCP-SSE] Finished pushing messages for session: %s", sid)
	}()
}
