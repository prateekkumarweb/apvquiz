package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func contribute(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	question := r.FormValue("question")
	option1 := r.FormValue("option1")
	option2 := r.FormValue("option2")
	option3 := r.FormValue("option3")
	option4 := r.FormValue("option4")
	answer := r.FormValue("correct")
	topic := strings.ToLower(strings.Replace(r.FormValue("subject"), " ", "", -1))
	w.Header().Set("Content-Type", "application/json")
	_, err := database.Exec(fmt.Sprintf("INSERT INTO %s VALUES (0, '%s', '%s', '%s', '%s', '%s', %s)", topic, question, option1, option2, option3, option4, answer))
	if err != nil {
		data := struct {
			Status  bool
			Message string
		}{false, "Error!"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	var id, points, games, contributions int
	err = database.QueryRow("SELECT * FROM users WHERE username=? AND password=?", username, password).Scan(&id, &username, &password, &points, &games, &contributions)
	if err != nil {
		data := struct {
			Status  bool
			Message string
		}{false, "Error!"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	contributions += 1
	_, err = database.Exec("UPDATE users SET contributions=? WHERE username=?", contributions, username)
	if err != nil {
		data := struct {
			Status  bool
			Message string
		}{false, "Error!"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	data := struct {
		Status  bool
		Message string
	}{true, "Thanks for contributing"}
	js, _ := json.Marshal(data)
	w.Write(js)
}
