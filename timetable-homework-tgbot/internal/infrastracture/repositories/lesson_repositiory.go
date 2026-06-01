package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"timetable-homework-tgbot/internal/domain"
	"timetable-homework-tgbot/internal/infrastracture/database"
)

type LessonsRepository interface {
	GetLessonsGroup(ctx context.Context, group string) ([]domain.LessonBrief, error)
	GetLessonsTeacher(ctx context.Context, teacherFio string) ([]domain.LessonBrief, error)
	GetLessonsRoom(ctx context.Context, roomName string) ([]domain.LessonBrief, error)
	GetDaysWithLessonsByGroup(ctx context.Context, group string) ([]string, error)
	LessonsByDayGroupForTeacher(ctx context.Context, userID int64, group, day string) ([]domain.LessonBrief, error)
	HasTeacherLesson(ctx context.Context, userID int64, group, day, subject string) (bool, error)
}

type LessonsRepo struct {
	db *database.DB
}

func NewLessonsRepo(db *database.DB) *LessonsRepo {
	return &LessonsRepo{db: db}
}

func (r *LessonsRepo) GetLessonsGroup(ctx context.Context, group string) ([]domain.LessonBrief, error) {
	const q = `
SELECT
    l.subject,
    l.lesson_type,
    t.teacher_name,
    l.start_time::text,
    l.weekday,
    a.room_name,
    l.week,
    string_agg(g2.group_name, ', ' ORDER BY g2.group_name) AS groups
FROM lessons l
JOIN lesson_groups lg ON lg.lesson_id = l.id
JOIN groups g ON g.id = lg.group_id
JOIN teachers t ON t.id = l.teacher_id
JOIN auditoriums a ON a.id = l.room_id
JOIN lesson_groups lg2 ON lg2.lesson_id = l.id
JOIN groups g2 ON g2.id = lg2.group_id
WHERE g.group_name = $1
GROUP BY l.id, t.teacher_name, a.room_name
ORDER BY l.weekday, l.start_time;
`
	log.Println("Querying group lessons:", group)
	return r.queryLessons(ctx, q, strings.ReplaceAll(strings.TrimSpace(group), " ", ""))
}

func (r *LessonsRepo) GetLessonsTeacher(ctx context.Context, teacherFio string) ([]domain.LessonBrief, error) {
	const q = `
SELECT
    l.subject,
    l.lesson_type,
    t.teacher_name,
    l.start_time::text,
    l.weekday,
    a.room_name,
    l.week,
    string_agg(g.group_name, ', ' ORDER BY g.group_name) AS groups
FROM lessons l
JOIN teachers t ON t.id = l.teacher_id
JOIN auditoriums a ON a.id = l.room_id
JOIN lesson_groups lg ON lg.lesson_id = l.id
JOIN groups g ON g.id = lg.group_id
WHERE lower(t.teacher_name) = lower($1)
GROUP BY l.id, t.teacher_name, a.room_name
ORDER BY l.weekday, l.start_time;
`
	log.Println("Querying teacher lessons:", teacherFio)
	return r.queryLessons(ctx, q, strings.TrimSpace(teacherFio))
}

func (r *LessonsRepo) GetLessonsRoom(ctx context.Context, roomName string) ([]domain.LessonBrief, error) {
	const q = `
SELECT
    l.subject,
    l.lesson_type,
    t.teacher_name,
    l.start_time::text,
    l.weekday,
    a.room_name,
    l.week,
    string_agg(g.group_name, ', ' ORDER BY g.group_name) AS groups
FROM lessons l
JOIN teachers t ON t.id = l.teacher_id
JOIN auditoriums a ON a.id = l.room_id
JOIN lesson_groups lg ON lg.lesson_id = l.id
JOIN groups g ON g.id = lg.group_id
WHERE a.room_name = $1
GROUP BY l.id, t.teacher_name, a.room_name
ORDER BY l.weekday, l.start_time;
`
	log.Println("Querying room lessons:", roomName)
	return r.queryLessons(ctx, q, strings.TrimSpace(roomName))
}

