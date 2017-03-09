package apvquizserver

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
	"sync"
)

// Database for handling connections to MySQL database
var database *sql.DB

var err error

// Question struct stores the question, options, correct answer, subject and contributor
type Question struct {
	Question    string
	Option1     string
	Option2     string
	Option3     string
	Option4     string
	Answer      int
	Subject     string
	Contributor string
}

// Questions struct is used to read questions from yml file to put in database
type Questions struct {
	Questions []Question
}

// Run function runs the server on port specified by flag input
// or uses 8000 as default
func Run() {

	// Read command line flags and parse them
	port := flag.String("http", ":8000", "Port on which the server is to be hosted")
	mysql := flag.String("mysql", "root:123", "Username and password used to connect to the database")
	db := flag.String("db", "apvquiz", "MySql database")
	init := flag.String("init", "", "Add questions to database from given file")
	flag.Parse()

	// Initialize waiting struct which maintains the players waiting for other players to join
	waiting.players = make(map[string]Players)

	// Open a connection to mysql database
	database, err = sql.Open("mysql", *mysql+"@/"+*db)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = database.Ping()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer database.Close() // Close the database at the end

	// Create users table if does not exists in the database
	database.Exec(`CREATE TABLE IF NOT EXISTS users (
		id int auto_increment,
		username varchar(180) not null unique,
		password varchar(180) not null,
		points int DEFAULT 0,
		games int DEFAULT 0,
		contributions int DEFAULT 0,
		primary key (id)
	)`)

	// Create topic wise tables for storing questions
	topics := []string{"harrypotter", "gk", "movies", "anime", "science", "sports", "got", "trivia", "computers"}
	for _, t := range topics {
		go func(topic string) {
			database.Exec("CREATE TABLE IF NOT EXISTS " + topic + " (id int auto_increment, question text not null, option1 varchar(180) not null, option2 varchar(180) not null, option3 varchar(180) not null, option4 varchar(180) not null, answer int not null, primary key (id))")
		}(t)
	}

	// If -init flaf is specified, add questions to the database from the given file
	if *init != "" {

		// Read questions from given yaml file
		data, _ := ioutil.ReadFile(*init)
		questions := Questions{}
		err = yaml.Unmarshal(data, &questions)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Create go routines that will add questions to database and join them at end
		var wg sync.WaitGroup
		for _, q := range questions.Questions {
			wg.Add(1)
			go func(question Question) {
				defer wg.Done()
				database.Exec("INSERT INTO "+question.Subject+" VALUES(0, ?, ?, ?, ?, ?, ?)", question.Question, question.Option1, question.Option2, question.Option3, question.Option4, question.Answer)
				if question.Contributor != "" {
					var id, points, games, contributions int
					var username, password string
					err := database.QueryRow("SELECT * FROM users WHERE username=?", question.Contributor).Scan(&id, &username, &password, &points, &games, &contributions)
					if err != nil {
						fmt.Println(err)
						return
					}
					contributions += 1
					_, err = database.Exec("UPDATE users SET contributions=? WHERE username=?", contributions, username)
					if err != nil {
						fmt.Println(err)
						return
					}
				}

			}(q)
		}
		wg.Wait()
		return
	}

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
	http.HandleFunc("/login", Login)

	// Signup handler
	// Receives username and password from a post request and creates a new user and adds to the database
	http.HandleFunc("/signup", Signup)

	// PlayerDetails handler
	// Receives username and password from a post request and send back the details of the player
	http.HandleFunc("/details", PlayerDetails)

	// Contribution handler
	// Receives question from user and adds them to the database
	http.HandleFunc("/contri", Contribute)

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
		go HandleClient(conn)
	})

	// Host the server on port given as command line flag
	http.ListenAndServe(*port, nil)
}
