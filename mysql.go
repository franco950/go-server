package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
)

type MySQLdb struct {
	*sql.DB
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
