package main

import (
	"encoding/json"
	"net/http"
)

func login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	w.Header().Set("Content-Type", "application/json")

	if username != "" && password != "" {
		if validateUser(username, password) {
			data := struct {
				Auth bool
			}{true}
			js, _ := json.Marshal(data)
			w.Write(js)
		} else {
			data := struct {
				Auth bool
			}{false}
			js, _ := json.Marshal(data)
			w.Write(js)
		}

	} else {
		data := struct {
			Auth bool
		}{false}
		js, _ := json.Marshal(data)
		w.Write(js)
	}
}
