package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Handler struct {
	dbm     *DBManager
	imgBase string
}

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("/api/databases", h.listDatabases)
	mux.HandleFunc("/api/db/", h.routeDB)
	mux.HandleFunc("/api/health", h.health)
	mux.HandleFunc("/api/files/delete", h.deleteFiles)
	mux.HandleFunc("/api/nfo/", h.findNFO)
	mux.HandleFunc("/api/media/", h.routeMedia)
	mux.HandleFunc("/api/decode-config", h.decodeConfig)
	mux.HandleFunc("/img/", h.proxyImage)
}

// StreamInfo describes a single media stream (video/audio/subtitle track)
type StreamInfo struct {
	Type       string `json:"type"`
	Codec      string `json:"codec"`
	Language   string `json:"language,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	Bitrate    int    `json:"bitrate,omitempty"`
	Channels   int    `json:"channels,omitempty"`
	SampleRate string `json:"sample_rate,omitempty"`
	IsDefault  bool   `json:"is_default"`
	Forced     bool   `json:"forced"`
	External   bool   `json:"is_external"`
	Profile    string `json:"profile,omitempty"`
	PixFmt     string `json:"pix_fmt,omitempty"`
	Duration   int    `json:"duration,omitempty"`
	Index      int    `json:"index"`
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]string{"status": "ok"})
}

func (h *Handler) listDatabases(w http.ResponseWriter, r *http.Request) {
	dbs := h.dbm.ListDatabases()
	jsonOK(w, dbs)
}

// routeDB handles /api/db/{dbname}/... routes
func (h *Handler) routeDB(w http.ResponseWriter, r *http.Request) {
	// Parse: /api/db/{dbname}/{rest...}
	path := strings.TrimPrefix(r.URL.Path, "/api/db/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		jsonError(w, "database name required", http.StatusBadRequest)
		return
	}

	dbName := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}

	switch {
	case rest == "" || rest == "tables":
		h.handleTables(w, r, dbName)
	case strings.HasPrefix(rest, "schema/"):
		table := strings.TrimPrefix(rest, "schema/")
		h.handleSchema(w, r, dbName, table)
	case rest == "query":
		h.handleQuery(w, r, dbName)
	case strings.HasPrefix(rest, "table/"):
		h.handleTableCRUD(w, r, dbName, strings.TrimPrefix(rest, "table/"))
	default:
		jsonError(w, "unknown endpoint", http.StatusNotFound)
	}
}

func (h *Handler) handleTables(w http.ResponseWriter, r *http.Request, dbName string) {
	tables, err := h.dbm.ListTables(dbName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, tables)
}

func (h *Handler) handleSchema(w http.ResponseWriter, r *http.Request, dbName, table string) {
	cols, err := h.dbm.GetTableSchema(dbName, table)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, cols)
}

func (h *Handler) handleQuery(w http.ResponseWriter, r *http.Request, dbName string) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.dbm.Query(dbName, req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, result)
}

// handleTableCRUD handles /api/db/{db}/table/{table}[/...] routes
func (h *Handler) handleTableCRUD(w http.ResponseWriter, r *http.Request, dbName, rest string) {
	parts := strings.SplitN(rest, "/", 2)
	table := parts[0]

	switch r.Method {
	case http.MethodGet:
		h.handleTableList(w, r, dbName, table)
	case http.MethodPost:
		h.handleTableInsert(w, r, dbName, table)
	case http.MethodPut:
		h.handleTableUpdate(w, r, dbName, table)
	case http.MethodDelete:
		h.handleTableDelete(w, r, dbName, table)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleTableList(w http.ResponseWriter, r *http.Request, dbName, table string) {
	// Build a simple SELECT * from query params
	page := 1
	size := 50
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if s := r.URL.Query().Get("size"); s != "" {
		fmt.Sscanf(s, "%d", &size)
	}

	sqlStr := fmt.Sprintf("SELECT * FROM %q", table)
	if sort := r.URL.Query().Get("sort"); sort != "" {
		if err := validateIdentifier(sort); err == nil {
			order := "ASC"
			if r.URL.Query().Get("order") == "desc" {
				order = "DESC"
			}
			sqlStr += fmt.Sprintf(" ORDER BY %q %s", sort, order)
		}
	}

	where := r.URL.Query().Get("where")
	if where != "" {
		sqlStr += " WHERE " + where
	}

	req := QueryRequest{SQL: sqlStr, Page: page, Size: size}
	result, err := h.dbm.Query(dbName, req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, result)
}

func (h *Handler) handleTableInsert(w http.ResponseWriter, r *http.Request, dbName, table string) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		jsonError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.dbm.InsertRow(dbName, table, data)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]interface{}{"last_insert_id": id})
}

func (h *Handler) handleTableUpdate(w http.ResponseWriter, r *http.Request, dbName, table string) {
	var req struct {
		Set   map[string]interface{} `json:"set"`
		Where string                 `json:"where"`
		Args  []interface{}          `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	affected, err := h.dbm.UpdateRow(dbName, table, req.Set, req.Where, req.Args)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]interface{}{"affected_rows": affected})
}

