package supervisor

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func (s *Supervisor) Run() {
	socketPath := SocketPath()
	crashCount := 0
	resetWindow := time.NewTicker(5 * time.Minute)

	for {
		select {
		case <-resetWindow.C:
			crashCount = 0 // Reset crash counter periodically
		default:
		}

		// Launch child process
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", socketEnv, socketPath))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Start(); err != nil {
			panic(err)
		}

		exitCode := 0

		// Wait for child to exit
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()

					// Check if killed by signal (segfault, etc.)
					if status.Signaled() {
						crashCount++
						slog.Error("Child killed by signal", "signal", status.Signal())
					}
				}
			}
		}

		if exitCode == 0 {
			os.Exit(0)
		}

		crashCount++
		slog.Error("Child exited", "code", exitCode, "crashCount", crashCount)

		if crashCount >= 3 {
			slog.Error("Too many crashes, initiating rollback")

			if err := s.Rollback(); err != nil {
				slog.Error("Rollback failed", "error", err)
			}

			crashCount = 0
		}

		time.Sleep(1 * time.Second)
	}
}

func IsSupervisorProcess() bool {
	return os.Getenv(socketEnv) == ""
}

func SocketPath() string {
	if IsSupervisorProcess() {
		return fmt.Sprintf("/tmp/knockknock-%d.sock", os.Getpid())
	}

	return os.Getenv(socketEnv)
}
