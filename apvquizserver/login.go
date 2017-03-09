package apvquizserver

import (
	"encoding/json"
	"net/http"
)

// login handler validates usernane and password
// and replies whether the user is authenticated
// Reply is in json form
//   {
//     "Auth" : true // true if authenticated else false
//   }
func Login(w http.ResponseWriter, r *http.Request) {

	// Get username and password from the request object
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Set content type of response
	w.Header().Set("Content-Type", "application/json")

	// Verify if username and password not empty
	if username != "" && password != "" {
		// Validate the username and password
		if ValidateUser(username, password) {
			//  Case authenticated
			data := struct {
				Auth bool
			}{true}
			js, _ := json.Marshal(data)
			w.Write(js)
		} else {
			// Case invalid username or password
			data := struct {
				Auth bool
			}{false}
			js, _ := json.Marshal(data)
			w.Write(js)
		}
	} else {
		// Case uername or password is empty
		data := struct {
			Auth bool
		}{false}
		js, _ := json.Marshal(data)
		w.Write(js)
	}
}
