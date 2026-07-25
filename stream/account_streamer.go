package stream

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// TokenFunc returns a valid tastytrade access token (refreshing as needed).
type TokenFunc func(ctx context.Context) (string, error)

// AccountStreamStatus is a point-in-time snapshot of the streamer's health.
type AccountStreamStatus struct {
	Enabled         bool      `json:"enabled"`
	Connected       bool      `json:"connected"`
	SessionID       string    `json:"session_id,omitempty"`
	Accounts        []string  `json:"accounts"`
	LastConnectedAt time.Time `json:"last_connected_at,omitempty"`
	LastHeartbeatAt time.Time `json:"last_heartbeat_at,omitempty"`
	LastError       string    `json:"last_error,omitempty"`
	Notifications   int       `json:"notifications"`
	Orders          int       `json:"cached_orders"`
	Balances        int       `json:"cached_balances"`
	Positions       int       `json:"cached_positions"`
}

const (
	heartbeatInterval = 30 * time.Second
	reconnectMin      = 1 * time.Second
	reconnectMax      = 30 * time.Second
)

// AccountStreamer maintains a one-directional websocket to the tastytrade account
// streamer, sending heartbeats and routing notifications into an in-memory cache.
type AccountStreamer struct {
	url      string
	accounts []string
	token    TokenFunc
	dialer   *websocket.Dialer

	reqID int64

	mu        sync.Mutex
	cache     *AcctCache
	status    AccountStreamStatus
	writeConn *websocket.Conn
	writeMu   sync.Mutex
}

// NewAccountStreamer constructs (but does not start) the streamer.
func NewAccountStreamer(url string, accounts []string, token TokenFunc) *AccountStreamer {
	return &AccountStreamer{
		url:      url,
		accounts: accounts,
		token:    token,
		dialer:   websocket.DefaultDialer,
		cache:    NewAcctCache(),
		status:   AccountStreamStatus{Enabled: true, Accounts: accounts},
	}
}

// Start launches the connect/read loop in the background. It returns immediately;
// the loop runs until ctx is cancelled.
func (s *AccountStreamer) Start(ctx context.Context) {
	go s.loop(ctx)
}

// Status returns a copy of the current status (with live cache counts).
func (s *AccountStreamer) Status() AccountStreamStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status
	st.Orders = len(s.cache.Orders)
	st.Balances = len(s.cache.Balances)
	st.Positions = len(s.cache.Positions)
	return st
}

func (s *AccountStreamer) loop(ctx context.Context) {
	backoff := reconnectMin
	for ctx.Err() == nil {
		err := s.session(ctx)
		s.mu.Lock()
		s.status.Connected = false
		if err != nil && ctx.Err() == nil {
			s.status.LastError = err.Error()
		}
		s.mu.Unlock()
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > reconnectMax {
			backoff = reconnectMax
		}
	}
}

// session runs one full connection lifecycle, returning when it drops.
func (s *AccountStreamer) session(ctx context.Context) error {
	token, err := s.token(ctx)
	if err != nil {
		return err
	}
	conn, _, err := s.dialer.DialContext(ctx, s.url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	s.mu.Lock()
	s.writeConn = conn
	s.mu.Unlock()

	// Close the conn when ctx is cancelled so the read loop unblocks.
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	if err := s.write(conn, connectFrame(s.accounts, token, s.nextReqID())); err != nil {
		return err
	}

	hbCtx, stopHB := context.WithCancel(ctx)
	defer stopHB()
	go s.heartbeatLoop(hbCtx, conn)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		s.handleMessage(msg)
	}
}

func (s *AccountStreamer) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	t := time.NewTicker(heartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			token, err := s.token(ctx)
			if err != nil {
				return
			}
			if err := s.write(conn, heartbeatFrame(token, s.nextReqID())); err != nil {
				return
			}
		}
	}
}

func (s *AccountStreamer) handleMessage(msg []byte) {
	var meta struct {
		Status    string `json:"status"`
		Action    string `json:"action"`
		SessionID string `json:"web-socket-session-id"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal(msg, &meta); err != nil {
		return
	}
	if meta.Action != "" { // control response (connect / heartbeat ack)
		s.mu.Lock()
		if meta.SessionID != "" {
			s.status.SessionID = meta.SessionID
		}
		switch meta.Action {
		case "connect":
			s.status.Connected = true
			s.status.LastConnectedAt = time.Now()
			s.status.LastError = ""
		case "heartbeat":
			s.status.LastHeartbeatAt = time.Now()
		}
		s.mu.Unlock()
		return
	}
	if meta.Type != "" { // notification
		s.mu.Lock()
		_, _ = s.cache.route(msg)
		s.status.Notifications++
		s.mu.Unlock()
	}
}

func (s *AccountStreamer) write(conn *websocket.Conn, v any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return conn.WriteJSON(v)
}

func (s *AccountStreamer) nextReqID() int64 {
	return atomic.AddInt64(&s.reqID, 1)
}
