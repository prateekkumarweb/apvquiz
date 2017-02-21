package main

import (
	"io"
	"bufio"
	"fmt"
	"net/http"
	"net"
	"log"
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
    username := r.FormValue("username")
	password := r.FormValue("password")
	w.Header().Set("Content-Type", "application/json")
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
    username := r.FormValue("username")
	password := r.FormValue("password")
	w.Header().Set("Content-Type", "application/json")
    fmt.Println(username+":"+password)
    if (username != "" && password != "") {
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



type Player struct {
	conn net.Conn
	username string
	password string
	ch chan string
}

var players []Player

//var waiting Player

type Game struct {
	player1 Player
	player2 Player
}

func askCredentials(c net.Conn, bufc *bufio.Reader) (string, string) {
	user, _, _ := bufc.ReadLine()
	username := string(user)
	pass, _, _ := bufc.ReadLine()
	password := string(pass)
	return username, password
}

func validateUser(player Player) bool {
	username := player.username
	password := player.password
	rows, err := database.Query("SELECT * FROM users WHERE username=\""+username+"\" AND password=\""+password+"\"")
    defer rows.Close()
    if err == nil {
		var done bool
		done = false
		var id int
		var uname, pword string
    	for rows.Next() {
    		done = true
    		rows.Scan(&id, &uname, &pword)
    	}
		if done {
    		return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func handleClient(c net.Conn) {
	bufc := bufio.NewReader(c)
	defer c.Close()
	username, password := askCredentials(c, bufc)
	player := Player{c, username, password, make(chan string)}
	if !validateUser(player) {
		io.WriteString(player.conn, "Invalid\n")
		return
	}
	io.WriteString(player.conn, "Valid\n")
}

func main() {
	players = make([]Player, 0)
	database, _ = sql.Open("mysql", "root:123@/apvquiz")
	err := database.Ping()
	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		http.HandleFunc("/", hello)
		http.HandleFunc("/login", login)
		http.HandleFunc("/signup", signup)
		//http.HandleFunc("/play/", play)
		http.ListenAndServe(":8000", nil)
	}()

	ln, err := net.Listen("tcp", ":6000")
	if err != nil {
		log.Fatal(err)
		return
	}

	for {
		conn, err := ln.Accept()
		if (err != nil) {
			log.Println(err)
			continue
		}
		go handleClient(conn)
	}
}
