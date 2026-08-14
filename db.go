package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

type DBManager struct {
	dir string
	mu  sync.RWMutex
	dbs map[string]*sql.DB
}

func NewDBManager(dir string) (*DBManager, error) {
	m := &DBManager{dir: dir, dbs: make(map[string]*sql.DB)}
	// Pre-open known databases
	for _, name := range []string{"trimactivity.db", "trimmedia.db", "trimmedia_ext.db"} {
		if _, err := m.getDB(name); err != nil {
			log.Printf("warning: cannot open %s: %v", name, err)
		}
	}
	return m, nil
}

func (m *DBManager) getDB(name string) (*sql.DB, error) {
	m.mu.RLock()
	db, ok := m.dbs[name]
	m.mu.RUnlock()
	if ok {
		return db, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check
	if db, ok = m.dbs[name]; ok {
		return db, nil
	}

	// Validate name to prevent path traversal
	if strings.Contains(name, "/") || strings.Contains(name, "..") || strings.HasSuffix(name, ".") {
		return nil, fmt.Errorf("invalid database name: %s", name)
	}

	path := filepath.Join(m.dir, name)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("database file not found: %s", path)
	}

	dsn := fmt.Sprintf("file:%s?mode=ro", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(5)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping %s: %w", path, err)
	}

	m.dbs[name] = db
	return db, nil
}

// GetDB returns the raw *sql.DB for a database, or nil if not found.
func (m *DBManager) GetDB(name string) *sql.DB {
	db, _ := m.getDB(name)
	return db
}

func (m *DBManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, db := range m.dbs {
		db.Close()
		delete(m.dbs, name)
	}
}

func (m *DBManager) ListDatabases() []string {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil
	}
	var dbs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".db") {
			dbs = append(dbs, e.Name())
		}
	}
	return dbs
}

func (m *DBManager) ListTables(dbName string) ([]string, error) {
	db, err := m.getDB(dbName)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}
	return tables, nil
}

type ColumnInfo struct {
	CID          int         `json:"cid"`
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	NotNull      bool        `json:"notnull"`
	DefaultValue interface{} `json:"dflt_value"`
	PK           int         `json:"pk"`
}

func (m *DBManager) GetTableSchema(dbName, table string) ([]ColumnInfo, error) {
	db, err := m.getDB(dbName)
	if err != nil {
		return nil, err
	}
	// Sanitize table name
	if err := validateIdentifier(table); err != nil {
		return nil, err
	}

	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var c ColumnInfo
		var dflt sql.NullString
		if err := rows.Scan(&c.CID, &c.Name, &c.Type, &c.NotNull, &dflt, &c.PK); err != nil {
			return nil, err
		}
		if dflt.Valid {
			c.DefaultValue = dflt.String
		}
		cols = append(cols, c)
	}
	return cols, nil
}

type QueryRequest struct {
	SQL    string        `json:"sql"`
	Params []interface{} `json:"params"`
	Page   int           `json:"page"`
	Size   int           `json:"size"`
}

type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	Size    int             `json:"size"`
}

