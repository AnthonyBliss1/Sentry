package network

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	utils "github.com/anthonybliss1/Sentry/Node/Utils"
)

// MediaMTX Server credentials received from API

type Concierge struct {
	RTSPPublishBase string `json:"rtsp_publish_base"`
	WebRTCBase      string `json:"webrtc_base"`
	HLSBase         string `json:"hls_base"`

	roomServiceURL string // not exportable

	Stream
}

type Stream struct {
	rpiCmd    *exec.Cmd
	ffmpegCmd *exec.Cmd
	isRunning bool
}

func (c *Concierge) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")

	return string(b)
}

func (c *Concierge) StreamController(action <-chan Message) error {
	deviceID, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to collect hostname: %w", err)
	}

	for msg := range action {
		if msg.Action == "Stop" {
			utils.Blue.Println("> Stopping stream...")
			c.Stop()
		}
		if msg.Action == "Start" {
			utils.Blue.Println("> Starting stream...")
			go c.PublishStream(deviceID)
		}
	}

	return nil
}

func (c *Concierge) Stop() error {
	var firstErr error

	if c.rpiCmd != nil && c.rpiCmd.Process != nil {
		if err := c.rpiCmd.Process.Kill(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.ffmpegCmd != nil && c.ffmpegCmd.Process != nil {
		if err := c.ffmpegCmd.Process.Kill(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if c.rpiCmd != nil {
		c.rpiCmd.Process.Wait()
	}
	if c.ffmpegCmd != nil {
		c.ffmpegCmd.Process.Wait()
	}

	c.isRunning = false

	return firstErr
}

func (c *Concierge) PublishStream(deviceID string) error {
	if c.RTSPPublishBase == "" {
		return fmt.Errorf("rtsp publish base is empty")
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
		return fmt.Errorf("failed to create rpicam stdout pipe: %w", err)
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
		return fmt.Errorf("failed to start rpicam-vid: %w", err)
	}

	if err := ffmpegCmd.Start(); err != nil {
		rpiCmd.Process.Kill()
		rpiCmd.Wait()
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	utils.Green.Printf("[ Publishing -> %s ]\n", publishURL)

	c.Stream = Stream{
		rpiCmd:    rpiCmd,
		ffmpegCmd: ffmpegCmd,
		isRunning: true,
	}

	return nil
}
