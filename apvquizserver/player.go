package apvquizserver

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Player struct to store player object
type Player struct {
	sync.Mutex  // Lock while writing to conn
	conn        *websocket.Conn
	username    string
	password    string
	ch          chan string // channel through different client communicate
	otherPlayer []*Player   // slice of other players
	score       int         // score of the current player
}

// Players type to store slice of players (like a alias)
// Functions Len, Less and Swap are defined so that players can be sorted based on usernames
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

// waiting struct to store players waiting for the other players to join
var waiting struct {
	sync.Mutex
	players map[string]Players // map from topic to waiting players
}

// validatePlayer function validates player from the database
func ValidatePlayer(player Player) bool {
	username := player.username
	password := player.password
	return ValidateUser(username, password)
}

// validateUser function validates username and password
func ValidateUser(username, password string) bool {
	// Get hashed password from database
	var dbPassword string
	err := database.QueryRow("SELECT password FROM users WHERE username=?", username).Scan(&dbPassword)
	if err != nil {
		return false
	}
	// compare given password with its hash and validate
	err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
	if err != nil {
		return false
	}
	return true
}

// playerDeatils handler sends points, games and contributions of the player
func PlayerDetails(w http.ResponseWriter, r *http.Request) {

	// Get username and password from the request object
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Set content type of response
	w.Header().Set("Content-Type", "application/json")
	var id, points, games, contributions int
	err := database.QueryRow("SELECT * FROM users where username=?", username).Scan(&id, &username, &password, &points, &games, &contributions)
	if err != nil {
		data := struct {
			Status bool
			Games  int
			Points int
			Contri int
		}{false, 0, 0, 0}
		js, _ := json.Marshal(data)
		w.Write(js)
		return
	}
	data := struct {
		Status bool
		Games  int
		Points int
		Contri int
	}{true, games, points, contributions}
	js, _ := json.Marshal(data)
	w.Write(js)
}

// write function obtains a lock on the player and sends the message the client
// Lock is obtained to avoid concurrnt writes
func (player *Player) write(msgType int, message string) {
	player.Lock()
	player.conn.WriteMessage(msgType, []byte(message))
	player.Unlock()
}

