package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Movie struct {
	ID           int
	Title, Genre string
	Year         int
	Rating       float64
}

var movies = []Movie{

	{ID: 1, Title: "The Matrix", Genre: "Sci-Fi", Year: 1999, Rating: 8.7},

	{ID: 2, Title: "Interstellar", Genre: "Sci-Fi", Year: 2014, Rating: 8.7},

	{ID: 3, Title: "The Mask", Genre: "Comedy", Year: 1994, Rating: 6.9},

	{ID: 4, Title: "Dune: Part Two", Genre: "Sci-Fi", Year: 2024, Rating: 8.5},

	{ID: 5, Title: "The Grand Budapest Hotel", Genre: "Comedy", Year: 2014, Rating: 8.1},
}
var nextID = 5
var mutex sync.Mutex

func send(responseWriter http.ResponseWriter, status int, value any) {
	responseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	responseWriter.WriteHeader(status)
	json.NewEncoder(responseWriter).Encode(value)
}

func find(id int) int {
	for index, movie := range movies {
		if movie.ID == id {
			return index
		}
	}
	return -1
}

func home(responseWriter http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(responseWriter, request)
		return
	}
	fmt.Fprintln(responseWriter, "Movie Catalog API: /movies, /movies/{id}, /movies/genre/{genre}")
}

// Фильтры по году и рейтингу можно использовать одновременно.
func list(responseWriter http.ResponseWriter, request *http.Request) {
	min := 0.0
	year := 0
	value := request.URL.Query().Get("minRating")
	if value != "" {
		numberValue, operationError := strconv.ParseFloat(value, 64)
		if operationError != nil || math.IsNaN(numberValue) || numberValue < 0 || numberValue > 10 {
			http.Error(responseWriter, "Invalid rating", 400)
			return
		}
		min = numberValue
	}
	value = request.URL.Query().Get("year")
	if value != "" {
		numberValue, operationError := strconv.Atoi(value)
		if operationError != nil || numberValue <= 0 {
			http.Error(responseWriter, "Invalid year", 400)
			return
		}
		year = numberValue
	}
	mutex.Lock()
	defer mutex.Unlock()
	result := []Movie{}
	for _, movie := range movies {
		if movie.Rating >= min && (year == 0 || movie.Year == year) {
			result = append(result, movie)
		}
	}
	send(responseWriter, 200, result)
}

func get(responseWriter http.ResponseWriter, request *http.Request) {
	id, operationError := strconv.Atoi(request.PathValue("id"))
	if operationError != nil {
		http.Error(responseWriter, "Invalid ID", 400)
		return
	}
	mutex.Lock()
	defer mutex.Unlock()
	index := find(id)
	if index < 0 {
		http.NotFound(responseWriter, request)
		return
	}
	send(responseWriter, 200, movies[index])
}

func genre(responseWriter http.ResponseWriter, request *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	result := []Movie{}
	for _, movie := range movies {
		if strings.EqualFold(movie.Genre, request.PathValue("genre")) {
			result = append(result, movie)
		}
	}
	send(responseWriter, 200, result)
}

// Проверяем данные фильма и присваиваем новый ID.
func create(responseWriter http.ResponseWriter, request *http.Request) {
	var movie Movie
	if json.NewDecoder(request.Body).Decode(&movie) != nil || strings.TrimSpace(movie.Title) == "" || movie.Year <= 0 || movie.Rating < 0 || movie.Rating > 10 {
		http.Error(responseWriter, "Invalid movie", 400)
		return
	}
	mutex.Lock()
	defer mutex.Unlock()
	nextID = nextID + 1
	movie.ID = nextID
	movies = append(movies, movie)
	send(responseWriter, 201, movie)
}

func deleteMovie(responseWriter http.ResponseWriter, request *http.Request) {
	id, operationError := strconv.Atoi(request.PathValue("id"))
	if operationError != nil {
		http.Error(responseWriter, "Invalid ID", 400)
		return
	}
	mutex.Lock()
	defer mutex.Unlock()
	index := find(id)
	if index < 0 {
		http.NotFound(responseWriter, request)
		return
	}
	movies = append(movies[:index], movies[index+1:]...)
	responseWriter.WriteHeader(204)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", home)
	mux.HandleFunc("GET /movies", list)
	mux.HandleFunc("GET /movies/{id}", get)
	mux.HandleFunc("GET /movies/genre/{genre}", genre)
	mux.HandleFunc("POST /movies", create)
	mux.HandleFunc("DELETE /movies/{id}", deleteMovie)
	log.Println("http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
