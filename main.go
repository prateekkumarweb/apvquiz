package main

import (
	//"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	//"log"
	//"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	//"math/rand"
	"github.com/gorilla/websocket"
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
	conn        *websocket.Conn
	username    string
	password    string
	ch          chan string
	otherPlayer []*Player
	score       int
}

var players []Player

var waiting []*Player
var waitingMutex sync.Mutex

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

func handleClient(c *websocket.Conn) {
	msgType, username, _ := c.ReadMessage()
	msgType, password, _ := c.ReadMessage()
	player := Player{c, string(username), string(password), make(chan string), nil, 0}
	if !validateUser(player) {
		player.conn.WriteMessage(msgType, []byte("Invalid\n"))
		return
	}
	player.conn.WriteMessage(msgType, []byte("Valid\n"))

	waitingMutex.Lock()
	if len(waiting) < 2 {
		waiting = append(waiting, &player)
		waitingMutex.Unlock()
		<-player.ch
	} else {
		player.otherPlayer = waiting
		waiting = nil
		waitingMutex.Unlock()
		player.otherPlayer[0].otherPlayer = []*Player{player.otherPlayer[1], &player}
		player.otherPlayer[1].otherPlayer = []*Player{player.otherPlayer[0], &player}
		player.otherPlayer[0].ch <- "Play\n"
		player.otherPlayer[1].ch <- "Play\n"
		// r := rand.New(rand.NewSource(99))
		// rows, _ := database.Query("SELECT COUNT(*) FROM questions")
		// fmt.Println(rows)
		// rows.Close()

	}

	for i := 0; i < 5; i++ {
		player.conn.WriteMessage(msgType, []byte(fmt.Sprintf("%s$#$%s$#$%s$#$%s$#$%s$#$%v$#$%v$#$%s$#$%v$#$%s$#$%v", "Question1", "Option1", "Option2", "Option3", "Option4", 1, player.score, player.otherPlayer[0].username, player.otherPlayer[0].score, player.otherPlayer[1].username, player.otherPlayer[1].score)))

		_, answer, _ := player.conn.ReadMessage()
		answerStr := string(answer)
		fmt.Println(answerStr)
		//TODO
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

	player.conn.WriteMessage(msgType, []byte("25"))
}

func play(w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Client connected...")
	go handleClient(conn)
}

// func handleClientWS(conn *websocket.Conn) {
// 	msgType, msg, _ := conn.ReadMessage()
// 	conn.WriteMessage(msgType, []byte("Hello\n"))
// 	fmt.Println(msg)
// }

func main() {
	//players = make([]Player, 0)
	database, _ = sql.Open("mysql", "root:123@/apvquiz")
	//Open question table
	err := database.Ping()
	if err != nil {
		fmt.Println(err)
		return
	}

	func() {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		http.HandleFunc("/", hello)
		http.HandleFunc("/login", login)
		http.HandleFunc("/signup", signup)
		http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
			play(w, r, upgrader)
		})
		http.ListenAndServe(":8000", nil)
	}()

	// ln, err := net.Listen("tcp", ":6000")
	// if err != nil {
	// 	log.Fatal(err)
	// 	return
	// }

	// for {
	// 	conn, err := ln.Accept()
	// 	if err != nil {
	// 		log.Println(err)
	// 		continue
	// 	}
	// 	go handleClient(conn)
	// }
}
