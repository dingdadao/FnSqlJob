package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// TranscoderConfig holds hardware acceleration configuration detected at startup
type TranscoderConfig struct {
	FFmpegPath string // path to ffmpeg binary
	GPUType    string // "vaapi", "nvenc", or "cpu"
	GPUDevice  string // device path, e.g. "/dev/dri/renderD128"
	MaxThreads int    // max concurrent software transcodes (CPU mode)
}

// TranscodeParams holds frontend-specified transcoding parameters
type TranscodeParams struct {
	Mode          string // direct, transcode, auto
	TargetVCodec  string // h264, hevc (default: h264)
	TargetACodec  string // aac, opus (default: aac)
	TargetBitrate int    // target video bitrate in kbps (0 = auto)
	TargetHeight  int    // target resolution height (0 = source)
	StartTime     string // seek position (HH:MM:SS or seconds)
	Duration      string // duration to transcode
	HwOverride    string // "cpu" to force software encoding (skip GPU)
}

var (
	transcoderCfg  *TranscoderConfig
	transcoderOnce sync.Once
)

// getTranscoderConfig lazily detects and caches the transcoder configuration
func getTranscoderConfig() *TranscoderConfig {
	transcoderOnce.Do(func() {
		transcoderCfg = detectTranscoderConfig()
		log.Printf("transcoder config: ffmpeg=%s gpu=%s device=%s",
			transcoderCfg.FFmpegPath, transcoderCfg.GPUType, transcoderCfg.GPUDevice)
	})
	return transcoderCfg
}

// detectTranscoderConfig probes the system for FFmpeg and GPU devices
func detectTranscoderConfig() *TranscoderConfig {
	cfg := &TranscoderConfig{
		GPUType:    "cpu",
		MaxThreads: 2,
	}

	// Find FFmpeg binary
	candidates := []string{
		"/usr/local/bin/ffmpeg",
		"/usr/bin/ffmpeg",
		"/usr/local/apps/mediasrv/bin/ffmpeg",
		"/usr/local/apps/trim.media/bin/ffmpeg",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			cfg.FFmpegPath = p
			break
		}
	}
	if cfg.FFmpegPath == "" {
		if p, err := exec.LookPath("ffmpeg"); err == nil {
			cfg.FFmpegPath = p
		}
	}

	// Detect GPU: Intel VAAPI (QuickSync) first, then NVIDIA
	if _, err := os.Stat("/dev/dri/renderD128"); err == nil {
		cfg.GPUType = "vaapi"
		cfg.GPUDevice = "/dev/dri/renderD128"
	} else if _, err := os.Stat("/dev/nvidia0"); err == nil {
		cfg.GPUType = "nvenc"
		cfg.GPUDevice = "/dev/nvidia0"
	}

	return cfg
}

// effectiveGPUType returns the GPU type after applying frontend override
func effectiveGPUType(params TranscodeParams, cfg *TranscoderConfig) string {
	if params.HwOverride == "cpu" {
		return "cpu"
	}
	return cfg.GPUType
}

// buildFFmpegArgs constructs the FFmpeg command line for transcoding.
// The output is fragmented MP4 piped to stdout.
func buildFFmpegArgs(inputPath string, params TranscodeParams, cfg *TranscoderConfig) []string {
	gpuType := effectiveGPUType(params, cfg)
	args := []string{}

	// Suppress verbose output, keep errors
	args = append(args, "-loglevel", "error")

	// Seek position BEFORE -i for fast seek (no decode before seeking)
	if params.StartTime != "" {
		args = append(args, "-ss", params.StartTime)
	}

	// Hardware acceleration for DECODING the input
	// Note: we do NOT use -hwaccel_output_format here.
	// Software filters (scale) need frames in system memory.
	// hwupload will push them back to GPU for encoding.
	switch gpuType {
	case "vaapi":
		args = append(args,
			"-hwaccel", "vaapi",
			"-vaapi_device", cfg.GPUDevice,
		)
	case "nvenc":
		args = append(args,
			"-hwaccel", "cuda",
		)
	}

	// Input file
	args = append(args, "-i", inputPath)

	// Duration limit (after -i for accuracy)
	if params.Duration != "" {
		args = append(args, "-t", params.Duration)
	}

	// Video encoder selection based on GPU type and target codec
	vcodec := params.TargetVCodec
	if vcodec == "" {
		vcodec = "h264"
	}
	switch gpuType {
	case "vaapi":
		if vcodec == "hevc" {
			args = append(args, "-c:v", "hevc_vaapi")
		} else {
			args = append(args, "-c:v", "h264_vaapi")
		}
	case "nvenc":
		if vcodec == "hevc" {
			args = append(args, "-c:v", "hevc_nvenc")
		} else {
			args = append(args, "-c:v", "h264_nvenc")
		}
	default:
		// CPU software encoding
		if vcodec == "hevc" {
			args = append(args, "-c:v", "libx265", "-preset", "fast")
		} else {
			args = append(args, "-c:v", "libx264", "-preset", "veryfast")
		}
	}

	// Bitrate control
	bitrate := params.TargetBitrate
	if bitrate == 0 {
		bitrate = 4000 // default 4 Mbps
	}
	args = append(args,
		"-b:v", fmt.Sprintf("%dk", bitrate),
		"-maxrate", fmt.Sprintf("%dk", bitrate*3/2),
		"-bufsize", fmt.Sprintf("%dk", bitrate*2),
	)

	// Build filter chain
	// For VAAPI: software scale → format nv12 → hwupload to GPU surfaces
	// For NVENC: software scale → format nv12 → hwupload_cuda
	// For CPU: software scale only
	filters := []string{}
	if params.TargetHeight > 0 {
		filters = append(filters, fmt.Sprintf("scale=-2:%d", params.TargetHeight))
	}
	switch gpuType {
	case "vaapi":
		filters = append(filters, "format=nv12", "hwupload")
	case "nvenc":
		filters = append(filters, "format=nv12", "hwupload_cuda")
	}
	if len(filters) > 0 {
		args = append(args, "-vf", strings.Join(filters, ","))
	}

	// Audio codec
	acodec := params.TargetACodec
	if acodec == "" {
		acodec = "aac"
	}
	args = append(args, "-c:a", acodec, "-b:a", "128k")

	// Output: fragmented MP4 to pipe (streaming-friendly, no moov atom needed upfront)
	args = append(args,
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov",
		"pipe:1",
	)

	return args
}

