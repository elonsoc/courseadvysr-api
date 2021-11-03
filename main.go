package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/courses", courseHandler).Methods("GET")
	r.HandleFunc("/courses/{crn}/", crnCourseHandler).Methods("GET")
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/register", registerHandler).Methods("POST")
	r.HandleFunc("/refresh", refreshHandler).Methods("POST")
	r.HandleFunc("/search", searchHandler).Methods("POST")
	r.HandleFunc("/commit", commitCoursesHandler).Methods("POST")
	r.HandleFunc("/commit", selectedCoursesHandler).Methods("GET")
	r.HandleFunc("/commit", deleteSelectedCoursesHandler).Methods("DELETE")
	r.HandleFunc("/options", getCourseOptionsHandler).Methods("GET")
	r.HandleFunc("/terms", getTermsHandler).Methods("GET")

	//DEV: this will be removed once I figure out a better way to have a dev version
	allowedOrigins := handlers.AllowedOrigins([]string{"http://courseadvysr.com",
		"https://courseadvysr.com", "http://localhost:3000"})
	allowCredentials := handlers.AllowCredentials()
	allowMethods := handlers.AllowedMethods([]string{"GET", "POST", "DELETE"})
	allowedHeaders := handlers.AllowedHeaders([]string{"content-type", "X-Requested-With",
		"Origin", "Accept", "X-PINGOTHER"})

	srv := &http.Server{
		Handler: handlers.CORS(allowMethods, allowedOrigins, allowedHeaders,
			allowCredentials)(r),
		Addr:         "127.0.0.1:1337",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server started up and running at: %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())

}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var creds UserCredentials

	err := json.NewDecoder(r.Body).Decode(&creds)

	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		log.Print(err)
		return
	}

	//We expect only a username and password to be sent to us.

	if creds.Password == "" || creds.Username == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	//we'll need to verify/clear/not screw around with sql injections.

	response, err := CheckPasswordHash(creds.Password, creds.Username)

	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if response {

		token, _ := GenerateKey(creds.Username)
		//TODO: actually stop being lazy and implement a refresh timer
		expTime := time.Now().Add(5 * time.Hour)

		http.SetCookie(w, &http.Cookie{Name: "token",
			Value: token, Path: "/", Expires: expTime})

	} else {
		w.WriteHeader(http.StatusForbidden)
		return
	}

}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var creds RegisteringCredentials

	err := json.NewDecoder(r.Body).Decode(&creds)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}

	if creds.Password == "" || creds.Username == "" || creds.Email == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	_, err = RegisterUser(creds.Username, creds.Password, creds.Email, creds.Referrer)

	token, _ := GenerateKey(creds.Username)
	//TODO: actually stop being lazy and implement a refresh timer
	expTime := time.Now().Add(5 * time.Hour)

	http.SetCookie(w, &http.Cookie{Name: "token",
		Value: token, Path: "/", Expires: expTime})

	if err != nil {
		log.Print(err)

		w.WriteHeader(http.StatusTeapot)

	}
}

func courseHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	_, err = CheckToken(c.Value)
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

func getTermsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	_, err = CheckToken(c.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	enc := json.NewEncoder(w)
	terms, _ := getTerms()
	enc.Encode(terms)

}

func crnCourseHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	_, err = CheckToken(c.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	crn := mux.Vars(r)["crn"]

	if len(crn) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	enc := json.NewEncoder(w)
	desc := GetCourseDescription(crn)

	enc.Encode(desc)

}

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: handle refreshes and stop being a nerd and allowing 5hrs on a single log-in.
	c, err := r.Cookie("token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	_, err = CheckToken(c.Value)
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

	_, err = CheckToken(c.Value)
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

	queryReturns, err := SearchCourses(info)
	if err != nil {
		log.Print(err)
		return
	}

	enc := json.NewEncoder(w)

	enc.Encode(queryReturns)
}

func commitCoursesHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")

	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var username string
	username, err = CheckToken(c.Value)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var courseSelections GenericData

	err = json.NewDecoder(r.Body).Decode(&courseSelections)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	commit, err := CommitSelectedCourses(courseSelections.Data, username)

	if !commit || err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func selectedCoursesHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	log.Print(c)

	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var username string
	username, err = CheckToken(c.Value)

	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var courseSelections []Course

	courseSelections, err = GetSelectedCourses(username)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	enc.Encode(courseSelections)

}

func deleteSelectedCoursesHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")

	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	username, err := CheckToken(c.Value)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var courseSelections GenericData

	err = json.NewDecoder(r.Body).Decode(&courseSelections)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	DeleteSelectedCourses(courseSelections.Data, username)
}

func getCourseOptionsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")

	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	_, err = CheckToken(c.Value)

	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	subj, err := getCourseSubjects()
	if err != nil {
		log.Print(err)
		return
	}

	log.Print(subj)

	enc := json.NewEncoder(w)
	enc.Encode(subj)
}
