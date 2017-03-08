package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
)

// contribute function saves the questions sent by players in the database
// Reply is in the json form
// {
//   "Status": true, // Is contribution Successful
//   "Message": "" // Reason if contribution is unsuccessful
// }
func contribute(w http.ResponseWriter, r *http.Request) {

	// Read from post data sent by the user
	username := r.FormValue("username")
	password := r.FormValue("password")
	question := r.FormValue("question")
	option1 := r.FormValue("option1")
	option2 := r.FormValue("option2")
	option3 := r.FormValue("option3")
	option4 := r.FormValue("option4")
	answer := r.FormValue("correct")
	topic := strings.ToLower(strings.Replace(r.FormValue("subject"), " ", "", -1))

	// Set content tyoe of response
	w.Header().Set("Content-Type", "application/json")

	var id, points, games, contributions int
	var dbPassword string
	err = database.QueryRow("SELECT * FROM users WHERE username=?", username).Scan(&id, &username, &dbPassword, &points, &games, &contributions)
	if err != nil {
		// TODO Log error
		data := struct {
			Status  bool
			Message string
		}{false, "Error!"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}

	// Comapre user password with password in databse
	err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
	if err != nil {
		// If wrong password, don't save
		data := struct {
			Status  bool
			Message string
		}{false, "Wrong Password!"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}

	// Save question sent by the user into the database
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

	// Add 1 to contribution by the user and save in the database
	contributions += 1
	_, err = database.Exec("UPDATE users SET contributions=? WHERE username=?", contributions, username)
	if err != nil {
		// If not saved
		// TODO Log error
		data := struct {
			Status  bool
			Message string
		}{false, "Error!"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}

	// Now the data has been saved, so send the Status
	data := struct {
		Status  bool
		Message string
	}{true, "Thanks for contributing"}
	js, _ := json.Marshal(data)
	w.Write(js)
}
