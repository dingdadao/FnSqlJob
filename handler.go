package main

import (
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
	mux.HandleFunc("/img/", h.proxyImage)
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

	var dirs []string
	for rows.Next() {
		var dir string
		if err := rows.Scan(&dir); err == nil && dir != "" {
			dirs = append(dirs, dir)
		}
	}

	// 对于 TV 类型，也查找季/集所在的父目录
	if itemType == "TV" || itemType == "Season" {
		var parentDirs []string
		for _, dir := range dirs {
			// 向上查找一级父目录
			parent := dir
			if idx := strings.LastIndex(dir, "/"); idx > 0 {
				parent = dir[:idx]
			}
			if idx := strings.LastIndex(parent, "/"); idx > 0 {
				parentDirs = append(parentDirs, parent[:idx])
			}
		}
		dirs = append(dirs, parentDirs...)
	}

	// 去重
	seen := make(map[string]bool)
	var uniqueDirs []string
	for _, dir := range dirs {
		if !seen[dir] {
			seen[dir] = true
			uniqueDirs = append(uniqueDirs, dir)
		}
	}

	// 搜索 nfo 文件
	type nfoResult struct {
		Path string `json:"path"`
		Size int64  `json:"size"`
	}
	var nfoFiles []nfoResult

	for _, dir := range uniqueDirs {
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
				info, err := entry.Info()
				if err == nil {
					nfoFiles = append(nfoFiles, nfoResult{
						Path: dir + "/" + entry.Name(),
						Size: info.Size(),
					})
				}
			}
		}
	}

	jsonOK(w, map[string]interface{}{
		"guid":      guid,
		"title":     title,
		"type":      itemType,
		"dirs":      uniqueDirs,
		"nfo_files": nfoFiles,
	})
}

func (h *Handler) proxyImage(w http.ResponseWriter, r *http.Request) {
	// /img/{path} -> /vol1/@appmeta/trim.media/cache/img/{path}.{size}.0.-1
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

	// 拼接实际文件路径: imgBase/4b/17/xxx.webp.400.0.-1
	fullPath := filepath.Join(h.imgBase, imgPath+"."+size+".0.-1")

	file, err := os.Open(fullPath)
	if err != nil {
		// 回退尝试不带 size 后缀的原始文件
		fullPath = filepath.Join(h.imgBase, imgPath)
		file, err = os.Open(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "image not found", http.StatusNotFound)
			} else {
				http.Error(w, "open error", http.StatusInternalServerError)
			}
			return
		}
	}
	defer file.Close()

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
