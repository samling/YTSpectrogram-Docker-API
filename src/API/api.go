package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	_ "github.com/docker/go-connections/nat"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/acme/autocert"
	_ "golang.org/x/net/context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Data struct {
	Exists bool
}

type Samples struct {
	Id         string
	SampleData string
}

func GetConnectionString(dbHost string, dbUser string, dbPass string, dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", dbUser, dbPass, dbHost, dbName)
}

func Redirect(w http.ResponseWriter, req *http.Request) {
	// remove/add not default ports from req.Host
	target := "https://" + req.Host + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	log.Printf("redirect to: %s", target)
	http.Redirect(w, req, target,
		// see @andreiavrammsd comment: often 307 > 301
		http.StatusTemporaryRedirect)
}

func ReadConfig(filePath string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var lines []string
	var linesSplit []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	for _, line := range lines {
		l := strings.Split(line, "=")
		linesSplit = append(linesSplit, l[1])
	}

	return linesSplit
}

func VerifyAndCreate(w http.ResponseWriter, r *http.Request) {
	// Get values from our config file
	config := ReadConfig("./config.cfg")
	host := config[0]
	user := config[1]
	pass := config[2]
	name := config[3]

	// Connect to our database
	connString := GetConnectionString(host, user, pass, name)
	db := sqlx.MustConnect("mysql", connString)

	// Set our response header to allow cross-origin access
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Retrieve the youtube video Id from the URL parameter
	vars := mux.Vars(r)
	// TODO: Sanitize the crap out of this (either here and/or in the extension itself):
	id := string(vars["Id"])

	// Try and retrieve sample data from the database
	data, err := GetSampleData(id, db)
	if err != nil {
		// If the video Id doesn't exist in the database, spawn a new
		// YTS container to download, analyze and store the sample data
		err := CreateContainer(id)
		if err != nil {
			log.Print(err)
		} else {
			// Then display the data
			data, err := GetSampleData(id, db)
			if err != nil {
				log.Print(err)
			} else {
				fmt.Fprintf(w, string(data))
			}
		}
	} else {
		// If the video in question is already in the DB, just display the JSON data
		fmt.Fprintf(w, string(data))
	}
}

func GetSampleData(id string, db *sqlx.DB) (string, error) {
	sample, err := db.Preparex(`SELECT SampleData FROM Sample WHERE Id=?`)
	var s string
	err = sample.Get(&s, id)

	if err != nil {
		log.Print(err)
		return "", err
	}

	return s, nil
}

func CreateContainer(Id string) error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	config := ReadConfig("./config.cfg")

	env := make([]string, 5)
	env[0] = "YTID=" + Id
	env[1] = "DB_HOST=" + config[0]
	env[2] = "DB_USER=" + config[1]
	env[3] = "DB_PASS=" + config[2]
	env[4] = "DB_NAME=" + config[3]

	//portBindings := map[nat.Port][]nat.PortBinding{"3306/tcp": []nat.PortBinding{nat.PortBinding{HostPort: "3306"}}}

	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: "yts",
			Env:   env,
		}, &container.HostConfig{
			NetworkMode: "host",
			//PortBindings: portBindings,
		}, nil, "")
	if err != nil {
		return err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	if _, err = cli.ContainerWait(ctx, resp.ID); err != nil {
		return err
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return err
	}

	io.Copy(os.Stdout, out)

	return nil
}

func main() {
	router := mux.NewRouter().StrictSlash(true)

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("sboynton.com"), //your domain here
		Cache:      autocert.DirCache("./certs"),           //folder for storing certificates
	}

	server := &http.Server{
		Handler: router,
		Addr:    ":443",
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	go http.ListenAndServe(":80", http.HandlerFunc(Redirect))
	router.HandleFunc("/api/Samples/{Id}", VerifyAndCreate)
	log.Fatal(server.ListenAndServeTLS("", ""))
}
