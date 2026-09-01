package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/UNSAReport/tui/internal/config"
)

type CallbackResult struct {
	PAT   string
	Token string
	Code  string
	State string
	Err   error
}

type CallbackServer struct {
	Listener net.Listener
	State    string
	Done     chan CallbackResult
	server   *http.Server
}

func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "st_" + hex.EncodeToString(b), nil
}

func NewCallbackServer(state string) *CallbackServer {
	return &CallbackServer{State: state, Done: make(chan CallbackResult, 1)}
}

func (s *CallbackServer) Start() (string, error) {
	ln, err := net.Listen("tcp", config.DefaultCallbackHost)
	if err != nil {
		return "", err
	}
	s.Listener = ln
	mux := http.NewServeMux()
	mux.HandleFunc(config.CallbackPath, s.handleCallback)
	mux.HandleFunc("/", s.handleCallback)
	s.server = &http.Server{Handler: mux, ReadHeaderTimeout: config.CallbackReadHeaderTimeout}
	go func() { _ = s.server.Serve(ln) }()
	addr := ln.Addr().String()
	u := config.CallbackBaseURLPrefix + addr + config.CallbackPath
	return u, nil
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	state := q.Get("state")
	if s.State != "" && state != s.State {
		http.Error(w, "invalid state", http.StatusBadRequest)
		select {
		case s.Done <- CallbackResult{Err: fmt.Errorf("invalid state: got %q want %q", state, s.State)}:
		default:
			fmt.Fprintf(os.Stderr, "callback channel full\n")
		}
		return
	}
	res := CallbackResult{
		PAT:   q.Get("pat"),
		State: state,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html><body><script>window.close()</script><p>You can close this window. Return to the terminal.</p></body></html>`))
	select {
	case s.Done <- res:
	default:
		fmt.Fprintf(os.Stderr, "callback channel full\n")
	}
}

func (s *CallbackServer) Close() {
	if s.server != nil {
		_ = s.server.Close()
	}
	if s.Listener != nil {
		_ = s.Listener.Close()
	}
}

func (s *CallbackServer) Wait(timeout time.Duration) (CallbackResult, error) {
	select {
	case r := <-s.Done:
		return r, r.Err
	case <-time.After(timeout):
		return CallbackResult{}, fmt.Errorf("callback timeout after %s", timeout)
	}
}

func IsLoopbackURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	return host == "127.0.0.1" || host == "localhost"
}
