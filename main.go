package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
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
	Games  int
	Points int
	Contri int
}

func playerDetails(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	w.Header().Set("Content-Type", "application/json")
	rows, err := database.Query(fmt.Sprintf("SELECT * FROM users where username='%s' AND password='%s'", username, password))
	defer rows.Close()
	if err != nil {
		data := PlayerDetails{false, 0, 0, 0}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	var id, points, games, contributions int
	for rows.Next() {
		rows.Scan(&id, &username, &password, &points, &games, &contributions)
	}
	data := PlayerDetails{true, games, points, contributions}
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

	msgs := make(chan string)

	msgType, username, _ := c.ReadMessage()
	msgType, password, _ := c.ReadMessage()
	player := Player{c, string(username), string(password), make(chan string), nil, 0}
	if !validateUser(player) {
		player.conn.WriteMessage(msgType, []byte("Invalid\n"))
		return
	}
	var questions [5]string
	msgType, t, _ := c.ReadMessage()
	topic := string(t)

	go func() {
		for {
			_, msg, err := c.ReadMessage()
			if err != nil || string(msg) == "closed" {
				for _, p := range player.otherPlayer {
					p.conn.WriteMessage(msgType, []byte("Opponent has left the game"))
					p.conn.Close()
				}
				waitingMutex.Lock()
				for i, p := range waiting[topic] {
					if p == &player {
						waiting[topic] = append(waiting[topic][:i], waiting[topic][i+1:]...)
						break
					}
				}
				waitingMutex.Unlock()
				player.conn.Close()
				return
			}
			msgs <- string(msg)
		}
	}()

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
		rows, _ := database.Query("SELECT COUNT(*) FROM " + topic)
		defer rows.Close()
		var count int
		for rows.Next() {
			rows.Scan(&count)
		}
		perm := r.Perm(count)
		for i, _ := range questions {
			func() {
				rows, _ := database.Query(fmt.Sprintf("SELECT * FROM %s WHERE id=%v", topic, perm[i]+1))
				defer rows.Close()
				var id, answer int
				var question, option1, option2, option3, option4 string
				for rows.Next() {
					rows.Scan(&id, &question, &option1, &option2, &option3, &option4, &answer)
				}
				questions[i] = fmt.Sprintf("%s@#@%s@#@%s@#@%s@#@%s@#@%v", question, option1, option2, option3, option4, answer)
				player.otherPlayer[0].ch <- questions[i]
				player.otherPlayer[1].ch <- questions[i]
			}()
		}
	}

	for i, _ := range questions {
		player.conn.WriteMessage(msgType, []byte(fmt.Sprintf("%s@#@%v@#@%s@#@%v@#@%s@#@%v", questions[i], player.score, player.otherPlayer[0].username, player.otherPlayer[0].score, player.otherPlayer[1].username, player.otherPlayer[1].score)))

		answerStr := <-msgs
		timeStr := <-msgs

		if answerStr == "closed" || timeStr == "closed" {
			fmt.Println("closed.......//")
		}

		if answerStr == "1" && i != 4 {
			score, _ := strconv.Atoi(timeStr)
			player.score += score
		} else if answerStr == "1" && i == 4 {
			score, _ := strconv.Atoi(timeStr)
			player.score += score * 2
		}
		time.Sleep(2 * time.Second)
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
	rows, err := database.Query("SELECT * FROM users WHERE username='" + player.username + "'")
	defer rows.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	var id, points, games, contributions int
	for rows.Next() {
		rows.Scan(&id, &username, &password, &points, &games, &contributions)
	}
	fmt.Println(fmt.Sprintf("%s B== %v == %v", player.username, games, points))
	games += 1
	points += player.score
	fmt.Println(fmt.Sprintf("%s A== %v == %v", player.username, games, points))
	_, err = database.Query(fmt.Sprintf("UPDATE users SET games=%v, points=%v WHERE username=\"%s\"", games, points, username))
	if err != nil {
		fmt.Println(err)
	}
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
	database, _ = sql.Open("mysql", "root:123@/apvquiz")
	err := database.Ping()
	if err != nil {
		fmt.Println(err)
		return
	}
	rows, _ := database.Query(`CREATE TABLE IF NOT EXISTS users (
		id int auto_increment,
		username varchar(180) not null unique,
		password varchar(180) not null,
		points int DEFAULT 0,
		games int DEFAULT 0,
		contributions int DEFAULT 0,
		primary key (id)
	)`)
	rows.Close()
	r := rand.New(rand.NewSource(99))
	topics := []string{"harrypotter", "gk", "movies", "anime", "science", "cricket", "got", "trivia", "computerscience"}
	for _, t := range topics {
		go func(topic string) {
			rows, _ := database.Query("CREATE TABLE IF NOT EXISTS " + topic + " (id int auto_increment, question text not null, option1 varchar(180) not null, option2 varchar(180) not null, option3 varchar(180) not null, option4 varchar(180) not null, answer int not null, primary key (id))")
			rows.Close()
			rows, _ = database.Query("SELECT COUNT(*) FROM " + topic)
			var count int
			for rows.Next() {
				rows.Scan(&count)
			}
			rows.Close()
			if count < 25 {
				for i := 0; i < 25; i++ {
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

	go initialize()

	func() {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
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
