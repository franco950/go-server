package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-sql-driver/mysql"
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
type MySQLdb struct {
	*sql.DB
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

func (db *MySQLdb) MilkById(id string) (*milk, error) {
	var cow milk
	row := db.QueryRow("SELECT cowid, fat, protein, pH, scc FROM milk WHERE cowid=?", id)

	rowErr := row.Scan(&cow.CowID, &cow.Fat, &cow.Protein, &cow.PH, &cow.SCC)
	if rowErr == sql.ErrNoRows {
		return nil, ErrNotFound //fmt.Errorf("cow not found: %q", id)
	}
	if rowErr != nil {
		return nil, fmt.Errorf("database-milkbyid %q:%v", id, rowErr)
	}
	return &cow, nil

}
func (db *MySQLdb) AllMilk() ([]milk, error) {
	var cows []milk
	rows, err := db.Query("SELECT cowid, fat, protein, pH, scc FROM milk")
	if err != nil {
		return nil, fmt.Errorf("allmilk query error: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cow milk
		scanErr := rows.Scan(&cow.CowID, &cow.Fat, &cow.Protein, &cow.PH, &cow.SCC)
		if scanErr != nil {
			return nil, fmt.Errorf("allmilk rows scan error:%v", scanErr)
		}
		cows = append(cows, cow)
	}
	return cows, nil

}
func (db *MySQLdb) SendMilk(m milk) (int64, error) {
	row, err := db.Exec("INSERT INTO milk (cowid, fat, protein, pH, scc) VALUES (?,?,?,?,?)",
		m.CowID, m.Fat, m.Protein, m.PH, m.SCC)

	if err != nil {
		return 0, fmt.Errorf("error sending milk: %v", err)
	}

	id, idErr := row.LastInsertId()
	if idErr != nil {
		return 0, fmt.Errorf("error returning new entry id: %v", idErr)
	}
	return id, nil

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
func MySQLsetup() *MySQLdb {
	cfg := mysql.NewConfig()
	cfg.User = os.Getenv("DBUSER")
	cfg.Passwd = os.Getenv("DBPASS")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("DBHOST") + ":" + os.Getenv("DBPORT")
	cfg.DBName = os.Getenv("DBNAME")

	newdb, dbErr := sql.Open("mysql", cfg.FormatDSN())
	if dbErr != nil {
		log.Fatal(dbErr)
	}
	sqldb := &MySQLdb{newdb}
	pingErr := sqldb.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("connected to MySQL database!")
	return sqldb

}
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading env variables")
	}
	sqldb := MySQLsetup()
	app := &App{sqldb}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /milk", app.allMilk)
	mux.HandleFunc("GET /", home)
	mux.HandleFunc("GET /milk/{id}", app.milkById)
	mux.HandleFunc("POST /milk", app.sendMilk)
	log.Fatal(http.ListenAndServe(":8080", mux))

}
