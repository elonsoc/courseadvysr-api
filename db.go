package main

//lifted from bondkeepr 2020-08-02

import (
	"database/sql"
	"log"
	"regexp"
	"strings"

	//PostgreSQL driver
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

/*
By the time queries reach the database, we assume that everything is in place
and heaven forbid if it isn't.

Hopefully we've safely assumed that by the time a request reaches these
queries that everything is authenticated.

I hope.
*/

//This is probably illegal relax I'll fix it later man
//DEV: Also, we need to find a better way to understand the difference between dev
//build and prod build for golang
const connStr = "postgres://vysr@localhost:5432/vysr?sslmode=disable"

//GetHash returns the hashed password stored for the specified user
func GetHash(username string) (string, error) {

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	/*
		I'm not sure if I should close after being done with this respective
		query or not. Will have to read up.
	*/
	defer db.Close()

	type Hash struct {
		Hash    string
		IsValid bool
	}

	/*
		weird note.

		I like how we're selecting the password (which would set alarms) of the
		user but really it's just the hash of the user's password. I wonder if
		anyone who read this code would scream if they saw the following query.

		TechLead: "ARE YOU STORING THE PASSWORDS IN PLAINTEXT?"

		Me: "Look at passwords.go"
	*/

	var requestedHash Hash
	err = db.QueryRow("SELECT password,\"isValid\" FROM public.users where username = $1",
		username).Scan(&requestedHash.Hash, &requestedHash.IsValid)

	if err != nil {
		log.Println(err)
		return "", err
	}

	if requestedHash.IsValid != true {
		return "", err
	}

	return requestedHash.Hash, err

}

//GetCourses returns all the courses located in the database
func GetCourses() []Course {

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	//this'll have to be long until I re-write this
	rows, err := db.Query("SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated FROM public.courses")
	//Once we enable the ability to query the database directy without pulling the whole thing down, we'll have to granularlize this.
	if err != nil {
		log.Fatal(err)
	}

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

	return returnCourses
}

//RegisterUser registers (but does not validate them) when given new credentials
func RegisterUser(username string, password string, email string) (bool, error) {
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	_, err = db.Exec(`INSERT INTO "public"."users" ("username","password","email") VALUES ($1, $2, $3)`, username, GeneratePasswordHash(password), email)

	if err != nil {
		log.Print(err)
		return false, err
	}

	return true, err

}

//SearchCourses takes a query and returns a slice of courses that meet the query
func SearchCourses(query SearchQuery) []Course {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	/*
		SELECT * FROM "public"."courses" WHERE ("coursenumber" = '111') AND
		("coursesubject" = 'CHM') ORDER BY "courseregistrationnumber"
		LIMIT 150 OFFSET 0;
	*/
	var someQuery []string
	if strings.Contains(query.Query, ",") {
		someQuery = strings.Split(query.Query, ",")
	} else {
		someQuery = append(someQuery, query.Query)
	}

	var selectedCourses []Course
	//TODO: might want to refactor to clean up the tabulation
	for i := range someQuery {
		var course Course

		//TODO: SQL INJECTION SITE HERE!
		query := strings.TrimSpace(someQuery[i])

		//TODO: not handling errors rn
		courseSubStmt := `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where ("coursesubject" = $1)`
		reCourseSub := regexp.MustCompile(`[A-Z]{3}`)
		reCourseNum := regexp.MustCompile(`[0-9]{3}`)

		//TODO: not handling errors rn but honestly I should but whatever lmao
		courseSubNumStmt := `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where ("coursesubject" = $1) AND ("coursenumber" = $2)`

		reCourseSubNum := regexp.MustCompile(`[A-Z]{3} [0-9]{3}`)

		//TODO: not handling errors rn but honestly I should but whatever lmao
		courseTitleStmt := `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where "coursetitle"::TEXT LIKE $1`

		reCourseTitle := regexp.MustCompile(`^(([^A-Z].{2}|.[^A-Z].|.{2}[^A-Z]).*|.{0,2})$`)

		if query != "" {

			//matches course subject lookup e.g. "CHM"
			if reCourseSub.Match([]byte(query)) {
				log.Print(string(query))
				rows, err := db.Query(courseSubStmt, query)
				if err != nil {
					log.Print(err)
				}

				for rows.Next() {
					rows.Scan(&course.TermCode, &course.SectionStatus, &course.CourseTitle,
						&course.CourseSubject, &course.CourseSection, &course.CourseNumber,
						&course.CourseRegistrationNumber, pq.Array(&course.MeetingDates),
						pq.Array(&course.MeetingDays), pq.Array(&course.MeetingTimes), &course.MeetingBuilding,
						&course.MeetingRoom, &course.Faculty, &course.Credits,
						&course.CurrStudents, &course.MaxStudents, &course.TimeUpdated)

					log.Print(course)

					selectedCourses = append(selectedCourses, course)
				}
			}

			//matches course subject and specific course number e.g. "CHM 111"
			if reCourseSubNum.Match([]byte(query)) {
				courseSub := reCourseSub.Find([]byte(query))
				courseNum := reCourseNum.Find([]byte(query))

				courseNumString := string(courseNum)
				courseSubString := string(courseSub)

				rows, err :=
					db.Query(courseSubNumStmt,
						courseSubString,
						courseNumString)
				if err != nil {
					log.Print(err)
				}

				for rows.Next() {

					rows.Scan(&course.TermCode,
						&course.SectionStatus,
						&course.CourseTitle,
						&course.CourseSubject,
						&course.CourseSection,
						&course.CourseNumber,
						&course.CourseRegistrationNumber,
						pq.Array(&course.MeetingDates),
						pq.Array(&course.MeetingDays),
						pq.Array(&course.MeetingTimes),
						&course.MeetingBuilding,
						&course.MeetingRoom,
						&course.Faculty,
						&course.Credits,
						&course.CurrStudents,
						&course.MaxStudents,
						&course.TimeUpdated)

					selectedCourses = append(selectedCourses, course)
				}
			}

			//matches course title w/o sub or number e.g. "Chemistry III"
			if reCourseTitle.Match([]byte(query)) {

				rows, err := db.Query(courseTitleStmt, "%"+query+"%")
				if err != nil {
					log.Print(err)
				}

				for rows.Next() {
					rows.Scan(&course.TermCode, &course.SectionStatus, &course.CourseTitle,
						&course.CourseSubject, &course.CourseSection, &course.CourseNumber,
						&course.CourseRegistrationNumber, pq.Array(&course.MeetingDates),
						pq.Array(&course.MeetingDays), pq.Array(&course.MeetingTimes), &course.MeetingBuilding,
						&course.MeetingRoom, &course.Faculty, &course.Credits,
						&course.CurrStudents, &course.MaxStudents, &course.TimeUpdated)

					selectedCourses = append(selectedCourses, course)
				}
			}
		}

	}

	return selectedCourses

}

//CommitSelectedCourses takes a slice of CRNs and commits them to the user_courses db
func CommitSelectedCourses(courses []string, username string) (bool, error) {

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	//TODO: We don't check for already present course
	for _, crn := range courses {
		res, err := db.Exec(`INSERT INTO "public"."user_courses" ("username", "course") VALUES ($1,$2)`, username, crn)
		log.Print(res)
		if err != nil {
			log.Print(err)

		}
	}

	return true, nil
}

//GetSelectedCourses returns the selected courses for a given user
func GetSelectedCourses(username string) ([]Course, error) {
	var returnCourses []Course
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
		return returnCourses, err
	}

	defer db.Close()

	rows, err := db.Query(`
		SELECT
			termcode,
			sectionstatus,
			coursetitle,
			coursesubject,
			coursesection,
			coursenumber,
			courseregistrationnumber,
			meetingdates,
			meetingdays,
			meetingtimes,
			meetingbuilding,
			meetingroom,
			faculty,
			credits,
			currstudents,
			maxstudents,
			timeupdated
		FROM
			user_courses
		INNER JOIN 
			courses 
		ON 
			courses.courseregistrationnumber = user_courses.course
		WHERE
			user_courses.username = $1`,
		username)

	if err != nil {
		log.Print(err)
		return returnCourses, err
	}

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

	return returnCourses, err
}

func DeleteSelectedCourses(courses []string, username string) (bool, error) {

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
		return false, err
	}

	defer db.Close()
	for _, crn := range courses {
		_, err = db.Exec(`
			DELETE FROM 
				"public"."user_courses" 
			WHERE 
				("username" = $1 AND "course" = $2)`,
			username, crn)

		if err != nil {
			log.Print(err)
		}
	}

	return true, err

}
