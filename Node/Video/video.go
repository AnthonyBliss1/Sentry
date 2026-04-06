package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// private helper function
func validateOutputDir() error {
	ex, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find binary path: %w", err)
	}

	cwd := filepath.Dir(ex)
	hlsDir := filepath.Join(cwd, "HLS")

	// Ensure HLS Dir exists
	if err := os.MkdirAll(hlsDir, 0o755); err != nil {
		return fmt.Errorf("failed to make HLS Dir: %w", err)
	}

	return nil
}

func StartStream() (*Stream, error) {
	if err := validateOutputDir(); err != nil {
		return nil, err
	}

	cmd := exec.Command(
		"ffmpeg",
		"-f", "v4l2",
		"-framerate", "30",
		"-video_size", "640x480",
		"-input_format", "mjpeg",
		"-i", "/dev/video0",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-pix_fmt", "yuv420p",
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "6",
		"-hls_flags", "delete_segments",
		"-hls_segment_filename", "HLS/segment_%03d.ts",
		"HLS/stream.m3u8",
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	stream := Stream{cmd: cmd}

	return &stream, nil
}
