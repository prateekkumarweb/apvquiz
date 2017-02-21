package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sort"
)

var database *sql.DB

func hello(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello world!")
}

type Data struct {
	Auth bool
}

type Result struct {
	Status  bool
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
	fmt.Println(username + ":" + password)
	if username != "" && password != "" {
		rows, err := database.Query("SELECT * FROM users WHERE username=\"" + username + "\" AND password=\"" + password + "\"")
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
	fmt.Println(username + ":" + password)
	if username != "" && password != "" {
		res, err := database.Exec("INSERT INTO users (username, password) VALUES (\"" + username + "\", \"" + password + "\")")
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
	conn        net.Conn
	username    string
	password    string
	ch          chan string
	otherPlayer []*Player
}

var players []Player

var waiting []*Player

func validateUser(player Player) bool {
	username := player.username
	password := player.password
	rows, err := database.Query("SELECT * FROM users WHERE username=\"" + username + "\" AND password=\"" + password + "\"")
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

type Players []*Player

func (ps Players) Len() int {
	return len(ps)
}

func (ps Players) Less(i, j int) bool {
	b := strings.Compare(ps[i].username, ps[j].username)
	if b == -1 {
		return true
	} else {
		return false
	}
}

func (ps Players) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

func handleClient(c net.Conn) {
	bufc := bufio.NewReader(c)
	defer c.Close()
	user, _, _ := bufc.ReadLine()
	username := string(user)
	pass, _, _ := bufc.ReadLine()
	password := string(pass)
	player := Player{c, username, password, make(chan string), nil}
	if !validateUser(player) {
		io.WriteString(player.conn, "Invalid\n")
		return
	}
	io.WriteString(player.conn, "Valid\n")
	// TODO waiting lock
	//Start lock
	if len(waiting) < 2 {
		waiting = append(waiting, &player)
		<-player.ch
	} else {
		player.otherPlayer = waiting
		waiting = nil
		player.otherPlayer[0].otherPlayer = []*Player{player.otherPlayer[1], &player}
		player.otherPlayer[1].otherPlayer = []*Player{player.otherPlayer[0], &player}
		player.otherPlayer[0].ch <- "Play\n"
		player.otherPlayer[1].ch <- "Play\n"
	}
	//End lock
	io.WriteString(player.conn, fmt.Sprintf("%s\n", player.otherPlayer[0].username))
	io.WriteString(player.conn, fmt.Sprintf("%s\n", player.otherPlayer[1].username))
	
	for i := 0; i < 5; i++ {
		io.WriteString(player.conn, "Question1\n")
		io.WriteString(player.conn, "Opt1\n")
		io.WriteString(player.conn, "Opt2\n")
		io.WriteString(player.conn, "Opt3\n")
		io.WriteString(player.conn, "Opt4\n")
		answer, _, _ := bufc.ReadLine()
		answerStr := string(answer)
		fmt.Println(answerStr)
		//TODO
		io.WriteString(player.conn, "1\n")
		players := Players{&player, player.otherPlayer[0], player.otherPlayer[1]}
		sort.Sort(players)
		for i, p := range players {
			
			if p == &player {
				if i != 2 {
					<-player.ch
					players[i+1].ch <- "Done\n"
				} else {
					players[0].ch <- "Done\n"
					<-player.ch
				}
				break
			}
		}
	}
	io.WriteString(player.conn, "25\n")
}

func main() {
	//players = make([]Player, 0)
	database, _ = sql.Open("mysql", "root:123@/apvquiz")
	//Open question table
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
		if err != nil {
			log.Println(err)
			continue
		}
		go handleClient(conn)
	}
}
