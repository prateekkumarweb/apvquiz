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
	rows, err := database.Query(fmt.Sprintf("INSERT INTO %s VALUES (0, '%s', '%s', '%s', '%s', '%s', %s)", topic, question, option1, option2, option3, option4, answer))
	defer rows.Close()
	if err != nil {
		data := struct {
			Status  bool
			Message string
		}{false, "Error!"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	rows1, err := database.Query("SELECT * FROM users WHERE username='" + username + "' AND password='" + password + "'")
	defer rows1.Close()
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
	for rows1.Next() {
		rows1.Scan(&id, &username, &password, &points, &games, &contributions)
	}
	contributions += 1
	rows2, err := database.Query(fmt.Sprintf("UPDATE users SET contributions=%v WHERE username=\"%s\"", contributions, username))
	defer rows2.Close()
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
