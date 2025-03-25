package internal

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

// SSEServer implements a Server-Sent Events (SSE) based MCP server.
// It provides real-time communication capabilities over HTTP using the SSE protocol.
type SSEServer struct {
	server          *MCPServer
	baseURL         string
	messageEndpoint string
	sseEndpoint     string
	sessions        map[string]bool
	redisClient     *RedisClient // Redis client for pub/sub
}

func (s *SSEServer) GetMessageEndpoint() string {
	return s.messageEndpoint
}

func (s *SSEServer) GetSSEEndpoint() string {
	return s.sseEndpoint
}

func (s *SSEServer) GetServerName() string {
	return s.server.name
}

// Option defines a function type for configuring SSEServer
type Option func(*SSEServer)

// WithBaseURL sets the base URL for the SSE server
func WithBaseURL(baseURL string) Option {
	return func(s *SSEServer) {
		s.baseURL = baseURL
	}
}

// WithMessageEndpoint sets the message endpoint path
func WithMessageEndpoint(endpoint string) Option {
	return func(s *SSEServer) {
		s.messageEndpoint = endpoint
	}
}

// WithSSEEndpoint sets the SSE endpoint path
func WithSSEEndpoint(endpoint string) Option {
	return func(s *SSEServer) {
		s.sseEndpoint = endpoint
	}
}

func WithRedisClient(redisClient *RedisClient) Option {
	return func(s *SSEServer) {
		s.redisClient = redisClient
	}
}

// NewSSEServer creates a new SSE server instance with the given MCP server and options.
func NewSSEServer(server *MCPServer, opts ...Option) *SSEServer {
	s := &SSEServer{
		server:          server,
		sseEndpoint:     "/sse",
		messageEndpoint: "/message",
		sessions:        make(map[string]bool),
	}

	// Apply all options
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// handleSSE handles incoming SSE connection requests.
// It sets up appropriate headers and creates a new session for the client.
func (s *SSEServer) HandleSSE(cb api.FilterCallbackHandler) {
	sessionID := uuid.New().String()

	s.sessions[sessionID] = true

	// sessionStore, _ := json.Marshal(s.sessions)
	// TODO: sse:sessions?
	// s.redisClient.Set("sse:sessions", string(sessionStore), 0)
	defer delete(s.sessions, sessionID)

	channel := fmt.Sprintf("sse:%s", sessionID)

	messageEndpoint := fmt.Sprintf(
		"%s%s?sessionId=%s",
		s.baseURL,
		s.messageEndpoint,
		sessionID,
	)

	// go func() {
	// 	for {
	// 		select {
	// 		case serverNotification := <-s.server.notifications:
	// 			// Only forward notifications meant for this session
	// 			if serverNotification.Context.SessionID == sessionID {
	// 				eventData, err := json.Marshal(serverNotification.Notification)
	// 				if err == nil {
	// 					select {
	// 					case session.eventQueue <- fmt.Sprintf("event: message\ndata: %s\n\n", eventData):
	// 						// Event queued successfully
	// 					case <-session.done:
	// 						return
	// 					}
	// 				}
	// 			}
	// 		case <-session.done:
	// 			return
	// 		case <-r.Context().Done():
	// 			return
	// 		}
	// 	}
	// }()

	err := s.redisClient.Subscribe(channel, func(message string) {
		defer cb.EncoderFilterCallbacks().RecoverPanic()
		api.LogInfof("SSE Send message: %s", message)
		cb.EncoderFilterCallbacks().InjectData([]byte(message))
	})
	if err != nil {
		api.LogErrorf("Failed to subscribe to Redis channel: %v", err)
	}

	// Send the initial endpoint event
	initialEvent := fmt.Sprintf("event: endpoint\ndata: %s\r\n\r\n", messageEndpoint)
	s.redisClient.Publish(channel, initialEvent)
}

// handleMessage processes incoming JSON-RPC messages from clients and sends responses
// back through both the SSE connection and HTTP response.
func (s *SSEServer) HandleMessage(w http.ResponseWriter, r *http.Request, body json.RawMessage) {
	if r.Method != http.MethodPost {
		s.writeJSONRPCError(w, nil, mcp.INVALID_REQUEST, fmt.Sprintf("Method %s not allowed", r.Method))
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		s.writeJSONRPCError(w, nil, mcp.INVALID_PARAMS, "Missing sessionId")
		return
	}

	// Set the client context in the server before handling the message
	ctx := s.server.WithContext(r.Context(), NotificationContext{
		ClientID:  sessionID,
		SessionID: sessionID,
	})

	//TODO： check session id
	// _, ok := s.sessions.Load(sessionID)
	// if !ok {
	// 	s.writeJSONRPCError(w, nil, mcp.INVALID_PARAMS, "Invalid session ID")
	// 	return
	// }

	// Process message through MCPServer
	response := s.server.HandleMessage(ctx, body)

	// Only send response if there is one (not for notifications)
	if response != nil {
		eventData, _ := json.Marshal(response)

		if sessionID != "" {
			channel := fmt.Sprintf("sse:%s", sessionID)
			publishErr := s.redisClient.Publish(channel, fmt.Sprintf("event: message\ndata: %s\n\n", eventData))

			if publishErr != nil {
				api.LogErrorf("Failed to publish message to Redis: %v", publishErr)
			}
		}
		// Send HTTP response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)
	} else {
		// For notifications, just send 202 Accepted with no body
		w.WriteHeader(http.StatusAccepted)
	}
}

// writeJSONRPCError writes a JSON-RPC error response with the given error details.
func (s *SSEServer) writeJSONRPCError(
	w http.ResponseWriter,
	id interface{},
	code int,
	message string,
) {
	response := createErrorResponse(id, code, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(response)
}