// transcodeStream runs FFmpeg and pipes its stdout to the HTTP response.
// If GPU transcoding produces 0 bytes, it automatically retries with CPU.
func transcodeStream(w http.ResponseWriter, r *http.Request, inputPath string, params TranscodeParams) {
	cfg := getTranscoderConfig()
	if cfg.FFmpegPath == "" {
		http.Error(w, "FFmpeg not found on this system", http.StatusInternalServerError)
		return
	}

	// Detect ISO files — FFmpeg can't directly read disc images
	if strings.HasSuffix(strings.ToLower(inputPath), ".iso") {
		http.Error(w, "ISO disc images are not supported for transcoding. Use direct mode or mount the ISO first.", http.StatusBadRequest)
		return
	}

	gpuType := effectiveGPUType(params, cfg)

	// First attempt: use detected GPU (or CPU if no GPU)
	args := buildFFmpegArgs(inputPath, params, cfg)
	log.Printf("transcode start [%s]: %s %s", gpuType, cfg.FFmpegPath, strings.Join(args, " "))

	// Capture stderr for error reporting
	var stderrBuf bytes.Buffer
	ctx := r.Context()
	cmd := exec.CommandContext(ctx, cfg.FFmpegPath, args...)
	cmd.Stderr = io.MultiWriter(&stderrBuf, &ffmpegLogWriter{})

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "pipe error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := cmd.Start(); err != nil {
		http.Error(w, "ffmpeg start error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Read first bytes to check if FFmpeg is actually producing output
	// We use a peeking reader to check
	peekSize := 4096
	peekBuf := make([]byte, peekSize)
	n, _ := io.ReadFull(stdout, peekBuf)
	peekBuf = peekBuf[:n]

	// If GPU mode produced 0 bytes and we have GPU, retry with CPU
	if n == 0 && gpuType != "cpu" {
		log.Printf("transcode: GPU (%s) produced 0 bytes, falling back to CPU", gpuType)
		cmd.Wait()

		// Retry with CPU
		cpuParams := params
		cpuParams.HwOverride = "cpu"
		cpuArgs := buildFFmpegArgs(inputPath, cpuParams, cfg)
		log.Printf("transcode retry [cpu]: %s %s", cfg.FFmpegPath, strings.Join(cpuArgs, " "))

		stderrBuf.Reset()
		cmd = exec.CommandContext(ctx, cfg.FFmpegPath, cpuArgs...)
		cmd.Stderr = io.MultiWriter(&stderrBuf, &ffmpegLogWriter{})

		stdout, err = cmd.StdoutPipe()
		if err != nil {
			http.Error(w, "pipe error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := cmd.Start(); err != nil {
			http.Error(w, "ffmpeg start error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Re-peek
		n, _ = io.ReadFull(stdout, peekBuf[:peekSize])
		peekBuf = peekBuf[:n]
		gpuType = "cpu"
	}

	// Set response headers
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Transcode-GPU", gpuType)
	w.Header().Set("X-Transcode-VCodec", defaultStr(params.TargetVCodec, "h264"))
	w.Header().Set("X-Transcode-Bitrate", fmt.Sprintf("%d", params.TargetBitrate))

	// If still 0 bytes, return error with FFmpeg stderr
	if n == 0 {
		errMsg := strings.TrimSpace(stderrBuf.String())
		log.Printf("transcode failed (0 bytes): %s", errMsg)
		w.Header().Set("X-Transcode-Error", truncate(errMsg, 200))
		http.Error(w, "FFmpeg produced no output: "+truncate(errMsg, 500), http.StatusInternalServerError)
		cmd.Wait()
		return
	}

	// Write the peeked bytes, then pipe the rest
	w.WriteHeader(http.StatusOK)
	w.Write(peekBuf)
	io.Copy(w, stdout)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == nil {
			log.Printf("ffmpeg exited with error: %v", err)
		}
	}
	log.Printf("transcode done [%s]: %s", gpuType, inputPath)
}

// truncate cuts a string to maxLen, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ffmpegLogWriter captures FFmpeg stderr for debugging
type ffmpegLogWriter struct{}

func (lw *ffmpegLogWriter) Write(p []byte) (int, error) {
	line := strings.TrimSpace(string(p))
	if line != "" && !strings.Contains(line, "frame=") {
		log.Printf("[ffmpeg] %s", line)
	}
	return len(p), nil
}

// defaultStr returns val if non-empty, otherwise def
func defaultStr(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

// codecCompatibility checks if the source codec can be played directly by common clients
type codecCompatResult struct {
	BrowserSafe   bool   // H.264/H.265 + AAC in MP4 container
	MobileSafe    bool   // H.264 + AAC
	DesktopSafe   bool   // mpv/VLC can handle almost anything
	Reason        string // human-readable explanation
	NeedTranscode bool   // true if browser/mobile can't play natively
}

// checkCodecCompatibility analyzes the media streams and returns compatibility info
func checkCodecCompatibility(streams []StreamInfo) codecCompatResult {
	result := codecCompatResult{
		BrowserSafe: true,
		MobileSafe:  true,
		DesktopSafe: true,
	}

	var videoCodec, audioCodec string
	for _, s := range streams {
		if s.Type == "video" {
			videoCodec = s.Codec
		}
		if s.Type == "audio" && s.IsDefault {
			audioCodec = s.Codec
		}
	}
	if audioCodec == "" {
		for _, s := range streams {
			if s.Type == "audio" {
				audioCodec = s.Codec
				break
			}
		}
	}

	// Browser compatibility: Chrome/Firefox support H.264, some support H.265
	// AV1 is supported by modern browsers but may be slow on low-end devices
	browserVideoOK := map[string]bool{
		"h264": true, "h265": true, "hevc": true, "av1": true, "vp9": true,
		"mpeg4": false, "msmpeg4": false, "wmv3": false, "vc1": false,
		"mpeg2video": false, "flv1": false,
	}
	browserAudioOK := map[string]bool{
		"aac": true, "mp3": true, "opus": true, "vorbis": true,
		"ac3": false, "dts": false, "truehd": false, "flac": false,
		"eac3": false, "pcm_s16le": false,
	}

	if !browserVideoOK[videoCodec] {
		result.BrowserSafe = false
		result.Reason += fmt.Sprintf("video codec %s not supported by browser; ", videoCodec)
	}
	if !browserAudioOK[audioCodec] {
		result.BrowserSafe = false
		result.Reason += fmt.Sprintf("audio codec %s not supported by browser; ", audioCodec)
	}

	// Mobile: stricter than browser
	mobileVideoOK := map[string]bool{"h264": true, "hevc": true, "av1": false}
	if !mobileVideoOK[videoCodec] {
		result.MobileSafe = false
	}
	mobileAudioOK := map[string]bool{"aac": true, "mp3": true}
	if !mobileAudioOK[audioCodec] {
		result.MobileSafe = false
	}

	// Desktop players (mpv/VLC/IINA) can decode almost anything
	result.DesktopSafe = true

	result.NeedTranscode = !result.BrowserSafe

	if result.Reason == "" {
		result.Reason = "all codecs are browser-safe, direct play recommended"
	}

	return result
}

// recommendPlayMode decides the best playback mode based on codec + GPU availability
func recommendPlayMode(streams []StreamInfo, gpuType string) (mode string, reason string) {
	compat := checkCodecCompatibility(streams)

	if compat.BrowserSafe {
		return "direct", "source codecs are browser-safe, no transcoding needed"
	}

	// Need transcoding
	if gpuType == "cpu" {
		// No GPU — transcoding will be slow
		return "direct", "no GPU detected, recommend direct stream + client-side decoding (mpv/VLC handle " +
			"HEVC/DTS natively); CPU transcoding is too slow for real-time playback"
	}

	return "transcode", fmt.Sprintf("source codecs need transcoding for browser playback; GPU (%s) available for hardware transcoding", gpuType)
}
