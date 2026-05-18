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
	RpiCmd    *exec.Cmd
	FfmpegCmd *exec.Cmd
	IsRunning bool
}

func (c *Concierge) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")

	return string(b)
}

func (c *Concierge) StreamController(action <-chan Message) error {
	for msg := range action {
		if msg.Action == "stop" {
			if !c.IsRunning {
				utils.Red.Println("> There is no active stream!")
			} else {
				utils.Blue.Println("> Stopping stream...")
				c.IsRunning = false
				c.Stop()
			}
		}

		if msg.Action == "start" {
			if c.IsRunning {
				utils.Red.Println("> There is already an active stream!")
			} else {
				utils.Blue.Println("> Starting stream...")
				c.IsRunning = true
				go c.PublishStream(utils.Hostname)
			}
		}
	}

	return nil
}

func (c *Concierge) Stop() error {
	var firstErr error

	if c.RpiCmd != nil && c.RpiCmd.Process != nil {
		if err := c.RpiCmd.Process.Kill(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.FfmpegCmd != nil && c.FfmpegCmd.Process != nil {
		if err := c.FfmpegCmd.Process.Kill(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if c.RpiCmd != nil {
		c.RpiCmd.Process.Wait()
	}
	if c.FfmpegCmd != nil {
		c.FfmpegCmd.Process.Wait()
	}

	return firstErr
}

func (c *Concierge) PublishStream(deviceID string) error {
	if c.RTSPPublishBase == "" {
		return fmt.Errorf("rtsp publish base is empty")
	}

	publishURL := fmt.Sprintf("%s/%s", c.RTSPPublishBase, deviceID)

	rpiCmd := exec.Command(
		"rpicam-vid",
		"-n",
		"-t", "0",
		"--mode", "1296:972:10:P",
		"--width", "1280",
		"--height", "720",
		"--framerate", "30",
		"--codec", "mjpeg",
		"--quality", "95",
		"--shutter", "33333",
		"--gain", "16.0",
		"--awb", "auto",
		"--denoise", "auto",
		"-o", "-",
	)

	rpiStdout, err := rpiCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create rpicam stdout pipe: %w", err)
	}
	rpiCmd.Stderr = os.Stdout

	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-f", "mjpeg",
		"-framerate", "30",
		"-i", "pipe:0",
		"-an",
		"-vf", "scale=in_range=pc:out_range=tv,format=yuv420p",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-tune", "zerolatency",
		"-profile:v", "high",
		"-level:v", "4.1",
		"-pix_fmt", "yuv420p",
		"-b:v", "15000k",
		"-maxrate", "15000k",
		"-bufsize", "1000k",
		"-g", "1",
		"-bf", "0",
		"-f", "rtsp",
		"-rtsp_transport", "tcp",
		"-pkt_size", "1200",
		publishURL,
	)
	ffmpegCmd.Stdin = rpiStdout
	ffmpegCmd.Stdout = io.Discard
	ffmpegCmd.Stderr = os.Stdout

	if err := rpiCmd.Start(); err != nil {
		return fmt.Errorf("failed to start rpicam-vid: %w", err)
	}

	if err := ffmpegCmd.Start(); err != nil {
		rpiCmd.Process.Kill()
		rpiCmd.Wait()
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	utils.Green.Printf("[ Publishing -> %s ]\n", publishURL)

	c.RpiCmd = rpiCmd
	c.FfmpegCmd = ffmpegCmd

	return nil
}
