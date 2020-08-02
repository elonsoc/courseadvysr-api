package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/courses/", courseHandler).Methods("GET")
	var allowedOrigins []string
	allowedOrigins = append(allowedOrigins, "http://courseadvysr.com")
	allowedOrigins = append(allowedOrigins, "https://courseadvysr.com")

	srv := &http.Server{
		Handler:      handlers.CORS(handlers.AllowedOrigins(allowedOrigins))(r),
		Addr:         "127.0.0.1:1337",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server started up and running at: %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())

}

func courseHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Accessed Courses")

	enc := json.NewEncoder(w)
	//we'll have to use a dotfile to access the password but this is just to see if it works
	// const connStr = "sslmode=disable user=vysr password=aPassWord dbname=vysr"

	returnedInfo := GetCourses()

	enc.Encode(returnedInfo)
}
