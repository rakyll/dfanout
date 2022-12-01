package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dfanout/dfanout/clientcache"
	"github.com/dfanout/dfanout/fanout"
	pb "github.com/dfanout/dfanout/proto"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

var (
	listen       string
	peers        string
	postgresConn string
)

func main() {
	ctx := context.Background()

	flag.StringVar(&listen, "listen", ":8080", "")
	flag.StringVar(&peers, "peers", "", "")
	flag.StringVar(&postgresConn, "postgres-connection", "postgres://postgres:@localhost:5432/dfanout", "")
	flag.Parse()

	if port := os.Getenv("PORT"); port != "" {
		listen = ":" + port
	}
	conn, err := pgx.Connect(ctx, postgresConn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	ccache := clientcache.New()
	adminService := &adminService{pgConn: conn}
	adminServer := pb.NewAdminServiceServer(adminService)
	fanoutCache := fanout.NewFanoutCache(
		listen,
		strings.Split(peers, ","),
		ccache,
		adminService,
		1*time.Minute,
	)
	mux := mux.NewRouter()
	mux.Handle("/_groupcache/", fanoutCache)
	mux.Handle("/fanout/{name}", &fanout.Handler{
		ClientCache: ccache,
		FanoutCache: fanoutCache,
	})
	mux.PathPrefix(adminServer.PathPrefix()).Handler(adminServer)

	log.Printf("Starting server at %v...", listen)
	log.Fatal(http.ListenAndServe(listen, mux))
}
