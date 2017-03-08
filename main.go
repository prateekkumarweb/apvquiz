package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/http"
)

// Database for handling connections to MySQL database
var database *sql.DB

var err error

type Question struct {
	Question string
	Option1  string
	Option2  string
	Option3  string
	Option4  string
	Answer   int
	Subject  string
}

type Questions struct {
	Questions []Question
}

// main function that runs the server on port 8000
func main() {
	// Command line flags
	port := flag.String("http", ":8000", "Port on which the server is to be hosted")
	mysql := flag.String("mysql", "root:123", "Username and password used to connect to the database")
	db := flag.String("db", "apvquiz", "MySql database")
	init := flag.Bool("init", false, "Set true if questions are to be initialized from question.yml file")
	flag.Parse()

	// Initialize waiting struct which maintains the players waiting for other players to join
	waiting.players = make(map[string]Players)

	// Open a connection to mysql database
	database, err = sql.Open("mysql", *mysql+"@/"+*db)
	if err != nil {
		// TODO handle error
		panic(err.Error())
		return
	}
	err = database.Ping()
	if err != nil {
		// TODO handle error
		panic(err.Error())
		return
	}
	defer database.Close()

	database.Exec(`CREATE TABLE IF NOT EXISTS users (
		id int auto_increment,
		username varchar(180) not null unique,
		password varchar(180) not null,
		points int DEFAULT 0,
		games int DEFAULT 0,
		contributions int DEFAULT 0,
		primary key (id)
	)`)

	topics := []string{"harrypotter", "gk", "movies", "anime", "science", "sports", "got", "trivia", "computers"}
	for _, t := range topics {
		go func(topic string) {
			database.Exec("CREATE TABLE IF NOT EXISTS " + topic + " (id int auto_increment, question text not null, option1 varchar(180) not null, option2 varchar(180) not null, option3 varchar(180) not null, option4 varchar(180) not null, answer int not null, primary key (id))")
		}(t)
	}
	if *init {
		data, _ := ioutil.ReadFile("questions.yml")

		questions := Questions{}

		err = yaml.Unmarshal(data, &questions)
		if err != nil {
			fmt.Println("err")
			fmt.Println(err)
			return
		}
		for _, q := range questions.Questions {
			go func(question Question) {
				database.Exec("INSERT INTO "+question.Subject+" VALUES(0, ?, ?, ?, ?, ?, ?)", question.Question, question.Option1, question.Option2, question.Option3, question.Option4, question.Answer)
			}(q)
		}
	}

	//
	// r := rand.New(rand.NewSource(99))
	// topics := []string{"harrypotter", "gk", "movies", "anime", "science", "sports", "got", "trivia", "computers"}
	// for _, t := range topics {
	// 	go func(topic string) {
	// 		var count int
	// 		database.Exec("CREATE TABLE IF NOT EXISTS " + topic + " (id int auto_increment, question text not null, option1 varchar(180) not null, option2 varchar(180) not null, option3 varchar(180) not null, option4 varchar(180) not null, answer int not null, primary key (id))")
	// 		database.QueryRow("SELECT COUNT(*) FROM " + topic).Scan(&count)
	// 		if count < 25 {
	// 			for i := 0; i < 25; i++ {
	// 				_, err := database.Exec(fmt.Sprintf("INSERT INTO %s VALUES (0, '%s', '%s', '%s', '%s', '%s', %v)", topic, "question"+fmt.Sprintf("%v", i), "option"+fmt.Sprintf("%v", i)+"-1", "option"+fmt.Sprintf("%v", i)+"-2", "option"+fmt.Sprintf("%v", i)+"-3", "option"+fmt.Sprintf("%v", i)+"-4", r.Intn(4)+1))
	// 				if err != nil {
	// 					fmt.Println(err)
	// 				}
	// 			}
	// 		}
	// 	}(t)
	// }

	// updrader object containing configurations for upgrading a http connection to ws connection
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	// Hello World handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Welcome to APVQuiz Server! Read the documentation for details.")
	})

	// Login handler
	// Receives username and password from a post request and verifies the credentials
	http.HandleFunc("/login", login)

	// Signup handler
	// Receives username and password from a post request and creates a new user and adds to the database
	http.HandleFunc("/signup", signup)

	// PlayerDetails handler
	// Receives username and password from a post request and send back the details of the player
	http.HandleFunc("/details", playerDetails)

	// Contribution handler
	// Receives question from user and adds them to the database
	http.HandleFunc("/contri", contribute)

	// Play Game handler
	// Creates a go routine that handles the game play of that client
	http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		// Upgarde the http connection to websocket connection
		conn, err := upgrader.Upgrade(w, r, nil)

		// If error while upgarding break the connection
		if err != nil {
			// TODO Log the error
			fmt.Println(err)
			return
		}
		// Handle the client by creating a goroutine
		go handleClient(conn)
	})

	// Host the server on port given as command line flag
	http.ListenAndServe(*port, nil)
}
