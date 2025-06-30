package mcp

import (
	"time"

	"k8s.io/klog/v2"
)

// StartPushNotifications starts a goroutine that periodically sends notifications to the client via SSE.
func StartPushNotifications(sid string, sseServer *SSEServer) {
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(3 * time.Second)

			payload := map[string]interface{}{
				"message":   "This is a push from the server.",
				"timestamp": time.Now().Format(time.RFC3339),
				"count":     i + 1,
			}

			klog.Infof("[MCP-SSE-PUSH] Pushing message to session: %s", sid)
			err := sseServer.SendEventToSession(sid, payload)
			if err != nil {
				klog.Errorf("[MCP-SSE-PUSH] Failed to send notification to session %s: %v. Stopping push.", sid, err)
				return
			}
		}
		klog.Infof("[MCP-SSE-PUSH] Finished pushing messages for session: %s", sid)
	}()
}
