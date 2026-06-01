package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"timetable-homework-tgbot/internal/domain"
	"timetable-homework-tgbot/internal/infrastracture/database"
)

type HomeworkRepository interface {
	SaveForGroup(ctx context.Context, teacherUserID int64, group, subject, text string) error
	Update(ctx context.Context, teacherUserID, homeworkID int64, newText string) error
	UpdateStatus(ctx context.Context, userID, homeworkID int64) error
	DeleteByTeacher(ctx context.Context, teacherUserID, homeworkID int64) error
	ListForTeacher(ctx context.Context, teacherUserID int64) ([]domain.HWBrief, error)
	ListForUser(ctx context.Context, userID int64) ([]domain.HWBrief, error)
	CheckTeacherHomework(ctx context.Context, teacherUserID, homeworkID int64) (bool, error)
	CheckExistence(ctx context.Context, userID, homeworkID int64) (bool, error)
}

type HomeworkRepo struct {
	db *database.DB
}

func NewHomeworkRepo(db *database.DB) *HomeworkRepo {
	return &HomeworkRepo{db: db}
}

func (r *HomeworkRepo) SaveForGroup(ctx context.Context, teacherUserID int64, group, subject, text string) error {
	tx, err := r.db.GetSql().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SaveForGroup begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var teacherID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM teachers WHERE user_id = $1`, teacherUserID).Scan(&teacherID); err != nil {
		return fmt.Errorf("SaveForGroup teacher: %w", err)
	}

	var groupID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM groups WHERE group_name = $1`, group).Scan(&groupID); err != nil {
		return fmt.Errorf("SaveForGroup group: %w", err)
	}

	var homeworkID int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO homeworks (text, status, teacher_id)
VALUES ($1, $2, $3)
RETURNING id
`, fmt.Sprintf("%s: %s", subject, text), "new", teacherID).Scan(&homeworkID); err != nil {
		return fmt.Errorf("SaveForGroup homework: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO group_homeworks (group_id, homework_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, groupID, homeworkID); err != nil {
		return fmt.Errorf("SaveForGroup link: %w", err)
	}

	return tx.Commit()
}

func (r *HomeworkRepo) Update(ctx context.Context, teacherUserID, homeworkID int64, newText string) error {
	newText = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(newText), "•"))
	res, err := r.db.GetSql().ExecContext(ctx, `
UPDATE homeworks h
SET text = CASE
    WHEN POSITION(':' IN h.text) > 0 THEN LEFT(h.text, POSITION(':' IN h.text)) || ' • ' || $3
    ELSE $3
END
FROM teachers t
WHERE h.id = $1
  AND h.teacher_id = t.id
  AND t.user_id = $2
`, homeworkID, teacherUserID, newText)
	if err != nil {
		return fmt.Errorf("Update homework exec: %w", err)
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return fmt.Errorf("Update homework: not found or forbidden (id=%d)", homeworkID)
	}
	return nil
}

func (r *HomeworkRepo) ListForUser(ctx context.Context, userID int64) ([]domain.HWBrief, error) {
	const q = `
SELECT h.id, h.text, COALESCE(shs.status, 'new')
FROM homeworks h
JOIN group_homeworks gh ON gh.homework_id = h.id
JOIN students s ON s.group_id = gh.group_id
LEFT JOIN student_homework_statuses shs ON shs.homework_id = h.id AND shs.student_id = s.id
WHERE s.user_id = $1
ORDER BY h.id DESC;
`
	rows, err := r.db.GetSql().QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("ListForUser query: %w", err)
	}
	defer rows.Close()

	var res []domain.HWBrief
	for rows.Next() {
		var hw domain.HWBrief
		if err := rows.Scan(&hw.ID, &hw.HomeworkText, &hw.Status); err != nil {
			return nil, fmt.Errorf("ListForUser scan: %w", err)
		}
		hw.Subject = fmt.Sprintf("#%d", hw.ID)
		res = append(res, hw)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListForUser rows: %w", err)
	}
	return res, nil
}

func (r *HomeworkRepo) ListForTeacher(ctx context.Context, teacherUserID int64) ([]domain.HWBrief, error) {
	const q = `
