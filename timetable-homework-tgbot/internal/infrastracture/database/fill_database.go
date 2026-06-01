package database

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"timetable-homework-tgbot/internal/domain/lesson"
	"timetable-homework-tgbot/internal/domain/urlselector"
)

const workers = 20

var timetableSaveMu sync.Mutex

func (d *DB) FillDatabase() error {
	ctx := context.Background()

	log.Println("Filling timetable ...")
	if err := d.fillGroupsParallel(ctx); err != nil {
		return err
	}

	log.Println("database fill done")
	return nil
}

func (d *DB) fillGroupsParallel(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Println("Parsing faculties ...")
	faculties := d.parser.ParseFaculties()
	log.Printf("Parsed faculties: %d", len(faculties))

	var allGroups []urlselector.Group
	for i, faculty := range faculties {
		log.Printf("Parsing groups for faculty %d/%d: %s", i+1, len(faculties), faculty.FacultyName)
		groups := d.parser.ParseGroups(faculty.FullUrl)
		allGroups = append(allGroups, groups...)
		log.Printf("Faculty %d/%d done: %s, groups=%d, total_groups=%d", i+1, len(faculties), faculty.FacultyName, len(groups), len(allGroups))
	}

	log.Printf("Starting timetable parsing for groups: total=%d workers=%d", len(allGroups), workers)
	jobs := make(chan urlselector.Group, workers)
	errCh := make(chan error, 1)
	var started atomic.Int64
	var processed atomic.Int64

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for g := range jobs {
				current := started.Add(1)
				if current <= int64(workers) || current%25 == 0 {
					log.Printf("Parsing group timetable: %d/%d, group=%s", current, len(allGroups), g.GroupName)
				}
				lessons := d.parser.ParseLessonsStudent(g.GroupUrl)
				log.Printf("Group timetable parsed: group=%s, lessons=%d; saving to database", g.GroupName, len(lessons))
				if err := d.fillGroupSchedule(ctx, g.GroupName, lessons); err != nil {
					select {
					case errCh <- fmt.Errorf("fill group schedule %s: %w", g.GroupName, err):
					default:
					}
					cancel()
					return
				}
				done := processed.Add(1)
				if done == 1 || done%25 == 0 || int(done) == len(allGroups) {
					log.Printf("Parsed group timetables: %d/%d, last_group=%s, lessons=%d", done, len(allGroups), g.GroupName, len(lessons))
				}
			}
		}()
	}

	for _, g := range allGroups {
		select {
		case jobs <- g:
		case err := <-errCh:
			cancel()
			close(jobs)
			wg.Wait()
			return err
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			select {
			case err := <-errCh:
				return err
			default:
				return ctx.Err()
			}
		}
	}
	close(jobs)

	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		log.Printf("Group timetable parsing done: %d/%d", processed.Load(), len(allGroups))
		return nil
	}
}

func (d *DB) fillGroupSchedule(ctx context.Context, groupName string, lessons []lesson.LessonStudent) error {
	timetableSaveMu.Lock()
	defer timetableSaveMu.Unlock()

	groupName = normalizeGroup(groupName)
	for _, it := range lessons {
		if err := d.upsertGroupLesson(ctx, groupName, it.Subject, it.LessonType, it.Tutor, it.StartTime, it.Weekday, it.Room, it.Week); err != nil {
			return fmt.Errorf("insert group lesson: %w", err)
		}
	}
	return nil
}

func (d *DB) upsertGroupLesson(ctx context.Context, groupName, subject, lessonType, tutor, startTime, weekday, room, week string) error {
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	groupID, err := ensureGroup(ctx, tx, groupName)
	if err != nil {
		return err
	}
	teacherID, err := ensureTeacher(ctx, tx, strings.TrimSpace(tutor))
	if err != nil {
		return err
	}
	roomID, err := ensureRoom(ctx, tx, strings.TrimSpace(room))
	if err != nil {
		return err
	}
	lessonID, err := ensureLesson(ctx, tx, subject, lessonType, weekdayNumber(weekday), startTime, weekNumber(week), teacherID, roomID)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO lesson_groups (lesson_id, group_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, lessonID, groupID); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureGroup(ctx context.Context, tx *sql.Tx, groupName string) (int64, error) {
	var id int64
	groupName = normalizeGroup(groupName)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO groups (group_name)
VALUES ($1)
ON CONFLICT (group_name) DO NOTHING
`, groupName); err != nil {
		return 0, err
	}
	err := tx.QueryRowContext(ctx, `SELECT id FROM groups WHERE group_name = $1`, groupName).Scan(&id)
	return id, err
}

func ensureTeacher(ctx context.Context, tx *sql.Tx, teacherName string) (int64, error) {
	if teacherName == "" {
		teacherName = "Не указан"
	}
	userID := pseudoTelegramID("teacher:" + teacherName)
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (tg_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return 0, err
	}

	var id int64
	if _, err := tx.ExecContext(ctx, `
INSERT INTO teachers (user_id, teacher_name)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, userID, teacherName); err != nil {
		return 0, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM teachers WHERE teacher_name = $1`, teacherName).Scan(&id); err != nil {
		return 0, err
	}
	_, err := tx.ExecContext(ctx, `
UPDATE teachers
SET login = COALESCE(login, 'teacher_' || id::TEXT),
    password = COALESCE(password, 'teacher_' || id::TEXT)
WHERE id = $1
`, id)
	return id, err
}

func ensureRoom(ctx context.Context, tx *sql.Tx, roomName string) (int64, error) {
	if roomName == "" {
		roomName = "Не указана"
	}

	var id int64
	if _, err := tx.ExecContext(ctx, `
INSERT INTO auditoriums (room_name)
VALUES ($1)
ON CONFLICT (room_name) DO NOTHING
`, roomName); err != nil {
		return 0, err
	}
	err := tx.QueryRowContext(ctx, `SELECT id FROM auditoriums WHERE room_name = $1`, roomName).Scan(&id)
	return id, err
}

func ensureLesson(ctx context.Context, tx *sql.Tx, subject, lessonType string, weekday int, startTime string, week sql.NullInt64, teacherID, roomID int64) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
SELECT id
FROM lessons
WHERE subject = $1
  AND COALESCE(lesson_type, '') = COALESCE($2, '')
  AND weekday = $3
  AND start_time = $4
  AND COALESCE(week, 0) = COALESCE($5, 0)
  AND teacher_id = $6
  AND room_id = $7
LIMIT 1
`, subject, nullString(lessonType), weekday, startTime, week, teacherID, roomID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	err = tx.QueryRowContext(ctx, `
INSERT INTO lessons (subject, lesson_type, weekday, start_time, week, teacher_id, room_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id
`, subject, nullString(lessonType), weekday, startTime, week, teacherID, roomID).Scan(&id)
	return id, err
}

func normalizeGroup(group string) string {
	return strings.ReplaceAll(strings.TrimSpace(group), " ", "")
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
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
		return 1
	}
}

func weekNumber(week string) sql.NullInt64 {
	if strings.TrimSpace(week) == "" {
		return sql.NullInt64{}
	}
	for _, part := range strings.Fields(week) {
		n, err := strconv.Atoi(part)
		if err == nil && (n == 1 || n == 2) {
			return sql.NullInt64{Int64: int64(n), Valid: true}
		}
	}
	return sql.NullInt64{}
}

func pseudoTelegramID(key string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return -int64(h.Sum64()%9_000_000_000_000) - 1
}
