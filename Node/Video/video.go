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

func (s *Stream) Start(rtmpURL string) error {
	cmd := exec.Command(
		"rpicam-vid",
		"-t", "0",
		"--width", "1280",
		"--height", "720",
		"--framerate", "24",
		"--bitrate", "2200000",
		"--nopreview",
		"--inline",
		"--codec", "libav",
		"--libav-format", "flv",
		"-o", rtmpURL,
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
