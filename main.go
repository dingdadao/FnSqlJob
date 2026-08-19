package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	Version   = "dev"
	BuildTime = "unknown"

	dbPath  = flag.String("dbpath", "/usr/local/apps/@appdata/trim.media/database/", "SQLite database directory")
	addr    = flag.String("addr", ":8877", "HTTP listen address")
	imgPath = flag.String("imgpath", "@appmeta/trim.media/img", "Image directory (relative to mediasrv_cache_dir)")
)

func main() {
	flag.Parse()

	if err := os.MkdirAll(*dbPath, 0755); err != nil {
		log.Printf("warning: cannot create dbpath dir: %v", err)
	}

	dbm, err := NewDBManager(*dbPath)
	if err != nil {
		log.Fatalf("failed to init database manager: %v", err)
	}
	defer dbm.Close()

	// 从 sys_metadata 读取 mediasrv_cache_dir 动态构建图片路径
	imgBase := resolveImageBase(dbm)

	mux := http.NewServeMux()
	h := &Handler{dbm: dbm, imgBase: imgBase}
	RegisterRoutes(mux, h)

	server := &http.Server{Addr: *addr, Handler: corsMiddleware(logMiddleware(mux))}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down...")
		server.Close()
	}()

	fmt.Printf("FnSqlDB %s (built %s)\n", Version, BuildTime)
	fmt.Printf("Listening on %s\n", *addr)
	fmt.Printf("Database path: %s\n", *dbPath)
	fmt.Printf("Image base: %s\n", imgBase)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func resolveImageBase(dbm *DBManager) string {
	// 尝试从 trimmedia.db 的 sys_metadata 表读取 mediasrv_cache_dir
	db := dbm.GetDB("trimmedia.db")
	if db == nil {
		log.Printf("warning: cannot open trimmedia.db, using default img path")
		return "/vol1/" + *imgPath
	}

	var cacheDir string
	err := db.QueryRow("SELECT value FROM sys_metadata WHERE key = 'mediasrv_cache_dir'").Scan(&cacheDir)
	if err != nil || cacheDir == "" {
		log.Printf("warning: cannot read mediasrv_cache_dir: %v, using default", err)
		return "/vol1/" + *imgPath
	}

	log.Printf("mediasrv_cache_dir: %s", cacheDir)
	return strings.TrimRight(cacheDir, "/") + "/" + *imgPath
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