func (r *LessonsRepo) GetDaysWithLessonsByGroup(ctx context.Context, group string) ([]string, error) {
	const q = `
SELECT l.weekday
FROM lessons l
JOIN lesson_groups lg ON lg.lesson_id = l.id
JOIN groups g ON g.id = lg.group_id
WHERE g.group_name = $1
GROUP BY l.weekday
ORDER BY l.weekday;
`
	rows, err := r.db.GetSql().QueryContext(ctx, q, group)
	if err != nil {
		return nil, fmt.Errorf("GetDaysWithLessonsByGroup query: %w", err)
	}
	defer rows.Close()

	var res []string
	for rows.Next() {
		var weekday int
		if err := rows.Scan(&weekday); err != nil {
			return nil, fmt.Errorf("GetDaysWithLessonsByGroup scan: %w", err)
		}
		res = append(res, weekdayName(weekday))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetDaysWithLessonsByGroup rows: %w", err)
	}

	return res, nil
}

func (r *LessonsRepo) LessonsByDayGroupForTeacher(ctx context.Context, userID int64, group, day string) ([]domain.LessonBrief, error) {
	const q = `
SELECT
    l.subject,
    l.lesson_type,
    t.teacher_name,
    l.start_time::text,
    l.weekday,
    a.room_name,
    l.week,
    string_agg(g2.group_name, ', ' ORDER BY g2.group_name) AS groups
FROM lessons l
JOIN lesson_groups lg ON lg.lesson_id = l.id
JOIN groups g ON g.id = lg.group_id
JOIN teachers t ON t.id = l.teacher_id
JOIN auditoriums a ON a.id = l.room_id
JOIN lesson_groups lg2 ON lg2.lesson_id = l.id
JOIN groups g2 ON g2.id = lg2.group_id
WHERE g.group_name = $1
  AND l.weekday = $2
  AND t.user_id = $3
GROUP BY l.id, t.teacher_name, a.room_name
ORDER BY l.start_time;
`
	return r.queryLessons(ctx, q, group, weekdayNumber(day), userID)
}

func (r *LessonsRepo) HasTeacherLesson(ctx context.Context, userID int64, group, day, subject string) (bool, error) {
	const q = `
SELECT 1
FROM lessons l
JOIN lesson_groups lg ON lg.lesson_id = l.id
JOIN groups g ON g.id = lg.group_id
JOIN teachers t ON t.id = l.teacher_id
WHERE t.user_id = $1
  AND g.group_name = $2
  AND l.weekday = $3
  AND l.subject = $4
LIMIT 1;
`
	var exists int
	err := r.db.GetSql().QueryRowContext(ctx, q, userID, group, weekdayNumber(day), subject).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("HasTeacherLesson query: %w", err)
	}
	return true, nil
}

func (r *LessonsRepo) queryLessons(ctx context.Context, query string, args ...any) ([]domain.LessonBrief, error) {
	rows, err := r.db.GetSql().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query lessons: %w", err)
	}
	defer rows.Close()

	var res []domain.LessonBrief
	for rows.Next() {
		var (
			subject    string
			lessonType sql.NullString
			tutor      sql.NullString
			startTime  string
			weekday    int
			room       sql.NullString
			week       sql.NullInt64
			groups     sql.NullString
		)
		if err := rows.Scan(&subject, &lessonType, &tutor, &startTime, &weekday, &room, &week, &groups); err != nil {
			return nil, fmt.Errorf("scan lessons: %w", err)
		}

		res = append(res, domain.LessonBrief{
			Title:      subject,
			LessonType: nullToString(lessonType),
			Tutor:      nullToString(tutor),
			StartTime:  trimTime(startTime),
			Weekday:    weekdayName(weekday),
			Room:       nullToString(room),
			Groups:     nullToString(groups),
			Week:       nullIntToString(week),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lessons rows: %w", err)
	}

	return res, nil
}

func nullToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullIntToString(ni sql.NullInt64) string {
	if ni.Valid {
		return fmt.Sprintf("%d", ni.Int64)
	}
	return ""
}

func weekdayNumber(day string) int {
	switch strings.TrimSpace(day) {
	case "Понедельник":
		return 1
	case "Вторник":
		return 2
	case "Среда":
		return 3
	case "Четверг":
		return 4
	case "Пятница":
		return 5
	case "Суббота":
		return 6
	case "Воскресенье":
		return 7
	default:
		return 0
	}
}

func weekdayName(day int) string {
	switch day {
	case 1:
		return "Понедельник"
	case 2:
		return "Вторник"
	case 3:
		return "Среда"
	case 4:
		return "Четверг"
	case 5:
		return "Пятница"
	case 6:
		return "Суббота"
	case 7:
		return "Воскресенье"
	default:
		return ""
	}
}

func trimTime(s string) string {
	return strings.TrimSuffix(s, ":00")
}
