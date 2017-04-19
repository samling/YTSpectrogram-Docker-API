package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
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

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/Samples/{Id}/VerifyAndSpawn", VerifyAndSpawn)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func VerifyAndSpawn(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// TODO: Sanitize the crap out of this (either here and/or in the extension itself):
	id := vars["Id"]

	exists := Exists(id)

	if exists == true {
		fmt.Fprintf(w, "Value exists\n")
	} else {
		fmt.Fprintf(w, "Value does not exist\n")
	}

	err := SpawnContainer(id)

	if err != nil {
		fmt.Fprintf(w, "Container did not spawn\n", err)
	} else {
		fmt.Fprintf(w, "Container spawned\n")
	}

}

func Exists(id string) bool {
	// Check if the entry exists already
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://sboynton.com:3000/api/Samples/"+id+"/exists", nil)
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

func SpawnContainer(Id string) error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	env := make([]string, 1)
	env[0] = "YTID=" + Id

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
