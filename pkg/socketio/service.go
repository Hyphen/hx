package socketio

import (
	"fmt"
	"sync"
	"time"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	socket "github.com/zishang520/socket.io/clients/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

type Service struct {
	client          *socket.Socket
	organizationId  string
	oauthService    oauth.OAuthServicer
	mu              sync.Mutex
	connected       bool
	connectedCh     chan struct{}
	verboseCallback func(string)
}

func NewService() *Service {
	return &Service{
		oauthService: oauth.DefaultOAuthService(),
		connectedCh:  make(chan struct{}),
	}
}

// SetVerboseCallback sets a callback function for verbose logging messages
func (s *Service) SetVerboseCallback(cb func(string)) {
	s.verboseCallback = cb
}

func (s *Service) logVerbose(msg string) {
	if s.verboseCallback != nil {
		s.verboseCallback(msg)
	} else if flags.VerboseFlag {
		fmt.Printf("[Socket.io] %s\n", msg)
	}
}

func (s *Service) Connect(orgId string) error {
	s.mu.Lock()

	if s.connected && s.organizationId == orgId {
		s.mu.Unlock()
		return nil
	}

	cfg, err := config.RestoreConfig()
	if err != nil {
		s.mu.Unlock()
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
			s.mu.Unlock()
			return errors.Wrap(err, "Failed to get valid token")
		}
		auth["token"] = token
	}
	opts.SetAuth(auth)

	baseUrl := apiconf.GetIOBaseUrl()
	s.logVerbose("Connecting to stream server")

	client, err := socket.Connect(baseUrl, opts)
	if err != nil {
		s.mu.Unlock()
		return errors.Wrap(err, "Failed to connect to stream server")
	}

	s.client = client
	s.organizationId = orgId

	client.On("connect", func(args ...any) {
		s.mu.Lock()
		// We track whether we were previously connected to detect reconnects
		wasConnected := s.connected
		s.connected = true
		s.mu.Unlock()

		// If this is a reconnect, the connected channel will already have been closed, and trying to close it again will panic
		if wasConnected {
			s.logVerbose("Reconnected successfully")
		} else {
			s.logVerbose("Connected successfully")
			close(s.connectedCh)
		}
	})

	client.On("connect_error", func(args ...any) {
		s.logVerbose(fmt.Sprintf("Connection error: %v", args))
	})

	client.On("disconnect", func(args ...any) {
		s.logVerbose(fmt.Sprintf("Disconnected: %v", args))
		s.mu.Lock()
		s.connected = false
		s.mu.Unlock()
	})

	s.mu.Unlock()

	select {
	case <-s.connectedCh:
		return nil
	case <-time.After(10 * time.Second):
		s.logVerbose("Connection timed out after 10 seconds")
		return errors.New("Socket.io connection timeout")
	}
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
