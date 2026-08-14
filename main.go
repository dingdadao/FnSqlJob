package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	Version   = "dev"
	BuildTime = "unknown"

	dbPath  = flag.String("dbpath", "/usr/local/apps/@appdata/trim.media/database/", "SQLite database directory")
	imgPath = flag.String("imgpath", "/vol1/@appmeta/trim.media/cache/img", "Image cache directory")
	addr    = flag.String("addr", ":8877", "HTTP listen address")
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

	mux := http.NewServeMux()
	h := &Handler{dbm: dbm, imgBase: *imgPath}
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
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
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
