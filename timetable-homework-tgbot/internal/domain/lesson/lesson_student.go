package lesson

type LessonStudent struct {
	Subject    string
	LessonType string
	Tutor      string
	StartTime  string
	Weekday    string
	Room       string
	Week       string
}

func NewLessonStudent(subject string, lessonType string, tutor string, startTime string, weekday string, room string, week string) *LessonStudent {
	return &LessonStudent{Subject: subject, LessonType: lessonType, Tutor: tutor, StartTime: startTime, Weekday: weekday, Room: room, Week: week}
}
