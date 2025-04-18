package internal

var GlobalRegistry = NewServerRegistry()

type Server interface {
	ParseConfig(config map[string]any) error
	NewServer(serverName string) (*MCPServer, error)
}

type ServerRegistry struct {
	servers map[string]Server
}

func NewServerRegistry() *ServerRegistry {
	return &ServerRegistry{
		servers: make(map[string]Server),
	}
}

func (r *ServerRegistry) RegisterServer(name string, server Server) {
	r.servers[name] = server
}

func (r *ServerRegistry) GetServer(name string) Server {
	return r.servers[name]
}
