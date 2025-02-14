package onebot

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{}

type Service struct {
	log zerolog.Logger

	endpoint string
	timeout  time.Duration

	server *http.Server

	clients     map[string]*Client
	clientsLock sync.RWMutex
}

func NewService(log zerolog.Logger, endpoint string, timeout time.Duration) *Service {
	service := &Service{
		log:      log.With().Str("service", "onebot").Logger(),
		endpoint: endpoint,
		timeout:  timeout,
		clients:  make(map[string]*Client),
	}
	service.server = &http.Server{
		Addr:    endpoint,
		Handler: service,
	}

	return service
}

func (s *Service) Start() {
	s.log.Info().Msgf("Service starting to listen on %s", s.endpoint)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.log.Fatal().Err(err).Msg("Failed to listen and serve")
	}
}

func (s *Service) Stop() {
	s.log.Info().Msg("Stopping service")

	// Close all clients
	for _, clients := range s.clients {
		clients.Release()
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if err != nil {
		s.log.Warn().Err(err).Msg("Failed to shutdown server")
	}
}

func (s *Service) NewClient(log zerolog.Logger, id, token string) *Client {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	client := NewClient(log, id, token, s)
	s.clients[token] = client

	return client
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
		return
	}

	token := authHeader[7:]
	s.clientsLock.RLock()
	client, ok := s.clients[token]
	s.clientsLock.RUnlock()
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	if client.id != "" && client.id != r.Header.Get("X-Self-ID") {
		http.Error(w, "The login ID does not match", http.StatusUnauthorized)
		return
	}

	agent := r.Header.Get("User-Agent")
	if strings.HasPrefix(agent, "LLOneBot") {
		client.agentType = AgentLLOneBot
	} else if strings.HasPrefix(agent, "WeChat") {
		client.agentType = AgentWeChat
	} else {
		client.agentType = AgentNapCat
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Warn().Err(err).Msg("Failed to upgrade websocket request")
		return
	}

	go client.StartLoop(conn)
}

func (s *Service) removeClient(id string) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	delete(s.clients, id)
}
