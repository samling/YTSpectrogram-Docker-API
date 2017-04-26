package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"
	_ "golang.org/x/net/context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type Data struct {
	Exists bool
}

type Samples struct {
	Id         string
	SampleData []byte
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

func ReadLines(filePath string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return lines
}

func VerifyAndCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	// TODO: Sanitize the crap out of this (either here and/or in the extension itself):
	id := vars["Id"]

	exists := Exists(id)

	if exists == true {
		data, err := GetSampleData(id)
		if err != nil {
			fmt.Fprintf(w, "Could not parse sample data")
			log.Fatal(err)
		} else {
			fmt.Fprintf(w, string(data))
		}
	} else {
		err := CreateContainer(id)
		if err != nil {
			fmt.Fprintf(w, "Value does not exist\n Creating new container")
		}
	}
}

func Exists(id string) bool {
	// Check if the entry exists already
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://localhost:3001/api/Samples/"+id+"/exists", nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error when sending request ", err)
	}
	defer resp.Body.Close()

	// Read the response body into a byte array
	body, _ := ioutil.ReadAll(resp.Body)

	// Create a new Data struct to read into
	var data Data

	// Unmarshal our JSON byte array into a struct
	err = json.Unmarshal(body, &data)

	return data.Exists
}

func GetSampleData(id string) ([]byte, error) {
	// Check if the entry exists already
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://localhost:3001/api/Samples/"+id, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error when sending request ", err)
	}
	defer resp.Body.Close()

	// Read the response body into a byte array
	body, _ := ioutil.ReadAll(resp.Body)

	// Create a new Data struct to read into
	var samples Samples

	// Unmarshal our JSON byte array into a struct
	err = json.Unmarshal(body, &samples)

	return samples.SampleData, err
}

func CreateContainer(Id string) error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	config := ReadLines("./config")

	env := make([]string, 1)
	env[0] = "YTID=" + Id
	env[1] = config[0]
	env[2] = config[1]
	env[3] = config[2]
	env[4] = config[3]

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "yts",
		Env:   env,
	}, nil, nil, "")
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
