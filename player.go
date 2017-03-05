package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Player struct {
	conn        *websocket.Conn
	username    string
	password    string
	ch          chan string
	otherPlayer []*Player
	score       int
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

var waiting struct {
	sync.Mutex
	players map[string]Players
}

func validatePlayer(player Player) bool {
	username := player.username
	password := player.password
	return validateUser(username, password)
}

func validateUser(username, password string) bool {
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

func playerDetails(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	w.Header().Set("Content-Type", "application/json")
	rows, err := database.Query(fmt.Sprintf("SELECT * FROM users where username='%s' AND password='%s'", username, password))
	defer rows.Close()
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
	var id, points, games, contributions int
	for rows.Next() {
		rows.Scan(&id, &username, &password, &points, &games, &contributions)
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

func handleClient(c *websocket.Conn) {

	msgs := make(chan string)

	msgType, username, _ := c.ReadMessage()
	msgType, password, _ := c.ReadMessage()
	player := Player{c, string(username), string(password), make(chan string), nil, 0}
	if !validatePlayer(player) {
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
				waiting.Lock()
				for i, p := range waiting.players[topic] {
					if p == &player {
						waiting.players[topic] = append(waiting.players[topic][:i], waiting.players[topic][i+1:]...)
						break
					}
				}
				waiting.Unlock()
				player.conn.Close()
				return
			}
			msgs <- string(msg)
		}
	}()

	waiting.Lock()
	if len(waiting.players[topic]) < 2 {
		if waiting.players[topic] == nil {
			waiting.players[topic] = make([]*Player, 0)
		}
		waiting.players[topic] = append(waiting.players[topic], &player)
		waiting.Unlock()
		<-player.ch
		for i, _ := range questions {
			questions[i] = <-player.ch
		}
	} else {
		player.otherPlayer = waiting.players[topic]
		waiting.players[topic] = nil
		waiting.Unlock()
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
