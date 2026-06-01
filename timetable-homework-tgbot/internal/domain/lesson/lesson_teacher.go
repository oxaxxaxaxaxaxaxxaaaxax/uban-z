package lesson

type LessonTeacher struct {
	Subject    string
	LessonType string
	Groups     []string
	StartTime  string
	Weekday    string
	Room       string
	Week       string
}

func NewLessonTeacher(subject string, lessonType string, groups []string, startTime string, weekday string, room string, week string) *LessonTeacher {
	return &LessonTeacher{Subject: subject, LessonType: lessonType, Groups: groups, StartTime: startTime, Weekday: weekday, Room: room, Week: week}
}
