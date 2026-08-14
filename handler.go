package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Handler struct {
	dbm *DBManager
}

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("/api/databases", h.listDatabases)
	mux.HandleFunc("/api/db/", h.routeDB)
	mux.HandleFunc("/api/health", h.health)
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
