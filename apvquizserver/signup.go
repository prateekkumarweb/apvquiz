package apvquizserver

import (
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"regexp"
)

// signup function ceates new user and saves in the database
// Reply is in the json from
//   {
//     "Status": true, // Is signup Successful
//     "Message": "" // Reason if signup is unsuccessful
//   }
func Signup(w http.ResponseWriter, r *http.Request) {

	// Get username and password from the request object
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Set content type of response
	w.Header().Set("Content-Type", "application/json")

	// Validate the username sent
	if regexp.MustCompile(`[\dA-Za-z]+`).MatchString(username) {
		data := struct {
			Status  bool
			Message string
		}{false, "Username should contain only alphanumeric characters"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}

	// Legth of username should be >= 4
	if len(username) < 4 {
		data := struct {
			Status  bool
			Message string
		}{false, "Username should atleast 4 characters"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}

	// Username or password cannot be empty
	if username == "" || password == "" {
		data := struct {
			Status  bool
			Message string
		}{false, "Username or password cannot be empty"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}

	// Hash the password to store in database
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// Save the user details in the database
	_, err = database.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, hashedPassword)
	if err == nil {
		// If saved in database
		data := struct {
			Status  bool
			Message string
		}{true, "Successful"}
		js, _ := json.Marshal(data)
		w.Write(js)
	} else {
		// If not saved in the database
		// TODO handle error
		data := struct {
			Status  bool
			Message string
		}{false, "Use another username"}
		js, _ := json.Marshal(data)
		w.Write(js)
	}
}
