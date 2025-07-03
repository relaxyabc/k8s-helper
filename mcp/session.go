package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/relaxyabc/k8s-helper/common"
	"github.com/relaxyabc/k8s-helper/crypto"
	"k8s.io/klog/v2"
)

var AESKey = "k8s-mcp-client" // 默认 AES 加密 key

// 是否允许同一用户多 session，默认 false
var AllowMultiSession = false

// 优化版 HTTPSessionManager

type HTTPSessionManager struct {
	sessions   map[string]*HTTPSession
	mutex      sync.RWMutex
	cleanup    *time.Ticker
	expireTime time.Duration
}

type HTTPSession struct {
	ID         string
	UserID     string
	CreatedAt  time.Time
	LastAccess time.Time
	Data       map[string]interface{}
	ExpiresAt  time.Time
}

// sessionId -> 用户信息映射
var sessionUserInfoMap = make(map[string]struct{ UserID, Role string })
var sessionUserInfoMapMutex sync.RWMutex // 新增：全局读写锁

// 新增：公共函数来管理 sessionUserInfoMap

// addSessionUserInfo 添加或更新 session 用户信息映射
func addSessionUserInfo(sessionID, userID, role string) {
	sessionUserInfoMapMutex.Lock()
	defer sessionUserInfoMapMutex.Unlock()
	sessionUserInfoMap[sessionID] = struct{ UserID, Role string }{userID, role}
}

// removeSessionUserInfo 删除 session 用户信息映射
func removeSessionUserInfo(sessionID string) {
	sessionUserInfoMapMutex.Lock()
	defer sessionUserInfoMapMutex.Unlock()
	delete(sessionUserInfoMap, sessionID)
}

// getSessionUserInfo 获取 session 用户信息
func getSessionUserInfo(sessionID string) (string, string) {
	sessionUserInfoMapMutex.RLock()
	defer sessionUserInfoMapMutex.RUnlock()
	if info, ok := sessionUserInfoMap[sessionID]; ok {
		return info.UserID, info.Role
	}
	return "", ""
}

// updateSessionRole 更新 session 的角色信息
func updateSessionRole(sessionID, userID, role string) {
	if role != "" {
		addSessionUserInfo(sessionID, userID, role)
	}
}

// NewHTTPSessionManager 创建 Session 管理器，expireTime 为 session 过期时长
func NewHTTPSessionManager(expireTime time.Duration, mcpServer *MCPServer) *HTTPSessionManager {
	sm := &HTTPSessionManager{
		sessions:   make(map[string]*HTTPSession),
		cleanup:    time.NewTicker(1 * time.Minute),
		expireTime: expireTime,
	}
	go sm.cleanupExpiredSessions(mcpServer)
	return sm
}

// CreateSession 创建新 session
func (sm *HTTPSessionManager) CreateSession(userID string) *HTTPSession {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if !AllowMultiSession && userID != "" {
		for _, s := range sm.sessions {
			if s.UserID == userID {
				return s
			}
		}
	}

	now := time.Now()
	session := &HTTPSession{
		ID:         generateSessionID(),
		UserID:     userID,
		CreatedAt:  now,
		LastAccess: now,
		Data:       make(map[string]interface{}),
		ExpiresAt:  now.Add(sm.expireTime),
	}
	sm.sessions[session.ID] = session
	// 新增：维护 sessionId -> 用户信息映射
	role := ""
	if v, ok := session.Data["role"]; ok {
		if s, ok := v.(string); ok {
			role = s
		}
	}
	addSessionUserInfo(session.ID, userID, role)
	return session
}

// GetSession 获取 session 并延长有效期
func (sm *HTTPSessionManager) GetSession(sessionID string, mcpServer *MCPServer) (*HTTPSession, bool) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists || time.Now().After(session.ExpiresAt) {
		if exists {
			delete(sm.sessions, sessionID)
			removeSessionUserInfo(sessionID) // 同步删除映射
			// 同步调用 MCPServer 的 UnregisterSession 函数
			if mcpServer != nil {
				mcpServer.UnregisterSession(sessionID)
			}
		}
		return nil, false
	}
	// 更新访问时间和过期时间
	now := time.Now()
	session.LastAccess = now
	session.ExpiresAt = now.Add(sm.expireTime)
	return session, true
}

// DeleteSession 主动删除 session
func (sm *HTTPSessionManager) DeleteSession(sessionID string, mcpServer *MCPServer) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.sessions, sessionID)
	removeSessionUserInfo(sessionID) // 同步删除映射

	// 同步调用 MCPServer 的 UnregisterSession 函数
	if mcpServer != nil {
		mcpServer.UnregisterSession(sessionID)
	}
}

// AddSession 允许外部以指定 ID 添加 session
func (sm *HTTPSessionManager) AddSession(session *HTTPSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.sessions[session.ID] = session
	// 新增：维护 sessionId -> 用户信息映射
	role := ""
	if v, ok := session.Data["role"]; ok {
		if s, ok := v.(string); ok {
			role = s
		}
	}
	addSessionUserInfo(session.ID, session.UserID, role)
}

// cleanupExpiredSessions 定时清理过期 session
func (sm *HTTPSessionManager) cleanupExpiredSessions(mcpServer *MCPServer) {
	for range sm.cleanup.C {
		now := time.Now()
		sm.mutex.Lock()
		for id, session := range sm.sessions {
			if now.After(session.ExpiresAt) {
				delete(sm.sessions, id)
				removeSessionUserInfo(id) // 同步删除映射
				// 同步调用 MCPServer 的 UnregisterSession 函数
				if mcpServer != nil {
					mcpServer.UnregisterSession(id)
				}
			}
		}
		sm.mutex.Unlock()
	}
}