// handleClient function is the go routine that runs while user is playing game
// One instance of this function per connection runs and communicate with
// each other while game is running
func HandleClient(c *websocket.Conn) {

	// msgs channel where messages from the user will be added
	msgs := make(chan string)

	// Ask user for username and password and validate them and create a player object
	msgType, username, _ := c.ReadMessage()
	msgType, password, _ := c.ReadMessage()
	player := Player{sync.Mutex{}, c, string(username), string(password), make(chan string), nil, 0}
	if !ValidatePlayer(player) {
		player.write(msgType, "Invalid\n")
		return
	}

	// List of questions
	var questions [5]string

	// Read topic from the user
	msgType, t, _ := c.ReadMessage()
	topic := string(t)

	// This goroutine reads messages sent by the user and adds them to msgs channel
	// from where the parent function will read the input
	go func() {
		for {
			// Wait until a message arrives
			_, msg, err := c.ReadMessage()

			// If one of the client has disconnected
			if err != nil || string(msg) == "closed" {
				player.score = 0
				// Send to other players that this player has diconnected and close their connections
				for _, p := range player.otherPlayer {
					p.write(msgType, "Opponent has left the game")
					p.conn.Close()
				}

				// If this player was there in the waiting list then remove him
				waiting.Lock()
				for i, p := range waiting.players[topic] {
					if p == &player {
						waiting.players[topic] = append(waiting.players[topic][:i], waiting.players[topic][i+1:]...)
						break
					}
				}
				waiting.Unlock()

				// Close the players connection
				player.conn.Close()
				return
			}

			// otherwise add the message to the channel
			msgs <- string(msg)
		}
	}()

	waiting.Lock()
	// If number of players waiting is less than two then add this player to waiting list
	if len(waiting.players[topic]) < 2 {
		if waiting.players[topic] == nil {
			waiting.players[topic] = make([]*Player, 0)
		}
		waiting.players[topic] = append(waiting.players[topic], &player)
		waiting.Unlock()
		// wait until another player send a success message
		<-player.ch
		// Get the questions from another player
		for i, _ := range questions {
			questions[i] = <-player.ch
		}
	} else {
		// If number of players are 3 then remove them from waiting queue and add
		// them to otherPlayers list
		player.otherPlayer = waiting.players[topic]
		waiting.players[topic] = nil
		waiting.Unlock()
		player.otherPlayer[0].otherPlayer = []*Player{player.otherPlayer[1], &player}
		player.otherPlayer[1].otherPlayer = []*Player{player.otherPlayer[0], &player}

		// Send a success message to other players
		player.otherPlayer[0].ch <- "Play\n"
		player.otherPlayer[1].ch <- "Play\n"

		// Select random questions in the given topic
		r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		var count int
		database.QueryRow("SELECT COUNT(*) FROM " + topic).Scan(&count)
		perm := r.Perm(count)

		// Select the questions and send them to other players
		for i, _ := range questions {
			func() {
				var id, answer int
				var question, option1, option2, option3, option4 string
				database.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE id=%v", topic, perm[i]+1)).Scan(&id, &question, &option1, &option2, &option3, &option4, &answer)
				questions[i] = fmt.Sprintf("%s@#@%s@#@%s@#@%s@#@%s@#@%v", question, option1, option2, option3, option4, answer)
				player.otherPlayer[0].ch <- questions[i]
				player.otherPlayer[1].ch <- questions[i]
			}()
		}
	}

	// Iterate over all the questions
	for i, _ := range questions {

		// Send the question to the player along with scores of other players
		player.write(msgType, fmt.Sprintf("%s@#@%v@#@%s@#@%v@#@%s@#@%v", questions[i], player.score, player.otherPlayer[0].username, player.otherPlayer[0].score, player.otherPlayer[1].username, player.otherPlayer[1].score))

		// Receive the answer and time from client
		answerStr := <-msgs
		timeStr := <-msgs

		// The client sends one if the user has answered correctly
		// If client has answered correctly, give scores accordingly
		if answerStr == "1" && i != 4 {
			score, _ := strconv.Atoi(timeStr)
			player.score += score
		} else if answerStr == "1" && i == 4 {
			score, _ := strconv.Atoi(timeStr)
			player.score += score * 2
		}

		// Wait for 2 seconds so that user can see the results
		time.Sleep(2 * time.Second)

		// Sync with other players until they have answered correctly or timer is up
		//For syncing with other players a barrier is implemented
		//Barrier works as ---
		//An array of players is created by sorting the names of the three players
		//Say the players go as player[0] A, player[1] B, player[2] C
		//   Player[0]		|    Player[1]		|    Player[2]
		//   wait(player[1])	|    wait(player[2])	|    signal(player[0])
		//   signal(player[1])	|    signal(player[2])  |    wait(player[1])
		// Player channels work as barrier here as they are waiting for other players to complete
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

	// At the end of the game send the score
	player.write(msgType, fmt.Sprintf("%v@#@%s@#@%v@#@%s@#@%v", player.score, player.otherPlayer[0].username, player.otherPlayer[0].score, player.otherPlayer[1].username, player.otherPlayer[1].score))

	// Save new points of the player, their number of games in the database
	var id, points, games, contributions int
	err := database.QueryRow("SELECT * FROM users WHERE username=?", username).Scan(&id, &username, &password, &points, &games, &contributions)
	if err != nil {
		fmt.Println(err)
		return
	}
	games += 1
	points += player.score
	_, err = database.Query(fmt.Sprintf("UPDATE users SET games=%v, points=%v WHERE username=\"%s\"", games, points, username))
	if err != nil {
		fmt.Println(err)
	}
}
