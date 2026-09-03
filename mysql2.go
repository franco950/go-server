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

type Mysqldb struct {
	*sql.DB
}

func (db *Mysqldb) Milkbyid2(ctx context.Context, id string) (*milk, error) {
	var cow milk
	row := db.QueryRowContext(ctx, "SELECT cow_id,fat,protein,ph,scc FROM milk WHERE id=?", id)

	rowerr := row.Scan(&cow.CowID, &cow.Fat, &cow.Protein, &cow.PH, &cow.SCC)
	if rowerr == sql.ErrNoRows {
		return nil, ErrNewnotfound
	}
	if rowerr != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled, %w", ctx.Err())
		}
		return nil, fmt.Errorf("error in db-milkbyid, %q,%w", id, rowerr)
	}
	return &cow, nil

}
func (db *Mysqldb) allmilk(ctx context.Context) ([]milk, error) {
	var cows []milk
	rows, err := db.QueryContext(ctx, "SELECT cow_id,fat,protein,ph,scc FROM milk")
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled, %w", ctx.Err())
		}
		return nil, fmt.Errorf("error querying from db, %w", err)

	}
	defer rows.Close()

	for rows.Next() {
		var cow milk
		scanerr := rows.Scan(&cow.CowID, &cow.Fat, &cow.PH, &cow.Protein, &cow.SCC)
		if scanerr != nil {
			return nil, fmt.Errorf("error scanning rows")
		}
		cows = append(cows, cow)

	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error returning rows,%q", err)
	}
	return cows, nil
}
func (db *Mysqldb) sendmilk(ctx context.Context, one milk) (int64, error) {
	row, err := db.ExecContext(ctx, "INSERT INTO milk(cow_id,fat,protein,ph,scc) VALUES (?,?,?,?,?)", one.CowID, one.Fat, one.Protein, one.SCC)
	if err != nil {
		if ctx.Err() != nil {
			return 0, fmt.Errorf("context cancelled, %w", ctx.Err())

		}
		return 0, fmt.Errorf("error adding to db %w", err)

	}
	id, iderr := row.LastInsertId()
	if iderr != nil {
		return 0, fmt.Errorf("error inserting row")
	}
	return id, nil

}
func mysqlconfig() *Mysqldb {
	cfg := mysql.NewConfig()
	cfg.User = os.Getenv("DBUSER")
	cfg.Passwd = os.Getenv("DBPASS")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("DBHOST" + ":" + "DBPORT")
	cfg.DBName = os.Getenv("DBMNAME")
	newdb, dberr := sql.Open("mysql", cfg.FormatDSN())
	if dberr != nil {
		log.Fatal(dberr)
	}
	newdb.SetMaxOpenConns(25)
	newdb.SetMaxIdleConns(5)
	newdb.SetConnMaxLifetime(5 * time.Minute)
	sqldb := &Mysqldb{newdb}
	pingerr := sqldb.Ping()
	if pingerr != nil {
		log.Fatal(pingerr)
	}
	return sqldb
}
