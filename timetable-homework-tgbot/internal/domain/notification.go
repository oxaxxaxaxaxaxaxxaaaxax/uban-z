package domain

import "time"

type Notification struct {
	UserID     int64
	HomeworkID int64
	Subject    string
	Timestamp  time.Time
}
