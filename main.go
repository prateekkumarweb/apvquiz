package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"io"
	"math/rand"
	"net/http"
)

// Database for handling connections to MySQL database
var database *sql.DB

// main function that runs the server on port 8000
func main() {
	go initialize()
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Welcome to APVQuiz Server! Read the documentation for details.")
	})
	http.HandleFunc("/login", login)
	http.HandleFunc("/signup", signup)
	http.HandleFunc("/details", playerDetails)
	http.HandleFunc("/contri", contribute)
	http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Client connected...")
		go handleClient(conn)
	})
	http.ListenAndServe(":8000", nil)
}

// initialize function that initializes all the global variables
// and puts dummy data into database
func initialize() {
	waiting.players = make(map[string]Players)
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
	topics := []string{"harrypotter", "gk", "movies", "anime", "science", "sports", "got", "trivia", "computers"}
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
