package main

//lifted from bondkeepr 2020-08-02

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v4"
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
func openConnection() *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), "database=vysr")
	if err != nil {
		log.Panic(err)
	}
	return conn
}

//GetHash returns the hashed password stored for the specified user
func GetHash(username string) (string, error) {

	conn := openConnection()
	defer conn.Close(context.Background())
	/*
		I'm not sure if I should close after being done with this respective
		query or not. Will have to read up.
	*/

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
	err := conn.QueryRow(context.Background(), "SELECT password,\"isValid\" FROM public.users where username = $1",
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

	conn := openConnection()
	defer conn.Close(context.Background())

	//this'll have to be long until I re-write this
	rows, err := conn.Query(context.Background(), "SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated FROM public.courses")
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
			&course.CourseRegistrationNumber, &course.MeetingDates,
			&course.MeetingDays, &course.MeetingTimes, &course.MeetingBuilding,
			&course.MeetingRoom, &course.Faculty, &course.Credits,
			&course.CurrStudents, &course.MaxStudents, &course.TimeUpdated)

		returnCourses = append(returnCourses, course)
	}

	return returnCourses
}

//RegisterUser registers (but does not validate them) when given new credentials
func RegisterUser(username string, password string, email string, referrer string) (bool, error) {
	conn := openConnection()
	defer conn.Close(context.Background())

	var referrerDoesExist bool
	err := conn.QueryRow(context.Background(), `select case when EXISTS (select email from undergraduates where email = $1 ) then true else false end from public.undergraduates limit 1`, referrer).Scan(&referrerDoesExist)

	if err != nil {
		log.Fatal(err)
		return false, err
	}

	if !referrerDoesExist {
		return false, errors.New("the given referrer does not exist")
	}

	_, err = conn.Exec(context.Background(), `INSERT INTO "public"."users" ("username","password","email", "isValid") VALUES ($1, $2, $3, $4)`, username, GeneratePasswordHash(password), email, true)

	if err != nil {
		// log.Print(err)
		return false, err
	}

	conn.Exec(context.Background(), `insert into public.friends ("friend","is_friend_of") values ((select id from public.users where username = $1), (select id from public.users where email= $2))`, username, referrer)

	return true, err

}

