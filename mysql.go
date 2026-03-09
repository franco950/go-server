package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
)

type MySQLdb struct {
	*sql.DB
}

func (db *MySQLdb) MilkById(ctx context.Context, id string) (*milk, error) {
	var cow milk
	row := db.QueryRowContext(ctx, "SELECT cowid, fat, protein, pH, scc FROM milk WHERE cowid=?", id)

	rowErr := row.Scan(&cow.CowID, &cow.Fat, &cow.Protein, &cow.PH, &cow.SCC)
	if rowErr == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if rowErr != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled, %w", ctx.Err())
		}
		return nil, fmt.Errorf("database-milkbyid %q:%w", id, rowErr)
	}
	return &cow, nil

}
func (db *MySQLdb) AllMilk(ctx context.Context) ([]milk, error) {
	var cows []milk
	rows, err := db.QueryContext(ctx, "SELECT cowid, fat, protein, pH, scc FROM milk")
	if err != nil {

		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled, %w", ctx.Err())
		}
		return nil, fmt.Errorf("allmilk query error: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cow milk
		scanErr := rows.Scan(&cow.CowID, &cow.Fat, &cow.Protein, &cow.PH, &cow.SCC)
		if scanErr != nil {

			return nil, fmt.Errorf("allmilk rows scan error:%w", scanErr)
		}
		cows = append(cows, cow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("allmilk rows error: %w", err)
	}
	return cows, nil

}
func (db *MySQLdb) SendMilk(ctx context.Context, m milk) (int64, error) {
	row, err := db.ExecContext(ctx, "INSERT INTO milk (cowid, fat, protein, pH, scc) VALUES (?,?,?,?,?)",
		m.CowID, m.Fat, m.Protein, m.PH, m.SCC)

	if err != nil {
		if ctx.Err() != nil {
			return 0, fmt.Errorf("context cancelled, %w", ctx.Err())
		}
		return 0, fmt.Errorf("error sending milk: %w", err)
	}

	id, idErr := row.LastInsertId()
	if idErr != nil {
		return 0, fmt.Errorf("error returning new entry id: %w", idErr)
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

	newdb.SetMaxOpenConns(25)
	newdb.SetMaxIdleConns(10)
	newdb.SetConnMaxLifetime(5 * time.Minute)
	sqldb := &MySQLdb{newdb}

	pingErr := sqldb.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("connected to MySQL database!")
	return sqldb

}
