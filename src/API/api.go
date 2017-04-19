package main

import (
	_ "database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "html"
	"log"
	"net/http"
	"os"
)

type Data struct {
	Id         string
	SampleData []byte
}

type Sample struct {
	Timestamp int
	Value     float64
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/Samples", Get)
	log.Fatal(http.ListenAndServe(":8080", router))
}

// Formulate a connection string from environment variables
func GetConnectionString(dbHost string, dbUser string, dbPass string, dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", dbUser, dbPass, dbHost, dbName)
}

func Get(w http.ResponseWriter, r *http.Request) {
	connString := GetConnectionString(os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"))
	db := sqlx.MustConnect("mysql", connString)

	db.MapperFunc(func(s string) string { return s })

	rows, err := db.Queryx("SELECT Id, SampleData FROM Sample")

	for rows.Next() {
		var data Data
		var sample Sample

		err = rows.StructScan(&data)
		if err != nil {
			log.Fatal(err)
		}

		err = json.Unmarshal(data.SampleData, &sample)

		fmt.Fprintf(w, "%+v", data)
	}
}
