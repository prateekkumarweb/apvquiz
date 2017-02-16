package main

import (
	"io"
	"fmt"
	"net/http"
	"encoding/json"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

var database *sql.DB

func hello(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello world!")
}

type Data struct {
	Auth bool
}

type Result struct {
	Status bool
	Message string
}

func login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("")
	fmt.Println("Login .....")
	fmt.Println("Method : ", r.Method)
	fmt.Println("Content-Type : ", r.Header.Get("Content-Type"))
	var username string
	var password string
    username = r.FormValue("username")
	password = r.FormValue("password")
    fmt.Println(username+":"+password)
    if (username != "" && password != "") {
    	rows, err := database.Query("SELECT * FROM users WHERE username=\""+username+"\" AND password=\""+password+"\"")
    	defer rows.Close()
    	if err == nil {
    		var done bool
    		done = false
	    	for rows.Next() {
	    		done = true
	    		var id int
	    		var uname, pword string
	    		rows.Scan(&id, &uname, &pword)
	    	}
    		if done {
	    		data := Data{true}
				js, _ := json.Marshal(data)
				w.Write(js)
			} else {
				data := Data{false}
				js, _ := json.Marshal(data)
				w.Write(js)
			}
		} else {
			data := Data{false}
			js, _ := json.Marshal(data)
			w.Write(js)
		}
	} else {
		data := Data{false}
		js, _ := json.Marshal(data)
		w.Write(js)
	}
	fmt.Println("..... close")
}

func signup(w http.ResponseWriter, r *http.Request) {
	fmt.Println("")
	fmt.Println("Login .....")
	fmt.Println("Method : ", r.Method)
	fmt.Println("Content-Type : ", r.Header.Get("Content-Type"))
	var username string
	var password string
    username = r.FormValue("username")
	password = r.FormValue("password")
    fmt.Println(username+":"+password)
    if (username != "" && password != "") {
    	fmt.Println(database)
    	res, err := database.Exec("INSERT INTO users (username, password) VALUES (\""+username+"\", \""+password+"\")")
    	fmt.Println(res)
    	if err == nil {
    		data := Result{true, "Successful"}
    		js, _ := json.Marshal(data)
    		w.Write(js)
    	} else {
    		data := Result{false, "Use another username"}
    		js, _ := json.Marshal(data)
    		w.Write(js)
    	}
    } else {
    	data := Result{false, "Username or password cannot be empty"}
		js, _ := json.Marshal(data)
		w.Write(js)
    }
	fmt.Println("..... close")
}

func main() {
	db, _ := sql.Open("mysql", "root:passw0rd@/apvquiz")
	database = db
	fmt.Println(db)
	fmt.Println(database.Ping())
	http.HandleFunc("/", hello)
	http.HandleFunc("/login", login)
	http.HandleFunc("/signup", signup)
	http.ListenAndServe(":8000", nil)
}