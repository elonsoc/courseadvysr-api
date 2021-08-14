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

const (
	getCoursesStmt string = `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where ("termcode" = $1)`

	//TODO: not handling errors rn
	courseSubStmt string = `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where ("coursesubject" = $1) AND ("termcode" = $2)`
	courseSubPat  string = `\A[A-Z]{3,4}`
	courseNumPat  string = `[0-9]{3,4}`

	//TODO: not handling errors rn but honestly I should but whatever lmao

	courseSubNumStmt string = `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where ("coursesubject" = $1) AND ("coursenumber"::TEXT LIKE $2%) AND ("termcode" = $3)`
	courseSubNumPat  string = `(?i)\A[A-Z]{3,4} [0-9]{3,4}`

	subMajNumStmt string = `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where ("coursesubject" = $1) AND ("coursenumber"::TEXT LIKE $2) AND ("termcode" = $3)`
	subMajNumPat  string = `(?i)([A-Z]{3,4}):{1}([0-4]){1}`

	//TODO: not handling errors rn but honestly I should but whatever lmao
	courseTitleStmt string = `SELECT termcode, sectionstatus, coursetitle, coursesubject, coursesection, coursenumber, courseregistrationnumber, meetingdates, meetingdays, meetingtimes, meetingbuilding, meetingroom, faculty, credits, currstudents, maxstudents, timeupdated from public.courses where "coursetitle"::TEXT ILIKE $1 AND ("termcode" = $2)`
	courseTitlePat  string = `^(([^A-Z].{2}|.[^A-Z].|.{2}[^A-Z]).*|.{0,2})$`
)

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

	if !requestedHash.IsValid {
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
	err := conn.QueryRow(context.Background(),
		`select case when EXISTS (select email from undergraduates where email = $1 ) then true else false end from public.undergraduates limit 1`,
		referrer).Scan(&referrerDoesExist)

	if err != nil {
		log.Fatal(err)
		return false, err
	}

	if !referrerDoesExist {
		return false, errors.New("the given referrer does not exist")
	}

	_, err = conn.Exec(context.Background(),
		`INSERT INTO "public"."users" ("username","password","email", "isValid") VALUES ($1, $2, $3, $4)`,
		username,
		GeneratePasswordHash(password),
		email,
		true)

	if err != nil {
		// log.Print(err)
		return false, err
	}

	conn.Exec(context.Background(),
		`insert into public.friends ("friend","is_friend_of") values ((select id from public.users where username = $1), (select id from public.users where email= $2))`,
		username,
		referrer)

	return true, err

}

//SearchCourses takes a query and returns a slice of courses that meet the query
func SearchCourses(query SearchQuery) ([]Course, error) {
	conn := openConnection()

	reCourseSub := regexp.MustCompile(courseSubPat)
	reCourseNum := regexp.MustCompile(courseNumPat)
	reCourseSubNum := regexp.MustCompile(courseSubNumPat)
	reCourseSubMajNum := regexp.MustCompile(subMajNumPat)
	reCourseTitle := regexp.MustCompile(courseTitlePat)
	defer conn.Close(context.Background())

	/*
		SELECT * FROM "public"."courses" WHERE ("coursenumber" = '111') AND
		("coursesubject" = 'CHM') ORDER BY "courseregistrationnumber"
		LIMIT 150 OFFSET 0;
	*/

	var selectedCourses []Course
	//TODO: might want to refactor to clean up the tabulation
	for i := range query.Query {
		//TODO: SQL INJECTION SITE HERE!
		aQuery := strings.TrimSpace(query.Query[i])

		if aQuery != "" {

			//matches course subject lookup e.g. "CHM"
			//TODO: I can do something here way better
			if reCourseSub.Match([]byte(aQuery)) && !reCourseNum.Match([]byte(aQuery)) && len(aQuery) == 3 || len(aQuery) == 4 {
				rows, err := conn.Query(context.Background(), courseSubStmt, strings.ToUpper(aQuery), query.Term)
				if err != nil {
					log.Print(err)
				}

				selectedCourses = append(selectedCourses, returnCourses(rows)...)

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
					return nil, err
				}

				selectedCourses = append(selectedCourses, returnCourses(rows)...)
			} else if reCourseSubMajNum.Match([]byte(aQuery)) {
				log.Println(query)
				rows, err := conn.Query(context.Background(), subMajNumStmt, strings.ToUpper(aQuery[0:3]), aQuery[4:5]+"%", query.Term)
				if err != nil {
					log.Print(err)
					log.Print(strings.ToUpper(aQuery[0:3]))
					log.Print(strings.ToUpper(aQuery[4:5]))
				}

				selectedCourses = append(selectedCourses, returnCourses(rows)...)

			} else if reCourseTitle.Match([]byte(aQuery)) {
				//matches course title w/o sub or number e.g. "Chemistry III"
				log.Println(query)
				rows, err := conn.Query(context.Background(), courseTitleStmt, "%"+aQuery+"%", query.Term)
				if err != nil {
					log.Print(err)
					return nil, err
				}

				selectedCourses = append(selectedCourses, returnCourses(rows)...)

			}
		} else {
			rows, err := conn.Query(context.Background(), getCoursesStmt, query.Term)
			if err != nil {
				log.Print(err)
				return nil, err
			}

			selectedCourses = append(selectedCourses, returnCourses(rows)...)

		}

	}

	return selectedCourses, nil

}

//CommitSelectedCourses takes a slice of CRNs and commits them to the user_courses table
func CommitSelectedCourses(courses []string, username string) (bool, error) {

	conn := openConnection()
	defer conn.Close(context.Background())

	//TODO: We don't check for already present course
	for _, crn := range courses {
		res, err := conn.Exec(context.Background(),
			`INSERT INTO "public"."user_courses" ("username", "course") VALUES ($1,$2)`,
			username, crn)
		log.Print(res)
		if err != nil {
			log.Print(err)

		}
	}

	return true, nil
}

//GetSelectedCourses returns the selected courses for a given user
func GetSelectedCourses(username string) ([]Course, error) {

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
		return nil, err
	}

	return returnCourses(rows), err
}

func GetCourseDescription(crn string) string {
	var description Description
	var courseSubject, courseNumber string

	conn := openConnection()
	defer conn.Close(context.Background())

	//This isn't exactly performant, but that's ok.
	row := conn.QueryRow(context.Background(),
		`SELECT courseSubject, courseNumber from courses where courseRegistrationNumber = $1`,
		crn)
	row.Scan(&courseSubject, &courseNumber)
	row = conn.QueryRow(context.Background(),
		`SELECT description from coursedescriptions where courseSubject = $1 AND courseNumber = $2`,
		courseSubject, courseNumber)

	row.Scan(&description.Description)

	print(description.Description)

	return description.Description

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

func getCourseSubjects() ([]CourseSubjects, error) {

	var csubj []CourseSubjects
	conn := openConnection()
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), `
		SELECT subject, title from subjects`)
	if err != nil {
		log.Print(err)
		return csubj, err
	}

	for rows.Next() {
		var subject CourseSubjects
		rows.Scan(&subject.Subject, &subject.Title)
		log.Print(subject)
		csubj = append(csubj, subject)
	}

	return csubj, nil
}

func returnCourses(rows pgx.Rows) []Course {
	var courseList []Course
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

		courseList = append(courseList, course)
	}

	return courseList
}
