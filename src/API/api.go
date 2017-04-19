package main

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/Samples", Get)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func Get(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("http://sboynton.com:3000/api/Samples")

	if err != nil {
		fmt.Println(err)
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Fprintf(w, string(body))
	}
}
