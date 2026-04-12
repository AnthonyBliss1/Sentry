package network

import (
	"bufio"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
)

type TCPServer struct {
	listener net.Listener
	wg       sync.WaitGroup
}

func handleTCPStream(conn net.Conn, hlsDir string) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	deviceID := conn.LocalAddr().String()
	if deviceID == "" {
		utils.Red.Println("empty deviceID from TCP client")
		return
	}

	deviceDir := filepath.Join(hlsDir, deviceID)
	if _, err := os.Stat(deviceDir); err == nil {
		// deviceDir exists so it needs to be reset
		if err := os.RemoveAll(deviceDir); err != nil {
			utils.Red.Printf("failed to clean dir from old stream: %s\n", deviceDir)
		}
	}

	if err := os.MkdirAll(deviceDir, 0o755); err != nil {
		utils.Red.Printf("failed creating device dir: %v\n", err)
		return
	}

	segmentPattern := filepath.Join(deviceDir, "segment_%03d.ts")
	playlistPath := filepath.Join(deviceDir, "stream.m3u8")

	cmd := exec.Command(
		"ffmpeg",
		"-loglevel", "warning",
		"-f", "h264",
		"-r", "15",
		"-i", "pipe:0",
		"-an",
		"-c:v", "copy",
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "6",
		"-hls_flags", "delete_segments+temp_file+omit_endlist",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		utils.Red.Printf("failed getting ffmpeg stdin: %v\n", err)
		return
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		stdin.Close()
		utils.Red.Printf("failed starting ffmpeg: %v\n", err)
		return
	}

	utils.Blue.Printf("[ Ingest started for %s ]\n", deviceID)

	_, copyErr := io.Copy(stdin, reader)
	stdin.Close()

	waitErr := cmd.Wait()

	if copyErr != nil {
		utils.Red.Printf("stream copy failed for %s: %v\n", deviceID, copyErr)
	}

	if waitErr != nil {
		utils.Red.Printf("ffmpeg exited for %s: %v\n", deviceID, waitErr)
	}
}