func SessionMiddleware(sm *HTTPSessionManager, mcpServer *MCPServer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		klog.Infof("[SESSION_TRACE] ===== START: RequestURI: %s =====", r.RequestURI)
		sid := r.Header.Get(common.HeaderMcpSessionId)
		klog.Infof("[SESSION_TRACE] 1. Got Mcp-Session-Id from header: '%s'", sid)

		mcpId := r.URL.Query().Get(common.McpIDParam)
		var ses *HTTPSession
		var userId, userRole string

		if mcpId != "" {
			klog.Infof("[SESSION_TRACE] 2. Found mcpId in URL: '%s'", mcpId)
			mcpId = strings.TrimSpace(mcpId)
			var err error
			mcpId, err = url.QueryUnescape(mcpId)
			if err != nil {
				klog.Infof("[SESSION_TRACE] 2a. ERROR unescaping mcpId: %v", err)
			}
			mcpId = strings.ReplaceAll(mcpId, " ", "+")
			userId, userRole = ParseUserIDAndRoleFromSID(mcpId)
			klog.Infof("[SESSION_TRACE] 2b. Parsed mcpId: userId=%s, userRole=%s", userId, userRole)
		} else {
			klog.Infof("[SESSION_TRACE] 2. mcpId not found in URL.")
		}

		if sid != "" {
			klog.Infof("[SESSION_TRACE] 3. Attempting to get session with sid: '%s'", sid)
			if s, ok := sm.GetSession(sid, mcpServer); ok {
				ses = s
				klog.Infof("[SESSION_TRACE] 3a. SUCCESS: Found active session: ID=%s, UserID=%s, ExpiresAt=%v", ses.ID, ses.UserID, ses.ExpiresAt)
				if userRole != "" {
					if v, ok := ses.Data["role"]; !ok || v == "" {
						ses.Data["role"] = userRole
						updateSessionRole(ses.ID, ses.UserID, userRole)
						klog.Infof("[SESSION_TRACE] 3b. UPDATED session role from mcpId: role=%s", userRole)
					}
				}
			} else {
				klog.Infof("[SESSION_TRACE] 3a. FAILED: No active session found for sid: '%s'", sid)
			}
		} else {
			klog.Infof("[SESSION_TRACE] 3. No Mcp-Session-Id in header, skipping session retrieval.")
		}

		if ses == nil && userId != "" {
			klog.Infof("[SESSION_TRACE] 4. Creating new session from mcpId: userId=%s, userRole=%s", userId, userRole)
			ses = sm.CreateSession(userId)
			ses.Data["role"] = userRole
			w.Header().Set(common.HeaderMcpSessionId, ses.ID)
			updateSessionRole(ses.ID, ses.UserID, userRole)
			klog.Infof("[SESSION_TRACE] 4a. SUCCESS: Created new session: ID=%s", ses.ID)
		}

		if ses == nil {
			klog.Infof("[SESSION_TRACE] 5. Creating new EMPTY session.")
			ses = sm.CreateSession("")
			w.Header().Set(common.HeaderMcpSessionId, ses.ID)
			klog.Infof("[SESSION_TRACE] 5a. SUCCESS: Created new empty session: ID=%s", ses.ID)
		}

		klog.Infof("[SESSION_TRACE] 6. Final validation check for session: ID=%s", ses.ID)
		if time.Now().After(ses.ExpiresAt) {
			klog.Infof("[SESSION_TRACE] 6a. FAILED: Session is expired. ExpiresAt=%v, Now=%v", ses.ExpiresAt, time.Now())
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("session expired or invalid"))
			klog.Infof("[SESSION_TRACE] ===== END: Responded with 401 Unauthorized =====")
			return
		}

		klog.Infof("[SESSION_TRACE] 6a. SUCCESS: Session is valid.")
		// Add session ID to context for downstream handlers
		ctx := context.WithValue(r.Context(), common.ContextKeyMcpSession, ses.ID)
		klog.Infof("[SESSION_TRACE] 7. Injecting sid '%s' into request context.", ses.ID)

		// 自动集成 streamable context map 管理
		SetSessionContext(ses.ID, ctx)

		// 保证每次请求都注册 session 到 MCPServer，初始化通知通道
		if mcpServer != nil && ses != nil {
			mcpServer.RegisterSession(ses.ID)
		}

		klog.Infof("[SESSION_TRACE] ===== END: Passing request to next handler =====")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// 通过 sid 解密出用户信息（如 userID 和 role）
func ParseUserIDAndRoleFromSID(sid string) (string, string) {
	plain, err := crypto.AESDecryptBase64(sid, AESKey)
	if err != nil {
		return "", ""
	}
	var obj struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := json.Unmarshal([]byte(plain), &obj); err != nil {
		return "", ""
	}
	return obj.Name, obj.Role
}

// 线程安全获取 sessionId 对应的用户角色
func GetUserRoleBySessionID(sessionID string) string {
	_, role := getSessionUserInfo(sessionID)
	klog.Infof("[SESSION] GetUserRoleBySessionID: sessionID=%s, role=%s", sessionID, role)
	return role
}

// 线程安全获取 sessionId 对应的用户ID
func GetUserIDBySessionID(sessionID string) string {
	userID, _ := getSessionUserInfo(sessionID)
	klog.Infof("[SESSION] GetUserIDBySessionID: sessionID=%s, userID=%s", sessionID, userID)
	return userID
}
