package main

import (
	"io"
	"fmt"
	"net/http"
	"encoding/json"
)

func hello(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello world!")
}

type Data struct {
	Auth bool
}

func login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	fmt.Println(username+":"+password)
	w.Header().Set("Content-Type", "application/json")
	if (username == "prateek" && password == "pass") {
		data := Data{true}
		js, _ := json.Marshal(data)
		w.Write(js)
	} else {
		data := Data{false}
		js, _ := json.Marshal(data)
		w.Write(js)
	}
	// io.WriteString(w, "username : " + username + "\npassword : " + password + "\n")
}

func main() {
	http.HandleFunc("/", hello)
	http.HandleFunc("/login", login)
	http.ListenAndServe(":8000", nil)
}