package video

import (
	"fmt"
	"io"
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

func (s *Stream) Start(hlsDir string) error {
	segmentPattern := filepath.Join(hlsDir, "segment_%03d.ts")
	playlistPath := filepath.Join(hlsDir, "stream.m3u8")

	cmd := exec.Command(
		"ffmpeg",
		"-loglevel", "warning",
		"-f", "v4l2",
		"-framerate", "30",
		"-video_size", "640x480",
		"-input_format", "mjpeg",
		"-i", "/dev/video0",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-pix_fmt", "yuv420p",
		"-g", "60",
		"-keyint_min", "60",
		"-sc_threshold", "0",
		"-force_key_frames", "expr:gte(t,n_forced*2)",
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "6",
		"-hls_flags", "delete_segments+temp_file+omit_endlist",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	)

	// need to output all these to log files but
	// should be careful of the log build up
	// for now just getting them out of the way
	cmd.Stderr = io.Discard
	cmd.Stdout = io.Discard

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	return nil
}