func (h *Handler) handleTableDelete(w http.ResponseWriter, r *http.Request, dbName, table string) {
	var req struct {
		Where string        `json:"where"`
		Args  []interface{} `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Where == "" {
		jsonError(w, "where clause required for delete", http.StatusBadRequest)
		return
	}

	affected, err := h.dbm.DeleteRow(dbName, table, req.Where, req.Args)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]interface{}{"affected_rows": affected})
}

func (h *Handler) deleteFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Paths) == 0 {
		jsonError(w, "paths is required", http.StatusBadRequest)
		return
	}

	type FileResult struct {
		Path    string `json:"path"`
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	var results []FileResult
	for _, path := range req.Paths {
		r := FileResult{Path: path}
		if path == "" {
			r.Error = "empty path"
		} else if _, err := os.Stat(path); os.IsNotExist(err) {
			r.Error = "file not found"
		} else if err := os.Remove(path); err != nil {
			r.Error = err.Error()
		} else {
			r.Success = true
		}
		results = append(results, r)
	}

	jsonOK(w, results)
}

func (h *Handler) findNFO(w http.ResponseWriter, r *http.Request) {
	// /api/nfo/{item_guid} - 在影片目录中查找 .nfo 文件
	guid := strings.TrimPrefix(r.URL.Path, "/api/nfo/")
	if guid == "" {
		jsonError(w, "item_guid required", http.StatusBadRequest)
		return
	}

	db, err := h.dbm.getDB("trimmedia.db")
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取 item 的 title 和 type
	var title, itemType string
	err = db.QueryRow("SELECT title, type FROM item WHERE guid = ?", guid).Scan(&title, &itemType)
	if err != nil {
		jsonError(w, "item not found", http.StatusNotFound)
		return
	}

	// 获取该 item 关联的所有文件目录
	rows, err := db.Query("SELECT DISTINCT dir FROM item_media WHERE item_guid = ?", guid)
	if err != nil {
		jsonError(w, "query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var fileDirs []string
	for rows.Next() {
		var dir string
		if err := rows.Scan(&dir); err == nil && dir != "" {
			fileDirs = append(fileDirs, dir)
		}
	}

	// 构建搜索目录列表
	searchDirs := make(map[string]bool)
	for _, dir := range fileDirs {
		searchDirs[dir] = true
	}

	// 对于 TV/Season/Episode，向上找 1 级父目录（剧集根目录）
	if itemType == "TV" || itemType == "Season" || itemType == "Episode" {
		for _, dir := range fileDirs {
			idx := strings.LastIndex(dir, "/")
			if idx > 0 {
				searchDirs[dir[:idx]] = true
			}
		}
	}

	// 搜索 nfo 文件
	type nfoResult struct {
		Path    string `json:"path"`
		Size    int64  `json:"size"`
		Content string `json:"content,omitempty"`
	}
	var nfoFiles []nfoResult
	seenPaths := make(map[string]bool)

	// ?raw 参数：直接返回第一个 NFO 的 XML 内容
	rawMode := r.URL.Query().Get("raw") == "true"

	for dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			if strings.HasSuffix(name, ".nfo") {
				fullPath := dir + "/" + entry.Name()
				if seenPaths[fullPath] {
					continue
				}
				seenPaths[fullPath] = true
				info, err := entry.Info()
				if err != nil {
					continue
				}
				nr := nfoResult{
					Path: fullPath,
					Size: info.Size(),
				}
				// 读取文件内容（限制 1MB）
				if info.Size() > 0 && info.Size() < 1024*1024 {
					data, err := os.ReadFile(fullPath)
					if err == nil {
						nr.Content = string(data)
					}
				}
				if rawMode {
					// 直接返回第一个 NFO 的 XML
					w.Header().Set("Content-Type", "application/xml")
					w.Write([]byte(nr.Content))
					return
				}
				nfoFiles = append(nfoFiles, nr)
			}
		}
	}

	jsonOK(w, map[string]interface{}{
		"guid":      guid,
		"title":     title,
		"type":      itemType,
		"file_dirs": fileDirs,
		"nfo_files": nfoFiles,
	})
}

func (h *Handler) proxyImage(w http.ResponseWriter, r *http.Request) {
	// /img/{path} -> {cacheDir}/{imgDir}/{path} 或 {cacheDir}/{imgDir}/{path}.{size}.0.-1
	imgPath := strings.TrimPrefix(r.URL.Path, "/img/")
	if imgPath == "" {
		http.Error(w, "image path required", http.StatusBadRequest)
		return
	}

	// 从数据库路径如 /4b/17/xxx.webp 去掉开头的 /
	imgPath = strings.TrimPrefix(imgPath, "/")

	// 支持 ?size= 参数，默认 400
	size := r.URL.Query().Get("size")
	if size == "" {
		size = "400"
	}

	// 尝试顺序：带 size 后缀 → 不带后缀
	tryPaths := []string{
		filepath.Join(h.imgBase, imgPath+"."+size+".0.-1"),
		filepath.Join(h.imgBase, imgPath),
	}

	var fullPath string
	var file *os.File
	for _, p := range tryPaths {
		f, err := os.Open(p)
		if err == nil {
			fullPath = p
			file = f
			defer file.Close()
			break
		}
	}
	if fullPath == "" {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "stat error", http.StatusInternalServerError)
		return
	}

	ext := filepath.Ext(imgPath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "image/webp"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.Header().Set("Cache-Control", "public, max-age=604800")
	io.Copy(w, file)
}

// routeMedia handles /api/media/{media_guid}[/stream|/play-info] routes
func (h *Handler) routeMedia(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/media/")
	if path == "" {
		jsonError(w, "media_guid required", http.StatusBadRequest)
		return
	}

	parts := strings.SplitN(path, "/", 2)
	mediaGuid := parts[0]

	if len(parts) == 2 {
		switch parts[1] {
		case "stream":
			h.streamMedia(w, r, mediaGuid)
			return
		case "play-info":
			h.playInfo(w, r, mediaGuid)
			return
		}
	}

	h.mediaInfo(w, r, mediaGuid)
}

// mediaInfo returns media file info + streams + decode config for a media_guid
func (h *Handler) mediaInfo(w http.ResponseWriter, r *http.Request, mediaGuid string) {
	db, err := h.dbm.getDB("trimmedia.db")
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 1. Get media file info from item_media
	var path, dir string
	var size int64
	var canPlay int
	err = db.QueryRow(
		"SELECT path, dir, size, can_play FROM item_media WHERE guid = ?", mediaGuid,
	).Scan(&path, &dir, &size, &canPlay)
	if err != nil {
		jsonError(w, "media not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Check file exists
	fileExists := false
	if info, err := os.Stat(path); err == nil {
		fileExists = true
		if size == 0 {
			size = info.Size()
		}
	}

	// 2. Get all streams (video/audio/subtitle)
	rows, err := db.Query(`
		SELECT codec_type, codec_name, language, width, height, bps,
		       channels, sample_rate, is_default, forced, is_external,
		       profile, pix_fmt, duration, "index"
		FROM media_stream WHERE media_guid = ? ORDER BY codec_type, "index"
	`, mediaGuid)
	if err != nil {
		jsonError(w, "stream query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var streams []StreamInfo
	for rows.Next() {
		var s StreamInfo
		var lang, sampleRate, profile, pixFmt string
		var width, height, bps, channels, duration, idx int
		var isDefault, forced, isExternal int

		if err := rows.Scan(&s.Type, &s.Codec, &lang, &width, &height, &bps,
			&channels, &sampleRate, &isDefault, &forced, &isExternal,
			&profile, &pixFmt, &duration, &idx); err != nil {
			continue
		}
		s.Language = lang
		s.Width = width
		s.Height = height
		s.Bitrate = bps
		s.Channels = channels
		s.SampleRate = sampleRate
		s.IsDefault = isDefault == 1
		s.Forced = forced == 1
		s.External = isExternal == 1
		s.Profile = profile
		s.PixFmt = pixFmt
		s.Duration = duration
		s.Index = idx
		streams = append(streams, s)
	}

	// 3. Get direct_link config from media_server
	var directLinkEnable, directLinkLevel int
	var directLinkDrives string
	_ = db.QueryRow(
		"SELECT direct_link_enable, direct_link_allowed_level, direct_link_allowed_drives FROM media_server LIMIT 1",
	).Scan(&directLinkEnable, &directLinkLevel, &directLinkDrives)

	// 4. Get decode/transcode config from sys_metadata (filtered by keywords)
	decodeKeys := []string{
		"transcode%", "hw_%", "gpu_%", "vaapi%", "qsv%", "nvenc%",
		"cuda%", "ffmpeg%", "decoder%", "encoder%", "hardware%",
		"accel%", "video_codecs", "audio_codecs", "mediasrv_%",
	}
	decodeConfig := make(map[string]string)
	for _, pattern := range decodeKeys {
		krows, err := db.Query("SELECT key, value FROM sys_metadata WHERE key LIKE ? AND private != 1", pattern)
		if err != nil {
			continue
		}
		for krows.Next() {
			var k, v string
			if err := krows.Scan(&k, &v); err == nil {
				decodeConfig[k] = v
			}
		}
		krows.Close()
	}

	// 5. Build response
	result := map[string]interface{}{
		"media_guid":  mediaGuid,
		"file_path":   path,
		"dir":         dir,
		"size":        size,
		"size_human":  humanizeBytes(size),
		"can_play":    canPlay == 1,
		"file_exists": fileExists,
		"streams":     streams,
		"direct_link": map[string]interface{}{
			"enabled":        directLinkEnable == 1,
			"allowed_level":  directLinkLevel,
			"allowed_drives": directLinkDrives,
		},
		"decode_config": decodeConfig,
	}

	// 6. Get item info (title, type) if available
	var itemTitle, itemType string
	_ = db.QueryRow(
		"SELECT i.title, i.type FROM item_media im JOIN item i ON im.item_guid = i.guid WHERE im.guid = ?",
		mediaGuid,
	).Scan(&itemTitle, &itemType)
	result["item_title"] = itemTitle
	result["item_type"] = itemType

	jsonOK(w, result)
}

// streamMedia serves the media file. Supports mode=direct (default) and mode=transcode.
// mode=direct  → http.ServeFile, supports Range/seek, client decodes
// mode=transcode → FFmpeg pipe, fragmented MP4, server decodes+encodes
// mode=auto → server decides based on codec compatibility
func (h *Handler) streamMedia(w http.ResponseWriter, r *http.Request, mediaGuid string) {
	db, err := h.dbm.getDB("trimmedia.db")
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var filePath string
	err = db.QueryRow("SELECT path FROM item_media WHERE guid = ?", mediaGuid).Scan(&filePath)
	if err != nil {
		jsonError(w, "media not found", http.StatusNotFound)
		return
	}

	if filePath == "" {
		jsonError(w, "empty file path", http.StatusNotFound)
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		jsonError(w, "file not found: "+filePath, http.StatusNotFound)
		return
	}

	if info.IsDir() {
		jsonError(w, "path is a directory", http.StatusBadRequest)
		return
	}

	// Parse mode parameter
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "direct"
	}

	switch mode {
	case "direct":
		// Direct file serving: client handles decoding
		log.Printf("streaming direct %s (%s) for %s", filePath, humanizeBytes(info.Size()), mediaGuid)
		http.ServeFile(w, r, filePath)

	case "transcode":
		// FFmpeg transcoding: server decodes + re-encodes
		params := TranscodeParams{
			Mode:          "transcode",
			TargetVCodec:  r.URL.Query().Get("vcodec"),
			TargetACodec:  r.URL.Query().Get("acodec"),
			TargetBitrate: parseIntDefault(r.URL.Query().Get("bitrate"), 0),
			TargetHeight:  parseIntDefault(r.URL.Query().Get("height"), 0),
			StartTime:     r.URL.Query().Get("start"),
			Duration:      r.URL.Query().Get("duration"),
			HwOverride:    r.URL.Query().Get("hw"),
		}
		transcodeStream(w, r, filePath, params)

	case "auto":
		// Server decides: check codec compatibility
		streams := h.getMediaStreams(db, mediaGuid)
		cfg := getTranscoderConfig()
		recommendedMode, reason := recommendPlayMode(streams, cfg.GPUType)
		log.Printf("auto mode: %s (%s) for %s", recommendedMode, reason, mediaGuid)

		if recommendedMode == "transcode" {
			params := TranscodeParams{
				Mode:         "transcode",
				TargetVCodec: "h264",
				TargetACodec: "aac",
			}
			transcodeStream(w, r, filePath, params)
		} else {
			http.ServeFile(w, r, filePath)
		}

	default:
		jsonError(w, "invalid mode: "+mode+" (use direct, transcode, or auto)", http.StatusBadRequest)
	}
}

// decodeConfig returns all sys_metadata entries related to transcoding/decoding
func (h *Handler) decodeConfig(w http.ResponseWriter, r *http.Request) {
	db, err := h.dbm.getDB("trimmedia.db")
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return ALL non-private sys_metadata keys (broad search)
	rows, err := db.Query("SELECT key, value FROM sys_metadata WHERE private != 1 ORDER BY key")
	if err != nil {
		jsonError(w, "query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			config[k] = v
		}
	}

	// Also get media_server direct_link config
	var name string
	var directLinkEnable, directLinkLevel int
	var directLinkDrives string
	_ = db.QueryRow(
		"SELECT name, direct_link_enable, direct_link_allowed_level, direct_link_allowed_drives FROM media_server LIMIT 1",
	).Scan(&name, &directLinkEnable, &directLinkLevel, &directLinkDrives)

	// Check for GPU device files
	gpuDevices := checkGPUDevices()

	jsonOK(w, map[string]interface{}{
		"sys_metadata":   config,
		"media_server": map[string]interface{}{
			"name":              name,
			"direct_link_enable": directLinkEnable == 1,
			"direct_link_level":  directLinkLevel,
			"direct_link_drives": directLinkDrives,
		},
		"gpu_devices": gpuDevices,
	})
}

// checkGPUDevices checks for common GPU device files on the system
func checkGPUDevices() map[string]interface{} {
	result := make(map[string]interface{})

	// Intel VAAPI (QuickSync): /dev/dri/renderD128
	if _, err := os.Stat("/dev/dri/renderD128"); err == nil {
		result["vaapi_device"] = "/dev/dri/renderD128"
		result["intel_qsv"] = true
	}
	// Intel i915 driver check
	if _, err := os.Stat("/dev/dri/card0"); err == nil {
		result["drm_card0"] = true
	}
	// NVIDIA: /dev/nvidia0
	if _, err := os.Stat("/dev/nvidia0"); err == nil {
		result["nvidia"] = true
	}
	// NVIDIA NVENC
	if _, err := os.Stat("/dev/nvidiactl"); err == nil {
		result["nvidia_ctl"] = true
	}

	return result
}

func humanizeBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// playInfo returns the recommended playback strategy for a media file.
// Frontend calls this first, then uses the result to choose mode=direct or mode=transcode.
func (h *Handler) playInfo(w http.ResponseWriter, r *http.Request, mediaGuid string) {
	db, err := h.dbm.getDB("trimmedia.db")
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 1. Get media file info
	var path, dir string
	var size int64
	var canPlay int
	err = db.QueryRow(
		"SELECT path, dir, size, can_play FROM item_media WHERE guid = ?", mediaGuid,
	).Scan(&path, &dir, &size, &canPlay)
	if err != nil {
		jsonError(w, "media not found: "+err.Error(), http.StatusNotFound)
		return
	}

	fileExists := false
	if info, err := os.Stat(path); err == nil {
		fileExists = true
		if size == 0 {
			size = info.Size()
		}
	}

	// 2. Get streams
	streams := h.getMediaStreams(db, mediaGuid)

	// 3. Get transcoder config
	cfg := getTranscoderConfig()

	// 4. Check codec compatibility
	compat := checkCodecCompatibility(streams)

	// 5. Get recommended mode
	recommendedMode, reason := recommendPlayMode(streams, cfg.GPUType)

	// 6. Get item title
	var itemTitle, itemType string
	_ = db.QueryRow(
		"SELECT i.title, i.type FROM item_media im JOIN item i ON im.item_guid = i.guid WHERE im.guid = ?",
		mediaGuid,
	).Scan(&itemTitle, &itemType)

	// 7. Build stream URLs
	baseURL := fmt.Sprintf("http://%s/api/media/%s", r.Host, mediaGuid)

	jsonOK(w, map[string]interface{}{
		"media_guid":   mediaGuid,
		"item_title":   itemTitle,
		"item_type":    itemType,
		"file_path":    path,
		"size":         size,
		"size_human":   humanizeBytes(size),
		"file_exists":  fileExists,
		"streams":      streams,

		// Playback strategy
		"recommended_mode": recommendedMode,
		"reason":           reason,

		// Codec compatibility matrix
		"codec_compatibility": map[string]interface{}{
			"browser_safe":    compat.BrowserSafe,
			"mobile_safe":     compat.MobileSafe,
			"desktop_safe":     compat.DesktopSafe,
			"need_transcode":  compat.NeedTranscode,
			"detail":          compat.Reason,
		},

		// Transcoder capability
		"transcoder": map[string]interface{}{
			"ffmpeg_path":   cfg.FFmpegPath,
			"gpu_type":      cfg.GPUType,
			"gpu_device":    cfg.GPUDevice,
			"max_cpu_threads": cfg.MaxThreads,
		},

		// Available stream URLs
		"stream_urls": map[string]interface{}{
			"direct":        baseURL + "/stream?mode=direct",
			"transcode":     baseURL + "/stream?mode=transcode",
			"transcode_cpu": baseURL + "/stream?mode=transcode&hw=cpu",
			"auto":          baseURL + "/stream?mode=auto",
			"transcode_custom": fmt.Sprintf("%s/stream?mode=transcode&vcodec=h264&acodec=aac&bitrate=4000&height=1080", baseURL),
		},
	})
}

// getMediaStreams queries media_stream table for a given media_guid
func (h *Handler) getMediaStreams(db *sql.DB, mediaGuid string) []StreamInfo {
	rows, err := db.Query(`
		SELECT codec_type, codec_name, language, width, height, bps,
		       channels, sample_rate, is_default, forced, is_external,
		       profile, pix_fmt, duration, "index"
		FROM media_stream WHERE media_guid = ? ORDER BY codec_type, "index"
	`, mediaGuid)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var streams []StreamInfo
	for rows.Next() {
		var s StreamInfo
		var lang, sampleRate, profile, pixFmt string
		var width, height, bps, channels, duration, idx int
		var isDefault, forced, isExternal int

		if err := rows.Scan(&s.Type, &s.Codec, &lang, &width, &height, &bps,
			&channels, &sampleRate, &isDefault, &forced, &isExternal,
			&profile, &pixFmt, &duration, &idx); err != nil {
			continue
		}
		s.Language = lang
		s.Width = width
		s.Height = height
		s.Bitrate = bps
		s.Channels = channels
		s.SampleRate = sampleRate
		s.IsDefault = isDefault == 1
		s.Forced = forced == 1
		s.External = isExternal == 1
		s.Profile = profile
		s.PixFmt = pixFmt
		s.Duration = duration
		s.Index = idx
		streams = append(streams, s)
	}
	return streams
}

// parseIntDefault parses an integer from a string, returning def on error
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	if err != nil {
		return def
	}
	return v
}

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": data,
	})
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    -1,
		"message": msg,
	})
	log.Printf("error [%d]: %s", code, msg)
}
