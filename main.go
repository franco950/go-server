package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

type dbStore interface {
	MilkById(id string) (*milk, error)
	AllMilk() ([]milk, error)
	SendMilk(m milk) (int64, error)
}
type App struct {
	store dbStore
}

type milk struct {
	CowID   string  `json:"cowid"`
	Fat     float64 `json:"fat"`
	Protein float64 `json:"protein"`
	PH      float64 `json:"pH"`
	SCC     int     `json:"scc"`
}

var ErrNotFound = errors.New("error, cow not found")

func (m milk) isValid() bool {
	return m.CowID != "" && m.Fat > 0 && m.Protein > 0 && m.PH > 0 && m.SCC > 0
}

func (a *App) milkById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	w.Header().Set("Content-Type", "application/json")
	cow, dbErr := a.store.MilkById(id)

	if dbErr == ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "cow not found"})
		return
	}
	if dbErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(dbErr)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		return
	}

	json.NewEncoder(w).Encode(cow)

}
func (a *App) allMilk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	milks, err := a.store.AllMilk()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		return
	}

	json.NewEncoder(w).Encode(milks)
}
func (a *App) sendMilk(w http.ResponseWriter, r *http.Request) {
	var cow milk
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&cow)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
		return
	}
	if !cow.isValid() {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing or invalid fields"})
		return
	}
	cowid, err := a.store.SendMilk(cow)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"success": "new cow created", "cowid": cowid})

}
func home(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("successfully connected to server")
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading env variables")
	}
	sqldb := MySQLsetup()
	app := &App{store: sqldb}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /milk", app.allMilk)
	mux.HandleFunc("GET /", home)
	mux.HandleFunc("GET /milk/{id}", app.milkById)
	mux.HandleFunc("POST /milk", app.sendMilk)
	log.Fatal(http.ListenAndServe(":8080", mux))

}
