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
	fmt.Println("")
	fmt.Println("Login .....")
	fmt.Println("Method : ", r.Method)
	fmt.Println("Content-Type : ", r.Header.Get("Content-Type"))
	ctype := r.Header.Get("Content-Type")
	var username string
	var password string
	if (ctype == "application/x-www-form-urlencoded") {
		// r.ParseForm()
		username = r.FormValue("username")
		password = r.FormValue("password")
	} else {
		fmt.Println("Cannot do")
	}
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
	fmt.Println("..... close")
}

func main() {
	http.HandleFunc("/", hello)
	http.HandleFunc("/login", login)
	http.ListenAndServe(":8000", nil)
}