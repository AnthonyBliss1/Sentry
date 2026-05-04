package network

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	utils "github.com/anthonybliss1/Sentry/Node/Utils"
)

// LiveKit Server credentials received from API

type Concierge struct {
	RTSPPublishBase string `json:"rtsp_publish_base"`
	WebRTCBase      string `json:"webrtc_base"`
	HLSBase         string `json:"hls_base"`

	roomServiceURL string // not exportable
}

type Stream struct {
	rpiCmd    *exec.Cmd
	ffmpegCmd *exec.Cmd
}

func (s *Stream) Stop() error {
	var firstErr error

	if s.rpiCmd != nil && s.rpiCmd.Process != nil {
		if err := s.rpiCmd.Process.Kill(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.ffmpegCmd != nil && s.ffmpegCmd.Process != nil {
		if err := s.ffmpegCmd.Process.Kill(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if s.rpiCmd != nil {
		s.rpiCmd.Process.Wait()
	}
	if s.ffmpegCmd != nil {
		s.ffmpegCmd.Process.Wait()
	}

	return firstErr
}

func (c *Concierge) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")

	return string(b)
}

func (c *Concierge) PublishStream(deviceID string) (Stream, error) {
	if c.RTSPPublishBase == "" {
		return Stream{}, fmt.Errorf("rtsp publish base is empty")
	}

	publishURL := fmt.Sprintf("%s/%s", c.RTSPPublishBase, deviceID)

	rpiCmd := exec.Command(
		"rpicam-vid",
		"-t", "0",
		"--nopreview",
		"--width", "1280",
		"--height", "720",
		"--framerate", "15",
		"--bitrate", "1200000",
		"--codec", "h264",
		"--profile", "baseline",
		"--level", "4",
		"--intra", "15",
		"--inline",
		"--flush",
		"-o", "-",
	)

	rpiStdout, err := rpiCmd.StdoutPipe()
	if err != nil {
		return Stream{}, fmt.Errorf("failed to create rpicam stdout pipe: %w", err)
	}
	rpiCmd.Stderr = io.Discard

	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "warning",
		"-fflags", "nobuffer",
		"-f", "h264",
		"-i", "pipe:0",
		"-an",
		"-c:v", "copy",
		"-f", "rtsp",
		"-rtsp_transport", "tcp",
		"-pkt_size", "1200",
		publishURL,
	)
	ffmpegCmd.Stdin = rpiStdout
	ffmpegCmd.Stdout = io.Discard
	ffmpegCmd.Stderr = io.Discard

	if err := rpiCmd.Start(); err != nil {
		return Stream{}, fmt.Errorf("failed to start rpicam-vid: %w", err)
	}

	if err := ffmpegCmd.Start(); err != nil {
		rpiCmd.Process.Kill()
		rpiCmd.Wait()
		return Stream{}, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	utils.Green.Printf("[ Publishing -> %s ]\n", publishURL)

	return Stream{
		rpiCmd:    rpiCmd,
		ffmpegCmd: ffmpegCmd,
	}, nil
}
