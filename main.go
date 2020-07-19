package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"database/sql"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/courses/", courseHandler).Methods("GET")

	srv := &http.Server{
		Handler:      handlers.CORS()(r),
		Addr:         "127.0.0.1:1337",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server started up and running at: %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())

}

//Course describes the relevant course in question
type Course struct {
	TermCode                 string   `json:"termCode"`
	SectionStatus            string   `json:"sectionStatus"`
	CourseTitle              string   `json:"courseTitle"`
	CourseSection            string   `json:"courseSection"`
	CourseSubject            string   `json:"courseSubject"`
	CourseNumber             string   `json:"courseNumber"`
	CourseRegistrationNumber string   `json:"courseRegistrationNumber"`
	MeetingDates             []string `json:"meetingDates"`
	MeetingDays              []string `json:"meetingDays"`
	MeetingTimes             []string `json:"meetingTimes"`
	MeetingBuilding          string   `json:"meetingBuilding"`
	MeetingRoom              string   `json:"meetingRoom"`
	Faculty                  string   `json:"faculty"`
	Credits                  float32  `json:"credits"`
	CurrStudents             int      `json:"currStudents"`
	MaxStudents              int      `json:"maxStudents"`
	TimeUpdated              float32  `json:"timeUpdated"`
}

func courseHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Accessed Courses")
	const connStr = "sslmode=disable user=jumarmartin dbname=vysr"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	//this'll have to be long until I re-write this
	rows, err := db.Query("SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated FROM public.courses")

	enc := json.NewEncoder(w)

	var returnCourses []Course

	var course Course
	for rows.Next() {

		//I hope to be able to map this away but it requires some cute stuff
		//because we use pq.Array unless we map pq.Array and access it weirdly
		//that isn't cool
		rows.Scan(&course.TermCode, &course.SectionStatus, &course.CourseTitle,
			&course.CourseSubject, &course.CourseSection, &course.CourseNumber,
			&course.CourseRegistrationNumber, pq.Array(&course.MeetingDates),
			pq.Array(&course.MeetingDays), pq.Array(&course.MeetingTimes), &course.MeetingBuilding,
			&course.MeetingRoom, &course.Faculty, &course.Credits,
			&course.CurrStudents, &course.MaxStudents, &course.TimeUpdated)

		returnCourses = append(returnCourses, course)
	}
	enc.Encode(returnCourses)
}
