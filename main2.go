package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

type dbstore2 interface {
	Milkbyid2(ctx context.Context, id string) (*milk, error)
	allmilk(ctx context.Context) ([]milk, error)
	sendmilk(ctx context.Context, one milk) (int64, error)
}
type app2 struct {
	store dbstore2
}

var ErrNewnotfound = errors.New("not found")

func (n *milk) isvalid2() bool {
	if n.CowID == "" || n.Fat < 0 || n.PH < 0 || n.SCC < 0 {
		return false
	}
	return true
}
func (a *app2) milkbyid2(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	w.Header().Set("Content-Type", "application/json")
	cow, err := a.store.Milkbyid2(r.Context(), id)
	if err == ErrNewnotfound {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "ccow no found"})
		return
	}
	if err != nil {
		log.Println(err)
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusGatewayTimeout)
			json.NewEncoder(w).Encode(map[string]string{"error": "time out"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		return
	}
	json.NewEncoder(w).Encode(&cow)
}
func (a *app2) allmilk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	milks, err := a.store.allmilk(r.Context())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusGatewayTimeout)
			json.NewEncoder(w).Encode(map[string]string{"error": "time out"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		return

	}
	json.NewEncoder(w).Encode(milks)
}
func (a *app2) sendmilks(w http.ResponseWriter, r *http.Request) {
	var cow milk
	defer r.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	err := json.NewDecoder(r.Body).Decode(&cow)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
		return
	}

	if !cow.isvalid2() {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "check your fields"})
		return
	}
	id, dberr := a.store.sendmilk(r.Context(), cow)
	if dberr != nil {
		if errors.Is(dberr, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusGatewayTimeout)
			json.NewEncoder(w).Encode(map[string]string{"error": "time out"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"success": "new cow added", "id": id})

}
func (a *app2) home2(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("you have reached server")

}
func main2() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("could not load env")
	}
	_, ok := os.LookupEnv("API_KEY")
	if !ok {
		log.Fatal("no apikey found")
	}
	sqldb := mysqlconfig()
	app := &app2{store: sqldb}
	mux := http.NewServeMux()
	http.HandleFunc("GET /", app.home2)
	http.HandleFunc("GET /milk", app.allmilk)
	http.HandleFunc("GET /milk/{id}", app.milkbyid2)
	http.HandleFunc("POST /milk", app.sendmilks)

	srv := http.Server{
		Addr:    ":8080",
		Handler: logger(Authenticator(TimeKeeper(mux))),
	}
	//internal errors
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()
	//external errors
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("internal server error,%v", err)

		}
	case <-quit:
		log.Println("shutdown signal received")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("error shutting down server: %v", err)
	}

}
