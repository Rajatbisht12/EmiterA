package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	// "fmt"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var ctx = context.Background()
var redisClient *redis.Client

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clients = make(map[string]*websocket.Conn)
	mutex   sync.RWMutex
)

type ScoreUpdate struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Score    int    `json:"score"`
	Previous int    `json:"previous"`
}

type Card struct {
	Type string `json:"type"`
}

type Game struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Deck      []Card `json:"deck"`
	HasDefuse bool   `json:"hasDefuse"`
	Points    int    `json:"points"`
}

type Player struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
}

var (
	redisURL       = os.Getenv("REDIS_URL")
	port           = os.Getenv("PORT")
	allowedOrigins = os.Getenv("ALLOWED_ORIGINS")
)

func initRedisClient() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL environment variable is required")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Invalid Redis URL: %v", err)
	}

	client := redis.NewClient(opt)

	// Verify connection
	_, err = client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	log.Println("Successfully connected to Redis")
	return client
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default to 8080 if not set
	}

	// Initialize Redis client
	redisClient = initRedisClient()
	defer redisClient.Close()

	// Test Redis connection
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully!")

	err = redisClient.Do(ctx, "SELECT", 0).Err()
	if err != nil {
		log.Fatalf("Failed to switch database: %v", err)
	}
	log.Println("Successfully switched to database 1")

	// Set up CORS middleware
	handler := corsMiddleware(setupRoutes())

	// Start the server
	log.Printf("Server starting on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/game/new", handleNewGame)
	mux.HandleFunc("/api/game/draw", handleDrawCard)
	mux.HandleFunc("/api/leaderboard", handleLeaderboard)
	mux.HandleFunc("/ws", handleWebSocket)
	mux.HandleFunc("/api/game/resume", handleResumeGame)
	return mux
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Get username from query parameter
	username := r.URL.Query().Get("username")
	if username == "" {
		conn.Close()
		return
	}

	// Store connection
	mutex.Lock()
	clients[username] = conn
	mutex.Unlock()

	// Clean up on disconnect
	defer func() {
		mutex.Lock()
		delete(clients, username)
		mutex.Unlock()
		conn.Close()
	}()

	// Keep connection alive and handle incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func handleResumeGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		GameID string `json:"gameId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get game from Redis
	gameData, err := redisClient.Get(ctx, "game:"+request.GameID).Result()
	if err != nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	var game Game
	if err := json.Unmarshal([]byte(gameData), &game); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleNewGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create or update user in Redis
	err := redisClient.HSetNX(ctx, "players", request.Username, "0").Err()
	if err != nil {
		log.Printf("Error creating user: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	game := createNewGame(request.Username)

	// Store game in Redis
	gameData, err := json.Marshal(game)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = redisClient.Set(ctx, "game:"+game.ID, gameData, 24*time.Hour).Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func handleDrawCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		GameID string `json:"gameId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get game from Redis
	gameData, err := redisClient.Get(ctx, "game:"+request.GameID).Result()
	if err != nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	var game Game
	if err := json.Unmarshal([]byte(gameData), &game); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(game.Deck) == 0 {
		http.Error(w, "No cards left in deck", http.StatusBadRequest)
		return
	}

	// Draw a card
	card := game.Deck[0]
	game.Deck = game.Deck[1:]

	var result struct {
		Game   Game   `json:"game"`
		Card   Card   `json:"card"`
		Status string `json:"status"`
	}
	result.Card = card
	result.Game = game

	// Handle card effects with modified scoring
	switch card.Type {
	case "bomb":
		if game.HasDefuse {
			game.HasDefuse = false
			result.Status = "defused"
			// Award points for successfully defusing a bomb
			updatePlayerScore(game.Username, 1)
		} else {
			result.Status = "exploded"
			// Penalty for explosion, but won't go below 0
			updatePlayerScore(game.Username, -1)
		}
	case "defuse":
		game.HasDefuse = true
		result.Game.HasDefuse = true
		result.Status = "continue"
	case "shuffle":
		game = createNewGame(game.Username)
		result.Status = "shuffled"
		result.Game = game
	case "cat":
		if len(game.Deck) == 0 {
			result.Status = "won"
			// Award points for winning
			updatePlayerScore(game.Username, 2)
		} else {
			result.Status = "continue"
		}
	}

	// Save updated game state
	updatedGameData, err := json.Marshal(game)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = redisClient.Set(ctx, "game:"+game.ID, updatedGameData, 24*time.Hour).Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all player scores from Redis
	players, err := redisClient.HGetAll(ctx, "players").Result()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var leaderboard []Player
	for username, score := range players {
		points := 0
		if score != "" {
			points = parseInt(score)
		}
		leaderboard = append(leaderboard, Player{
			Username: username,
			Score:    points,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leaderboard)
}

func createNewGame(username string) Game {
	// Create a new deck of cards
	deck := []Card{
		{Type: "cat"},
		{Type: "cat"},
		{Type: "defuse"},
		{Type: "shuffle"},
		{Type: "bomb"},
	}

	// Shuffle the deck
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })

	return Game{
		ID:        generateGameID(),
		Username:  username,
		Deck:      deck,
		HasDefuse: false,
		Points:    0,
	}
}

func updatePlayerScore(username string, points int) {
	// Get current score
	currentScore, err := redisClient.HGet(ctx, "players", username).Int()
	if err != nil {
		log.Printf("Error getting current score: %v", err)
		currentScore = 0
	}

	// Calculate new score, ensuring it doesn't go below 0
	newScore := currentScore + points
	if newScore < 0 {
		newScore = 0
	}

	// Update score in Redis
	err = redisClient.HSet(ctx, "players", username, newScore).Err()
	if err != nil {
		log.Printf("Error updating score: %v", err)
		return
	}

	// Broadcast score update to all clients
	update := ScoreUpdate{
		Type:     "score_update",
		Username: username,
		Score:    newScore,
		Previous: currentScore,
	}

	mutex.RLock()
	for _, conn := range clients {
		err := conn.WriteJSON(update)
		if err != nil {
			log.Printf("Error broadcasting score update: %v", err)
		}
	}
	mutex.RUnlock()
}

func generateGameID() string {
	return time.Now().Format("20060102150405")
}

func parseInt(s string) int {
	var result int
	for _, ch := range s {
		result = result*10 + int(ch-'0')
	}
	return result
}
