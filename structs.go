package main

//Course describes the relevant course in question, usually identified by the CRN
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

type UserCredentials struct {
	Username, Password string
}

type RegisteringCredentials struct {
	Username, Password, Email string
}

type SearchQuery struct {
	Query string
}

type GenericData struct {
	Data []string
}
