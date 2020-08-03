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
	//We need to also accept the OPTIONS method or we get red stop sign
	r.HandleFunc("/login", loginHandler).Methods("POST")

	allowedOrigins := handlers.AllowedOrigins([]string{"http://courseadvysr.com", "https://courseadvysr.com", "http://localhost:3000"})
	allowCredentials := handlers.AllowCredentials()
	allowedHeaders := handlers.AllowedHeaders([]string{"content-type", "X-Requested-With", "Origin", "Accept", "X-PINGOTHER"})

	//DEV: this will be removed once I figure out a better way to have a dev version

	srv := &http.Server{
		Handler:      handlers.CORS(allowedOrigins, allowedHeaders, allowCredentials)(r),
		Addr:         "127.0.0.1:1337",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server started up and running at: %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())

}

type loginInformation struct {
	Username, Password string
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var info loginInformation

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
		expTime := time.Now().Add(5 * time.Minute)

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
