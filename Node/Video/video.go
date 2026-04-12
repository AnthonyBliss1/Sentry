package video

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type Stream struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stopped bool
}

func (s *Stream) Stop() error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}

	cmd := s.cmd

	// reset stream fields
	s.stopped = true
	s.cmd = nil
	s.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}

	if cmd != nil {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("ffmpeg terminated: %w", err)
		}
	}

	return nil
}

func (s *Stream) Start(hubAddr string) error {
	// found this to be the best balance of quality and speed
	// may toss --low-latency in there if necessary
	cmd := exec.Command(
		"rpicam-vid",
		"-t", "0",
		"--width", "640",
		"--height", "480",
		"--framerate", "15",
		"--nopreview",
		"--inline",
		"--codec", "h264",
		"--intra", "30",
		"-o", hubAddr,
	)

	// need to output all these to log files but
	// should be careful of the log build up
	// for now just getting them out of the way
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	return nil
}
