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
	r.HandleFunc("/courses", courseHandler).Methods("GET")
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/refresh", refreshHandler).Methods("POST")
	r.HandleFunc("/search", searchHandler).Methods("POST")

	//DEV: this will be removed once I figure out a better way to have a dev version
	allowedOrigins := handlers.AllowedOrigins([]string{"http://courseadvysr.com", "https://courseadvysr.com", "http://localhost:3000"})
	allowCredentials := handlers.AllowCredentials()
	allowedHeaders := handlers.AllowedHeaders([]string{"content-type", "X-Requested-With", "Origin", "Accept", "X-PINGOTHER"})

	srv := &http.Server{
		Handler:      handlers.CORS(allowedOrigins, allowedHeaders, allowCredentials)(r),
		Addr:         "127.0.0.1:1337",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server started up and running at: %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())

}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var info LoginInformation

	err := json.NewDecoder(r.Body).Decode(&info)

	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		log.Print(err)
		return
	}

	//We expect only a username and password to be sent to us.

	if info.Password == "" || info.Username == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	//we'll need to verify/clear/not screw around with sql injections.

	response, err := CheckPasswordHash(info.Password, info.Username)

	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if response {

		token, _ := GenerateKey(info.Username)
		//TODO: actually stop being lazy and implement a refresh timer
		expTime := time.Now().Add(5 * time.Hour)

		http.SetCookie(w, &http.Cookie{Name: "token",
			Value: token, Path: "/", Expires: expTime})

	} else {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

}

func courseHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = CheckToken(c.Value)
	if err != nil {

		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	log.Print("Accessed Courses")

	enc := json.NewEncoder(w)
	//we'll have to use a dotfile to access the password but this is just to see if it works
	// const connStr = "sslmode=disable user=vysr password=aPassWord dbname=vysr"

	returnedInfo := GetCourses()

	enc.Encode(returnedInfo)
}

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: handle refreshes and stop being a nerd and allowing 5hrs on a single log-in.
	c, err := r.Cookie("token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = CheckToken(c.Value)
	if err != nil {

		w.WriteHeader(http.StatusUnauthorized)
		return
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = CheckToken(c.Value)
	if err != nil {

		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//wrt to the check, we have to make sure that we store this token and who it belongs to make sure it belongs to the right person.

	var info SearchQuery

	err = json.NewDecoder(r.Body).Decode(&info)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	queryReutrns := SearchCourses(info)

	enc := json.NewEncoder(w)

	enc.Encode(queryReutrns)

}
