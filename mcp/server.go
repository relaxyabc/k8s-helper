package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/relaxyabc/k8s-helper/common"
	"github.com/relaxyabc/k8s-helper/dao"
	"github.com/relaxyabc/k8s-helper/tools"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	proxy     string
	transport string // 当前运行协议类型
)

func Init(proxyStr, aesKey, transportType string) {
	proxy = proxyStr
	AESKey = aesKey
	transport = transportType
}

func GetTransport() string {
	return transport
}

type MCPServer struct {
	server *server.MCPServer
}

func NewMCPServer() *MCPServer {
	mcpServer := server.NewMCPServer(
		"k8s-helper",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
		server.WithToolFilter(func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
			role := ""
			sid := ""
			if session := server.ClientSessionFromContext(ctx); session != nil {
				sid = session.SessionID()
				role = GetUserRoleBySessionID(sid)
			}
			var toolNames []string
			for _, t := range tools {
				toolNames = append(toolNames, t.Name)
			}
			log.Printf("[TOOL_FILTER] sid=%s, role=%s, all_tools=%v", sid, role, toolNames)
			if role == "admin" {
				return tools // admin 全部
			}
			var filtered []mcp.Tool
			for _, tool := range tools {
				if role == "user" {
					// user 仅允许 get_clusters、get_pods、get_deployments、get_daemonsets
					if tool.Name == "get_clusters" || tool.Name == "get_pods" || tool.Name == "get_deployments" || tool.Name == "get_daemonsets" {
						filtered = append(filtered, tool)
					}
				} else if role == "guest" {
					if tool.Name == "get_clusters" {
						filtered = append(filtered, tool)
					}
				}
			}
			log.Printf("[TOOL_FILTER] sid=%s, role=%s, filtered_tools=%v", sid, role, func() []string {
				names := []string{}
				for _, t := range filtered {
					names = append(names, t.Name)
				}
				return names
			}())
			return filtered
		}),
	)

	// 工具注册（全部 http tool 风格）
	registerHTTPTool := func(toolName, desc string, handler func(ctx context.Context, method, url, body string) (*mcp.CallToolResult, error)) {
		t := mcp.NewTool(toolName,
			mcp.WithDescription(desc),
			mcp.WithString("method", mcp.Required(), mcp.Description("HTTP method: GET/POST/PUT/DELETE"), mcp.Enum("GET", "POST", "PUT", "DELETE")),
			mcp.WithString("url", mcp.Required(), mcp.Description("API 路径，如 /clusters /namespaces?cluster_name=xxx 等")),
			mcp.WithString("body", mcp.Description("请求体（POST/PUT 时可选)")),
		)
		mcpServer.AddTool(t, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			method, _ := args["method"].(string)
			url, _ := args["url"].(string)
			body := ""
			if b, ok := args["body"].(string); ok {
				body = b
			}
			sid := ""
			if session := server.ClientSessionFromContext(ctx); session != nil {
				sid = session.SessionID()
			}
			paramsJson, _ := json.Marshal(args)
			fmt.Printf("[%s][%s][sessionid:%s]-%s-%s\n", time.Now().Format("2006-01-02 15:04:05"), transport, sid, toolName, string(paramsJson))
			return handler(ctx, method, url, body)
		})
	}

	// get_clusters
	registerHTTPTool("get_clusters", "Get all clusters from database (HTTP tool 风格)", func(ctx context.Context, method, url, body string) (*mcp.CallToolResult, error) {
		if method != "GET" || url != "/clusters" {
			return mcp.NewToolResultError("仅支持 GET /clusters"), nil
		}
		result, err := dao.GetClusterInfos()
		if err != nil {
			return mcp.NewToolResultError("查询数据库失败: " + err.Error()), nil
		}
		jsonStr, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError("序列化失败: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(jsonStr)), nil
	})
	// get_namespaces
	registerHTTPTool("get_namespaces", "Get namespaces list for a cluster (HTTP tool 风格)", func(ctx context.Context, method, url, body string) (*mcp.CallToolResult, error) {
		if method != "GET" || !strings.HasPrefix(url, "/namespaces") {
			return mcp.NewToolResultError("仅支持 GET /namespaces?cluster_name=xxx"), nil
		}
		q := url[strings.Index(url, "?")+1:]
		params := make(map[string]string)
		for _, kv := range strings.Split(q, "&") {
			if kv == "" {
				continue
			}
			arr := strings.SplitN(kv, "=", 2)
			if len(arr) == 2 {
				params[arr[0]] = arr[1]
			}
		}
		clusterName := params["cluster_name"]
		if clusterName == "" {
			return mcp.NewToolResultError("参数 cluster_name 必填"), nil
		}
		nsList, err := tools.GetNamespacesTool(proxy, clusterName)
		if err != nil {
			return mcp.NewToolResultError("获取 namespace 失败: " + err.Error()), nil
		}
		jsonStr, err := json.Marshal(nsList)
		if err != nil {
			return mcp.NewToolResultError("序列化失败: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(jsonStr)), nil
	})
	// get_pods
	registerHTTPTool("get_pods", "Get pods in a namespace for a cluster (HTTP tool 风格)", func(ctx context.Context, method, url, body string) (*mcp.CallToolResult, error) {
		if method != "GET" || !strings.HasPrefix(url, "/pods") {
			return mcp.NewToolResultError("仅支持 GET /pods?cluster_name=xxx&namespace=xxx"), nil
		}
		q := url[strings.Index(url, "?")+1:]
		params := make(map[string]string)
		for _, kv := range strings.Split(q, "&") {
			if kv == "" {
				continue
			}
			arr := strings.SplitN(kv, "=", 2)
			if len(arr) == 2 {
				params[arr[0]] = arr[1]
			}
		}
		clusterName := params["cluster_name"]
		namespace := params["namespace"]
		if clusterName == "" || namespace == "" {
			return mcp.NewToolResultError("参数 cluster_name 和 namespace 必填"), nil
		}
		pods, err := tools.GetPodsTool(proxy, clusterName, namespace)
		if err != nil {
			return mcp.NewToolResultError("获取 pods 失败: " + err.Error()), nil
		}
		jsonStr, err := json.Marshal(pods)
		if err != nil {
			return mcp.NewToolResultError("序列化失败: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(jsonStr)), nil
	})
	// get_deployments
	registerHTTPTool("get_deployments", "Get deployments in a namespace for a cluster (HTTP tool 风格)", func(ctx context.Context, method, url, body string) (*mcp.CallToolResult, error) {
		if method != "GET" || !strings.HasPrefix(url, "/deployments") {
			return mcp.NewToolResultError("仅支持 GET /deployments?cluster_name=xxx&namespace=xxx"), nil
		}
		q := url[strings.Index(url, "?")+1:]
		params := make(map[string]string)
		for _, kv := range strings.Split(q, "&") {
			if kv == "" {
				continue
			}
			arr := strings.SplitN(kv, "=", 2)
			if len(arr) == 2 {
				params[arr[0]] = arr[1]
			}
		}
		clusterName := params["cluster_name"]
		namespace := params["namespace"]
		if clusterName == "" || namespace == "" {
			return mcp.NewToolResultError("参数 cluster_name 和 namespace 必填"), nil
		}
		deployments, err := tools.GetDeploymentsTool(proxy, clusterName, namespace)
		if err != nil {
			return mcp.NewToolResultError("获取 deployments 失败: " + err.Error()), nil
		}
		jsonStr, err := json.Marshal(deployments)
		if err != nil {
			return mcp.NewToolResultError("序列化失败: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(jsonStr)), nil
	})
	// get_daemonsets
	registerHTTPTool("get_daemonsets", "Get daemonsets in a namespace for a cluster (HTTP tool 风格)", func(ctx context.Context, method, url, body string) (*mcp.CallToolResult, error) {
		if method != "GET" || !strings.HasPrefix(url, "/daemonsets") {
			return mcp.NewToolResultError("仅支持 GET /daemonsets?cluster_name=xxx&namespace=xxx"), nil
		}
		q := url[strings.Index(url, "?")+1:]
		params := make(map[string]string)
		for _, kv := range strings.Split(q, "&") {
			if kv == "" {
				continue
			}
			arr := strings.SplitN(kv, "=", 2)
			if len(arr) == 2 {
				params[arr[0]] = arr[1]
			}
		}
		clusterName := params["cluster_name"]
		namespace := params["namespace"]
		if clusterName == "" || namespace == "" {
			return mcp.NewToolResultError("参数 cluster_name 和 namespace 必填"), nil
		}
		daemonsets, err := tools.GetDaemonSetsTool(proxy, clusterName, namespace)
		if err != nil {
			return mcp.NewToolResultError("获取 daemonsets 失败: " + err.Error()), nil
		}
		jsonStr, err := json.Marshal(daemonsets)
		if err != nil {
			return mcp.NewToolResultError("序列化失败: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(jsonStr)), nil
	})
	// rollout_restart_deployment
	registerHTTPTool("rollout_restart_deployment", "滚动重启指定 Deployment (HTTP tool 风格)", func(ctx context.Context, method, url, body string) (*mcp.CallToolResult, error) {
		if method != "POST" || !strings.HasPrefix(url, "/rollout_restart_deployment") {
			return mcp.NewToolResultError("仅支持 POST /rollout_restart_deployment?cluster_name=xxx&namespace=xxx&name=xxx"), nil
		}
		q := url[strings.Index(url, "?")+1:]
		params := make(map[string]string)
		for _, kv := range strings.Split(q, "&") {
			if kv == "" {
				continue
			}
			arr := strings.SplitN(kv, "=", 2)
			if len(arr) == 2 {
				params[arr[0]] = arr[1]
			}
		}
		clusterName := params["cluster_name"]
		namespace := params["namespace"]
		name := params["name"]
		if clusterName == "" || namespace == "" || name == "" {
			return mcp.NewToolResultError("参数 cluster_name、namespace、name 必填"), nil
		}
		err := tools.RolloutRestartDeploymentTool(proxy, clusterName, namespace, name)
		if err != nil {
			return mcp.NewToolResultError("滚动重启 Deployment 失败: " + err.Error()), nil
		}
		return mcp.NewToolResultText("Deployment rollout restarted successfully."), nil
	})
	// rollout_restart_daemonset
	registerHTTPTool("rollout_restart_daemonset", "滚动重启指定 DaemonSet (HTTP tool 风格)", func(ctx context.Context, method, url, body string) (*mcp.CallToolResult, error) {
		if method != "POST" || !strings.HasPrefix(url, "/rollout_restart_daemonset") {
			return mcp.NewToolResultError("仅支持 POST /rollout_restart_daemonset?cluster_name=xxx&namespace=xxx&name=xxx"), nil
		}
		q := url[strings.Index(url, "?")+1:]
		params := make(map[string]string)
		for _, kv := range strings.Split(q, "&") {
			if kv == "" {
				continue
			}
			arr := strings.SplitN(kv, "=", 2)
			if len(arr) == 2 {
				params[arr[0]] = arr[1]
			}
		}
		clusterName := params["cluster_name"]
		namespace := params["namespace"]
		name := params["name"]
		if clusterName == "" || namespace == "" || name == "" {
			return mcp.NewToolResultError("参数 cluster_name、namespace、name 必填"), nil
		}
		err := tools.RolloutRestartDaemonSetTool(proxy, clusterName, namespace, name)
		if err != nil {
			return mcp.NewToolResultError("滚动重启 DaemonSet 失败: " + err.Error()), nil
		}
		return mcp.NewToolResultText("DaemonSet 滚动重启成功"), nil
	})
	// get_k8s_version
	registerHTTPTool("get_k8s_version", "Get k8s version for a cluster (HTTP tool 风格)", func(ctx context.Context, method, url, body string) (*mcp.CallToolResult, error) {
		if method != "GET" || !strings.HasPrefix(url, "/k8s_version") {
			return mcp.NewToolResultError("仅支持 GET /k8s_version?cluster_name=xxx"), nil
		}
		q := url[strings.Index(url, "?")+1:]
		params := make(map[string]string)
		for _, kv := range strings.Split(q, "&") {
			if kv == "" {
				continue
			}
			arr := strings.SplitN(kv, "=", 2)
			if len(arr) == 2 {
				params[arr[0]] = arr[1]
			}
		}
		clusterName := params["cluster_name"]
		if clusterName == "" {
			return mcp.NewToolResultError("参数 cluster_name 必填"), nil
		}
		version, err := tools.GetK8sVersionTool(proxy, clusterName)
		if err != nil {
			return mcp.NewToolResultError("获取 k8s 版本失败: " + err.Error()), nil
		}
		return mcp.NewToolResultText(version), nil
	})

	mcpServer.AddTool(
		mcp.NewTool("start_sse_push",
			mcp.WithDescription("Starts a background task that pushes notifications to the client via SSE."),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sid := ""
			if session := server.ClientSessionFromContext(ctx); session != nil {
				sid = session.SessionID()
			}
			log.Printf("[MCP-SSE] Tool 'start_sse_push' called for session: %s", sid)

			mcpServer := server.ServerFromContext(ctx)
			if mcpServer == nil {
				log.Printf("[MCP-SSE] Could not get server from context for session: %s", sid)
				return mcp.NewToolResultError("could not get server from context"), nil
			}

			StartPushNotifications(ctx, mcpServer)

			return mcp.NewToolResultText("SSE push notifications started. You will receive 5 messages over 15 seconds."), nil
		},
	)

	return &MCPServer{server: mcpServer}
}

func (s *MCPServer) ServeHTTP() *server.StreamableHTTPServer {
	return server.NewStreamableHTTPServer(s.server)
}

func (s *MCPServer) ServeStdio() error {
	return server.ServeStdio(s.server)
}

func generateSessionID() string {
	return common.SessionIdPrefix + uuid.NewString()
}