func (m *DBManager) Query(dbName string, req QueryRequest) (*QueryResult, error) {
	db, err := m.getDB(dbName)
	if err != nil {
		return nil, err
	}

	sqlStr := strings.TrimSpace(req.SQL)
	if sqlStr == "" {
		return nil, fmt.Errorf("empty SQL")
	}

	upperSQL := strings.ToUpper(sqlStr)

	// Handle SELECT queries with pagination
	if strings.HasPrefix(upperSQL, "SELECT") || strings.HasPrefix(upperSQL, "WITH") {
		hasLimit := strings.Contains(upperSQL, " LIMIT ")

		if req.Page < 1 {
			req.Page = 1
		}
		if req.Size < 1 || req.Size > 1000 {
			req.Size = 50
		}

		var total int
		var pagedSQL string

		if hasLimit {
			// SQL already has LIMIT, don't wrap or add pagination
			pagedSQL = sqlStr
			total = -1 // unknown
		} else {
			// Get total count
			countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s)", sqlStr)
			if err := db.QueryRow(countSQL, req.Params...).Scan(&total); err != nil {
				return nil, fmt.Errorf("count query error: %w", err)
			}
			// Add pagination
			offset := (req.Page - 1) * req.Size
			pagedSQL = fmt.Sprintf("%s LIMIT %d OFFSET %d", sqlStr, req.Size, offset)
		}

		rows, err := db.Query(pagedSQL, req.Params...)
		if err != nil {
			return nil, fmt.Errorf("query error: %w", err)
		}
		defer rows.Close()

		columns, _ := rows.Columns()
		result := &QueryResult{
			Columns: columns,
			Total:   total,
			Page:    req.Page,
			Size:    req.Size,
		}

		for rows.Next() {
			values := make([]interface{}, len(columns))
			ptrs := make([]interface{}, len(columns))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return nil, fmt.Errorf("scan error: %w", err)
			}
			// Convert []byte to string for JSON
			for i, v := range values {
				if b, ok := v.([]byte); ok {
					values[i] = string(b)
				}
			}
			result.Rows = append(result.Rows, values)
		}
		return result, nil
	}

	// Handle INSERT/UPDATE/DELETE (non-SELECT)
	// For write operations, we need a writable connection
	result, err := db.Exec(sqlStr, req.Params...)
	if err != nil {
		return nil, fmt.Errorf("exec error: %w", err)
	}
	affected, _ := result.RowsAffected()
	return &QueryResult{
		Columns: []string{"affected_rows"},
		Rows:    [][]interface{}{{affected}},
		Total:   1,
		Page:    1,
		Size:    1,
	}, nil
}

// InsertRow inserts a row into the specified table
func (m *DBManager) InsertRow(dbName, table string, data map[string]interface{}) (int64, error) {
	db, err := m.getDB(dbName)
	if err != nil {
		return 0, err
	}
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("no data to insert")
	}

	var cols []string
	var placeholders []string
	var values []interface{}
	for col, val := range data {
		if err := validateIdentifier(col); err != nil {
			return 0, fmt.Errorf("invalid column name: %s", col)
		}
		cols = append(cols, fmt.Sprintf("%q", col))
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	sql := fmt.Sprintf("INSERT INTO %q (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	result, err := db.Exec(sql, values...)
	if err != nil {
		return 0, fmt.Errorf("insert error: %w", err)
	}
	return result.LastInsertId()
}

// UpdateRow updates rows matching the WHERE clause
func (m *DBManager) UpdateRow(dbName, table string, data map[string]interface{}, where string, whereArgs []interface{}) (int64, error) {
	db, err := m.getDB(dbName)
	if err != nil {
		return 0, err
	}
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}

	var sets []string
	var values []interface{}
	for col, val := range data {
		if err := validateIdentifier(col); err != nil {
			return 0, fmt.Errorf("invalid column name: %s", col)
		}
		sets = append(sets, fmt.Sprintf("%q = ?", col))
		values = append(values, val)
	}
	values = append(values, whereArgs...)

	sqlStr := fmt.Sprintf("UPDATE %q SET %s", table, strings.Join(sets, ", "))
	if where != "" {
		sqlStr += " WHERE " + where
	}

	result, err := db.Exec(sqlStr, values...)
	if err != nil {
		return 0, fmt.Errorf("update error: %w", err)
	}
	return result.RowsAffected()
}

// DeleteRow deletes rows matching the WHERE clause
func (m *DBManager) DeleteRow(dbName, table, where string, whereArgs []interface{}) (int64, error) {
	db, err := m.getDB(dbName)
	if err != nil {
		return 0, err
	}
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}

	sqlStr := fmt.Sprintf("DELETE FROM %q", table)
	if where != "" {
		sqlStr += " WHERE " + where
	}

	result, err := db.Exec(sqlStr, whereArgs...)
	if err != nil {
		return 0, fmt.Errorf("delete error: %w", err)
	}
	return result.RowsAffected()
}

func validateIdentifier(s string) error {
	if s == "" {
		return fmt.Errorf("empty identifier")
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("invalid identifier character: %c", c)
		}
	}
	return nil
}