//SearchCourses takes a query and returns a slice of courses that meet the query
func SearchCourses(query SearchQuery) []Course {
	conn := openConnection()
	defer conn.Close(context.Background())
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
		aQuery := strings.TrimSpace(someQuery[i])

		getCoursesStmt := `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where ("termcode" = $1)`

		//TODO: not handling errors rn
		courseSubStmt := `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where ("coursesubject" = $1) AND ("termcode" = $2)`
		reCourseSub := regexp.MustCompile(`(?i)\A[A-Z]{3}`)
		reCourseNum := regexp.MustCompile(`[0-9]{3}`)

		//TODO: not handling errors rn but honestly I should but whatever lmao
		courseSubNumStmt := `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where ("coursesubject" = $1) AND ("coursenumber" = $2) AND ("termcode" = $3)`

		reCourseSubNum := regexp.MustCompile(`(?i)\A[A-Z]{3} [0-9]{3}`)

		//TODO: not handling errors rn but honestly I should but whatever lmao
		courseTitleStmt := `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where "coursetitle"::TEXT ILIKE $1 AND ("termcode" = $2)`

		reCourseTitle := regexp.MustCompile(`^(([^A-Z].{2}|.[^A-Z].|.{2}[^A-Z]).*|.{0,2})$`)

		if aQuery != "" {

			//matches course subject lookup e.g. "CHM"
			if reCourseSub.Match([]byte(aQuery)) && reCourseNum.Match([]byte(aQuery)) == false && len(aQuery) == 3 {
				rows, err := conn.Query(context.Background(), courseSubStmt, strings.ToUpper(aQuery), query.Term)
				if err != nil {
					log.Print(err)
				}

				for rows.Next() {
					rows.Scan(&course.TermCode, &course.SectionStatus, &course.CourseTitle,
						&course.CourseSubject, &course.CourseSection, &course.CourseNumber,
						&course.CourseRegistrationNumber, &course.MeetingDates,
						&course.MeetingDays, &course.MeetingTimes, &course.MeetingBuilding,
						&course.MeetingRoom, &course.Faculty, &course.Credits,
						&course.CurrStudents, &course.MaxStudents, &course.TimeUpdated)

					log.Print(course)

					selectedCourses = append(selectedCourses, course)
				}
			} else if reCourseSubNum.Match([]byte(aQuery)) {
				courseSub := reCourseSub.Find([]byte(strings.ToUpper(aQuery)))
				courseNum := reCourseNum.Find([]byte(aQuery))

				courseNumString := string(courseNum)
				courseSubString := string(courseSub)

				rows, err :=
					conn.Query(context.Background(), courseSubNumStmt,
						courseSubString,
						courseNumString,
						query.Term)
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
						&course.MeetingDates,
						&course.MeetingDays,
						&course.MeetingTimes,
						&course.MeetingBuilding,
						&course.MeetingRoom,
						&course.Faculty,
						&course.Credits,
						&course.CurrStudents,
						&course.MaxStudents,
						&course.TimeUpdated)

					selectedCourses = append(selectedCourses, course)
				}
			} else if reCourseTitle.Match([]byte(aQuery)) {
				//matches course title w/o sub or number e.g. "Chemistry III"
				log.Println(query)
				rows, err := conn.Query(context.Background(), courseTitleStmt, "%"+aQuery+"%", query.Term)
				if err != nil {
					log.Print(err)
				}

				for rows.Next() {
					rows.Scan(&course.TermCode, &course.SectionStatus, &course.CourseTitle,
						&course.CourseSubject, &course.CourseSection, &course.CourseNumber,
						&course.CourseRegistrationNumber, &course.MeetingDates,
						&course.MeetingDays, &course.MeetingTimes, &course.MeetingBuilding,
						&course.MeetingRoom, &course.Faculty, &course.Credits,
						&course.CurrStudents, &course.MaxStudents, &course.TimeUpdated)

					selectedCourses = append(selectedCourses, course)
				}
			}
		} else {
			var course Course
			rows, err := conn.Query(context.Background(), getCoursesStmt, query.Term)
			if err != nil {
				log.Print(err)
			}
			for rows.Next() {
				rows.Scan(&course.TermCode, &course.SectionStatus, &course.CourseTitle,
					&course.CourseSubject, &course.CourseSection, &course.CourseNumber,
					&course.CourseRegistrationNumber, &course.MeetingDates,
					&course.MeetingDays, &course.MeetingTimes, &course.MeetingBuilding,
					&course.MeetingRoom, &course.Faculty, &course.Credits,
					&course.CurrStudents, &course.MaxStudents, &course.TimeUpdated)

				selectedCourses = append(selectedCourses, course)
			}

		}

	}

	return selectedCourses

}

//CommitSelectedCourses takes a slice of CRNs and commits them to the user_courses db
func CommitSelectedCourses(courses []string, username string) (bool, error) {

	conn := openConnection()
	defer conn.Close(context.Background())

	//TODO: We don't check for already present course
	for _, crn := range courses {
		res, err := conn.Exec(context.Background(), `INSERT INTO "public"."user_courses" ("username", "course") VALUES ($1,$2)`, username, crn)
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
	conn := openConnection()
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), `
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
			&course.CourseRegistrationNumber, &course.MeetingDates,
			&course.MeetingDays, &course.MeetingTimes, &course.MeetingBuilding,
			&course.MeetingRoom, &course.Faculty, &course.Credits,
			&course.CurrStudents, &course.MaxStudents, &course.TimeUpdated)

		returnCourses = append(returnCourses, course)
	}

	return returnCourses, err
}

func DeleteSelectedCourses(courses []string, username string) (bool, error) {

	conn := openConnection()
	defer conn.Close(context.Background())

	for _, crn := range courses {
		_, err := conn.Exec(context.Background(), `
			DELETE FROM
				"public"."user_courses"
			WHERE
				("username" = $1 AND "course" = $2)`,
			username, crn)

		if err != nil {
			log.Print(err)
		}
	}

	return true, errors.New("")

}
