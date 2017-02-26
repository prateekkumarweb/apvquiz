package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"math/rand"
	"github.com/gorilla/websocket"
	"time"
	"strconv"
	"regexp"
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
	username := r.FormValue("username")
	password := r.FormValue("password")
	w.Header().Set("Content-Type", "application/json")
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
	username := r.FormValue("username")
	password := r.FormValue("password")
	w.Header().Set("Content-Type", "application/json")
	if !regexp.MustCompile(`[\dA-Za-z]`).MatchString(username) {
		data := Result{false, "Username should contain only alphanumeric characters"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	if len(username) < 4 {
		data := Result{false, "Username should atleast 4 characters"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	if username == "" || password == "" {
		data := Result{false, "Username or password cannot be empty"}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
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
}

type PlayerDetails struct {
	Status bool
	Games int
	Contri int
}

func playerDetails(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	fmt.Println(username+password)
	w.Header().Set("Content-Type", "application/json")
	data := PlayerDetails{true, 5, 10}
	js, _ := json.Marshal(data)
	w.Write(js)
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

var waiting map[string][]*Player
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
	/*if !validateUser(player) {
		player.conn.WriteMessage(msgType, []byte("Invalid\n"))
		return
	}
	player.conn.WriteMessage(msgType, []byte("Valid\n"))*/
	var questions [5]string
	msgType, t, _ := c.ReadMessage()
	topic := string(t)
	fmt.Println(topic)

	waitingMutex.Lock()
	if len(waiting[topic]) < 2 {
		if waiting[topic] == nil {
			waiting[topic] = make([]*Player, 0)
		}
		waiting[topic] = append(waiting[topic], &player)
		waitingMutex.Unlock()
		<-player.ch
		for i, _ := range questions {
			questions[i] = <-player.ch
		}
	} else {
		player.otherPlayer = waiting[topic]
		waiting[topic] = nil
		waitingMutex.Unlock()
		player.otherPlayer[0].otherPlayer = []*Player{player.otherPlayer[1], &player}
		player.otherPlayer[1].otherPlayer = []*Player{player.otherPlayer[0], &player}
		player.otherPlayer[0].ch <- "Play\n"
		player.otherPlayer[1].ch <- "Play\n"
		r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		rows, _ := database.Query("SELECT COUNT(*) FROM "+topic)
		var count int
		for rows.Next() {
			rows.Scan(&count)
		}
		perm := r.Perm(count)
		fmt.Println(perm)
		for i, _ := range questions {
			rows, _ := database.Query(fmt.Sprintf("SELECT * FROM %s WHERE id=%v", topic,perm[i]+1))
			var id, answer int
			var question, option1, option2, option3, option4 string
			for rows.Next() {
				rows.Scan(&id, &question, &option1, &option2, &option3, &option4, &answer)
			}
			questions[i] = fmt.Sprintf("%s@#@%s@#@%s@#@%s@#@%s@#@%v", question, option1, option2, option3, option4, answer)
			player.otherPlayer[0].ch <- questions[i]
			player.otherPlayer[1].ch <- questions[i]
		}
	}

	for i, _ := range questions {
		player.conn.WriteMessage(msgType, []byte(fmt.Sprintf("%s@#@%v@#@%s@#@%v@#@%s@#@%v", questions[i], player.score, player.otherPlayer[0].username, player.otherPlayer[0].score, player.otherPlayer[1].username, player.otherPlayer[1].score)))

		_, answer, _ := player.conn.ReadMessage()
		_, timer, _ := player.conn.ReadMessage()
		answerStr := string(answer)
		timeStr := string(timer)
		if answerStr == "1" && i != 4 {
			score, _ := strconv.Atoi(timeStr)
			player.score  += score
		} else if answerStr == "1" && i == 4{
			score, _ := strconv.Atoi(timeStr)
			player.score  += score*2
		}
		//fmt.Println(answerStr+player.username)
		//TODO
		time.Sleep(3 * time.Second)
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

	player.conn.WriteMessage(msgType, []byte(fmt.Sprintf("%v@#@%s@#@%v@#@%s@#@%v", player.score, player.otherPlayer[0].username, player.otherPlayer[0].score, player.otherPlayer[1].username, player.otherPlayer[1].score)))
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

func initialize() {
	r := rand.New(rand.NewSource(99))
	topics := []string{"harrypotter", "gk", "movies", "anime", "science", "cricket", "got", "trivia"}
	for _, t := range topics {
		go func(topic string) {
			rows, _ := database.Query("CREATE TABLE IF NOT EXISTS " + topic + " (id int auto_increment, question text not null, option1 varchar(180) not null, option2 varchar(180) not null, option3 varchar(180) not null, option4 varchar(180) not null, answer int not null, primary key (id))")
			rows.Close()
			rows, _ = database.Query("SELECT COUNT(*) FROM "+topic)
			var count int
			for rows.Next() {
				rows.Scan(&count)
			}
			rows.Close()
			if count < 25 {
				for i := 0; i<25; i++ {
					rows, err := database.Query(fmt.Sprintf("INSERT INTO %s VALUES (0, '%s', '%s', '%s', '%s', '%s', %v)", topic, "question"+fmt.Sprintf("%v", i), "option"+fmt.Sprintf("%v", i)+"-1", "option"+fmt.Sprintf("%v", i)+"-2", "option"+fmt.Sprintf("%v", i)+"-3", "option"+fmt.Sprintf("%v", i)+"-4", r.Intn(4)+1))
					if err != nil {
						fmt.Println(err)
					}
					rows.Close()
				}
			}
		}(t)
	}
}

func main() {
	waiting = make(map[string][]*Player)
	database, _ = sql.Open("mysql", "root:123@/apvquiz")
	err := database.Ping()
	if err != nil {
		fmt.Println(err)
		return
	}
	go initialize()

	func() {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		http.HandleFunc("/", hello)
		http.HandleFunc("/login", login)
		http.HandleFunc("/signup", signup)
		http.HandleFunc("/details", playerDetails)
		http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
			play(w, r, upgrader)
		})
		http.ListenAndServe(":8000", nil)
	}()
}
