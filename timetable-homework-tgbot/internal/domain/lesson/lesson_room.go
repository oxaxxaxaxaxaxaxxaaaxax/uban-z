package lesson

type LessonRoom struct {
	Subject    string
	LessonType string
	Tutor      string
	StartTime  string
	Weekday    string
	Groups     []string
	Week       string
}

func NewLessonRoom(subject string, lessonType string, tutor string, startTime string, weekday string, groups []string, week string) *LessonRoom {
	return &LessonRoom{Subject: subject, LessonType: lessonType, Tutor: tutor, StartTime: startTime, Weekday: weekday, Groups: groups, Week: week}
}