SELECT h.id, h.text, COALESCE(h.status, ''), string_agg(g.group_name, ', ' ORDER BY g.group_name)
FROM homeworks h
JOIN group_homeworks gh ON gh.homework_id = h.id
JOIN groups g ON g.id = gh.group_id
JOIN teachers t ON t.id = h.teacher_id
WHERE t.user_id = $1
GROUP BY h.id
ORDER BY h.id DESC;
`
	rows, err := r.db.GetSql().QueryContext(ctx, q, teacherUserID)
	if err != nil {
		return nil, fmt.Errorf("ListForTeacher query: %w", err)
	}
	defer rows.Close()

	var res []domain.HWBrief
	for rows.Next() {
		var hw domain.HWBrief
		var groups string
		if err := rows.Scan(&hw.ID, &hw.HomeworkText, &hw.Status, &groups); err != nil {
			return nil, fmt.Errorf("ListForTeacher scan: %w", err)
		}
		hw.Subject = fmt.Sprintf("#%d", hw.ID)
		if groups != "" {
			hw.HomeworkText = fmt.Sprintf("[%s] %s", groups, hw.HomeworkText)
		}
		res = append(res, hw)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListForTeacher rows: %w", err)
	}
	return res, nil
}

func (r *HomeworkRepo) UpdateStatus(ctx context.Context, userID, homeworkID int64) error {
	res, err := r.db.GetSql().ExecContext(ctx, `
INSERT INTO student_homework_statuses (student_id, homework_id, status)
SELECT s.id, h.id, $3
FROM homeworks h
JOIN group_homeworks gh ON gh.homework_id = h.id
JOIN students s ON s.group_id = gh.group_id
WHERE s.user_id = $1
  AND h.id = $2
ON CONFLICT (student_id, homework_id) DO UPDATE
SET status = EXCLUDED.status
`, userID, homeworkID, "done")
	if err != nil {
		return fmt.Errorf("UpdateStatus exec: %w", err)
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return fmt.Errorf("UpdateStatus homework: not found or forbidden (id=%d, user_id=%d)", homeworkID, userID)
	}
	return nil
}

func (r *HomeworkRepo) DeleteByTeacher(ctx context.Context, teacherUserID, homeworkID int64) error {
	res, err := r.db.GetSql().ExecContext(ctx, `
DELETE FROM homeworks h
USING teachers t
WHERE h.id = $1
  AND h.teacher_id = t.id
  AND t.user_id = $2
`, homeworkID, teacherUserID)
	if err != nil {
		return fmt.Errorf("homeworks.DeleteByTeacher exec: %w", err)
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return fmt.Errorf("homeworks.DeleteByTeacher: not found or forbidden (id=%d)", homeworkID)
	}
	return nil
}

func (r *HomeworkRepo) CheckTeacherHomework(ctx context.Context, teacherUserID, homeworkID int64) (bool, error) {
	const q = `
SELECT 1
FROM homeworks h
JOIN teachers t ON t.id = h.teacher_id
WHERE t.user_id = $1
  AND h.id = $2
LIMIT 1;
`
	var dummy int
	err := r.db.GetSql().QueryRowContext(ctx, q, teacherUserID, homeworkID).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("homeworks.CheckTeacherHomework query: %w", err)
	}
	return true, nil
}

func (r *HomeworkRepo) CheckExistence(ctx context.Context, userID, homeworkID int64) (bool, error) {
	const q = `
SELECT 1
FROM homeworks h
JOIN group_homeworks gh ON gh.homework_id = h.id
JOIN students s ON s.group_id = gh.group_id
WHERE s.user_id = $1
  AND h.id = $2
LIMIT 1;
`
	var dummy int
	err := r.db.GetSql().QueryRowContext(ctx, q, userID, homeworkID).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("homeworks.CheckExistence query: %w", err)
	}
	return true, nil
}
