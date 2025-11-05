package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/zeitlos/knockknock/supervisor"
)

type Server struct {
	listener   net.Listener
	socketPath string
	supervisor *supervisor.Supervisor
}

type VersionsResponse struct {
	Update   *semver.Version  `json:"update"`
	Current  semver.Version   `json:"current"`
	Versions []semver.Version `json:"versions"`
}

type UpdateRequest struct {
	Version string `json:"version"`
}

type UpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type RollbackResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type HistoryResponse struct {
	History []HistoryEntry `json:"history"`
}

type HistoryEntry struct {
	Version       semver.Version `json:"version"`
	LastInstalled time.Time      `json:"last_installed"`
}

func NewIPCServer(sv *supervisor.Supervisor) (*Server, error) {
	socketPath := supervisor.SocketPath()

	// Clean up old socket if exists
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)

	if err != nil {
		return nil, fmt.Errorf("failed to create unix socket: %w", err)
	}

	server := Server{
		listener:   listener,
		socketPath: socketPath,
		supervisor: sv,
	}

	return &server, nil
}

func (s *Server) Serve() {
	mux := http.NewServeMux()

	mux.HandleFunc("/versions", s.handleVersions)
	mux.HandleFunc("/update", s.handleUpdate)
	mux.HandleFunc("/rollback", s.handleRollback)
	mux.HandleFunc("/history", s.handleHistory)

	go func() {
		if err := http.Serve(s.listener, mux); err != nil {
			slog.Error("IPC server error", "error", err)
		}
	}()
}

func (s *Server) Close() error {
	if s.listener != nil {
		s.listener.Close()
	}
	os.Remove(s.socketPath)
	return nil
}

func (s *Server) handleVersions(w http.ResponseWriter, r *http.Request) {
	update, versions, err := s.supervisor.CheckForUpdate(r.Context())

	if err != nil {
		slog.Error("failed to fetch versions", "error", err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := VersionsResponse{
		Update:   update,
		Current:  *s.supervisor.CurrentVersion(),
		Versions: versions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Version == "" {
		http.Error(w, "Version is required", http.StatusBadRequest)
		return
	}

	slog.Info("Updating to version", "version", req.Version)

	// Start update in background - this will kill the process
	go func() {
		if err := s.supervisor.Update(context.Background(), req.Version); err != nil {
			slog.Error("Update failed", "error", err, "version", req.Version)
		}
	}()

	// Return success immediately before process is killed
	response := UpdateResponse{
		Success: true,
		Message: fmt.Sprintf("Update to version %s initiated, process will restart", req.Version),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slog.Info("Initiating rollback")

	// Start rollback in background - this will kill the process
	go func() {
		if err := s.supervisor.Rollback(); err != nil {
			slog.Error("Rollback failed", "error", err)
		}
	}()

	// Return success immediately before process is killed
	response := RollbackResponse{
		Success: true,
		Message: "Rollback initiated, process will restart",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	history := s.supervisor.History()

	resp := HistoryResponse{
		History: make([]HistoryEntry, len(history)),
	}

	for i, h := range history {
		resp.History[i] = HistoryEntry{
			Version:       h.Version,
			LastInstalled: h.LastInstalled,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
