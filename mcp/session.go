package mcp

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/relaxyabc/k8s-helper/common"
	"github.com/relaxyabc/k8s-helper/crypto"
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

// NewHTTPSessionManager 创建 Session 管理器，expireTime 为 session 过期时长
func NewHTTPSessionManager(expireTime time.Duration) *HTTPSessionManager {
	sm := &HTTPSessionManager{
		sessions:   make(map[string]*HTTPSession),
		cleanup:    time.NewTicker(1 * time.Minute),
		expireTime: expireTime,
	}
	go sm.cleanupExpiredSessions()
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
	sessionUserInfoMap[session.ID] = struct{ UserID, Role string }{userID, role}
	return session
}

// GetSession 获取 session 并延长有效期
func (sm *HTTPSessionManager) GetSession(sessionID string) (*HTTPSession, bool) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists || time.Now().After(session.ExpiresAt) {
		if exists {
			delete(sm.sessions, sessionID)
			delete(sessionUserInfoMap, sessionID) // 同步删除映射
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
func (sm *HTTPSessionManager) DeleteSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.sessions, sessionID)
	delete(sessionUserInfoMap, sessionID) // 同步删除映射
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
	sessionUserInfoMap[session.ID] = struct{ UserID, Role string }{session.UserID, role}
}

// cleanupExpiredSessions 定时清理过期 session
func (sm *HTTPSessionManager) cleanupExpiredSessions() {
	for range sm.cleanup.C {
		now := time.Now()
		sm.mutex.Lock()
		for id, session := range sm.sessions {
			if now.After(session.ExpiresAt) {
				delete(sm.sessions, id)
				delete(sessionUserInfoMap, id) // 同步删除映射
			}
		}
		sm.mutex.Unlock()
	}
}

func SessionMiddleware(sm *HTTPSessionManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[SESSION] RequestURI: %s, Header.Mcp-Session-Id: %s", r.RequestURI, r.Header.Get(common.HeaderMcpSessionId))
		sid := r.Header.Get(common.HeaderMcpSessionId)
		mcpId := r.URL.Query().Get(common.McpIDParam)
		var ses *HTTPSession
		var userId, userRole string
		if mcpId != "" {
			mcpId = strings.TrimSpace(mcpId)
			var err error
			mcpId, err = url.QueryUnescape(mcpId)
			if err != nil {
				log.Printf("error unescaping mcpId: %v", err)
			}
			mcpId = strings.ReplaceAll(mcpId, " ", "+")
			userId, userRole = ParseUserIDAndRoleFromSID(mcpId)
			log.Printf("[SESSION] Parsed mcpId from url: userId=%s, userRole=%s", userId, userRole)
		}
		if sid != "" {
			if s, ok := sm.GetSession(sid); ok {
				ses = s
				// 如果 session 没有 role 且 mcpId 可解析，则补充 role
				if ses != nil && userRole != "" {
					if v, ok := ses.Data["role"]; !ok || v == "" {
						ses.Data["role"] = userRole
						// 同步更新 sessionUserInfoMap
						sessionUserInfoMapMutex.Lock()
						sessionUserInfoMap[ses.ID] = struct{ UserID, Role string }{ses.UserID, userRole}
						sessionUserInfoMapMutex.Unlock()
					}
				}
			}
		}
		// 如果 session 还不存在，且 mcpId 可解析，则用 mcpId 结果新建 session
		if ses == nil && userId != "" {
			ses = sm.CreateSession(userId)
			ses.Data["role"] = userRole
			w.Header().Set(common.HeaderMcpSessionId, ses.ID)
			// 同步更新 sessionUserInfoMap
			sessionUserInfoMapMutex.Lock()
			sessionUserInfoMap[ses.ID] = struct{ UserID, Role string }{userId, userRole}
			sessionUserInfoMapMutex.Unlock()
		}
		if ses == nil {
			ses = sm.CreateSession("")
			w.Header().Set(common.HeaderMcpSessionId, ses.ID)
		}
		if ses == nil || time.Now().After(ses.ExpiresAt) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("session expired or invalid"))
			return
		}
		// 直接传递原始 context
		next.ServeHTTP(w, r)
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
	var role string
	sessionUserInfoMapMutex.RLock()
	if info, ok := sessionUserInfoMap[sessionID]; ok {
		role = info.Role
	}
	sessionUserInfoMapMutex.RUnlock()
	log.Printf("[SESSION] GetUserRoleBySessionID: sessionID=%s, role=%s", sessionID, role)
	return role
}
