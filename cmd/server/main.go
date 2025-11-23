package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" 
	"pr-reviewer/internal/api"
	"pr-reviewer/internal/store"
)

func main() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", dbUser, dbPass, dbHost, dbPort, dbName)

	var db *sql.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("pgx", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for database... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer db.Close()
	
	log.Println("Connected to database")

	st := store.New(db)
	h := api.NewHandler(st)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("POST /team/add", h.CreateTeam)
	mux.HandleFunc("GET /team/get", h.GetTeam)
	mux.HandleFunc("POST /team/bulkDeactivate", h.BulkDeactivate)

	mux.HandleFunc("POST /users/setIsActive", h.SetUserActive)
	mux.HandleFunc("GET /users/getReview", h.GetUserReviews)

	mux.HandleFunc("POST /pullRequest/create", h.CreatePullRequest)
	mux.HandleFunc("POST /pullRequest/merge", h.MergePullRequest)
	mux.HandleFunc("POST /pullRequest/reassign", h.ReassignReviewer)

	mux.HandleFunc("GET /stats", h.GetStats)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("Server starting on :8080")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}