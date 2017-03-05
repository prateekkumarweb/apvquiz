package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

func signup(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	w.Header().Set("Content-Type", "application/json")
	if !regexp.MustCompile(`[\dA-Za-z]`).MatchString(username) {
		data := struct {
			Status  bool
			Message string
		}{false, "Username should contain only alphanumeric characters"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	if len(username) < 4 {
		data := struct {
			Status  bool
			Message string
		}{false, "Username should atleast 4 characters"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	if username == "" || password == "" {
		data := struct {
			Status  bool
			Message string
		}{false, "Username or password cannot be empty"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	res, err := database.Exec("INSERT INTO users (username, password) VALUES (\"" + username + "\", \"" + password + "\")")
	fmt.Println(res)
	if err == nil {
		data := struct {
			Status  bool
			Message string
		}{true, "Successful"}
		js, _ := json.Marshal(data)
		w.Write(js)
	} else {
		data := struct {
			Status  bool
			Message string
		}{false, "Use another username"}
		js, _ := json.Marshal(data)
		w.Write(js)
	}
}
