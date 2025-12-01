package socketio

import (
	"sync"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	socket "github.com/zishang520/socket.io/clients/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

type Service struct {
	client         *socket.Socket
	organizationId string
	oauthService   oauth.OAuthServicer
	mu             sync.Mutex
	connected      bool
}

func NewService() *Service {
	return &Service{
		oauthService: oauth.DefaultOAuthService(),
	}
}

func (s *Service) Connect(orgId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected && s.organizationId == orgId {
		return nil
	}

	cfg, err := config.RestoreConfig()
	if err != nil {
		return errors.Wrap(err, "Failed to load config")
	}

	opts := socket.DefaultOptions()
	opts.SetPath("/socket")
	opts.SetTransports(types.NewSet(socket.WebSocket))
	opts.SetReconnection(true)
	opts.SetReconnectionAttempts(5)

	auth := map[string]any{
		"organizationId": orgId,
	}

	if cfg.HyphenAPIKey != nil {
		auth["apiKey"] = *cfg.HyphenAPIKey
	} else {
		token, err := s.oauthService.GetValidToken()
		if err != nil {
			return errors.Wrap(err, "Failed to get valid token")
		}
		auth["token"] = token
	}
	opts.SetAuth(auth)

	baseUrl := apiconf.GetIOBaseUrl()
	client, err := socket.Connect(baseUrl, opts)
	if err != nil {
		return errors.Wrap(err, "Failed to connect to Socket.io server")
	}

	s.client = client
	s.organizationId = orgId
	s.connected = true

	return nil
}

func (s *Service) On(event string, handler func(...any)) {
	if s.client != nil {
		s.client.On(types.EventName(event), handler)
	}
}

func (s *Service) Emit(event string, data ...any) {
	if s.client != nil {
		s.client.Emit(event, data...)
	}
}

func (s *Service) Disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		s.client.Disconnect()
		s.client = nil
		s.connected = false
	}
}

func (s *Service) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected
}
