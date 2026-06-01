package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	parserdomain "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/port"
)

const defaultLessonDuration = 95 * time.Minute

var timePattern = regexp.MustCompile(`(\d{1,2})[:.](\d{2})`)

type Config struct {
	WeeksAhead      int
	DefaultBuilding string
	DefaultCapacity int
	Location        *time.Location
	Now             func() time.Time
}

type Service struct {
	source port.Source
	repo   port.ScheduleRepository
	cfg    Config
}

func New(source port.Source, repo port.ScheduleRepository, cfg Config) (*Service, error) {
	if source == nil {
		return nil, errors.New("parser source is required")
	}
	if repo == nil {
		return nil, errors.New("schedule repository is required")
	}
	if cfg.WeeksAhead <= 0 {
		return nil, errors.New("weeks ahead must be positive")
	}
	if cfg.DefaultCapacity <= 0 {
		return nil, errors.New("default capacity must be positive")
	}
	if strings.TrimSpace(cfg.DefaultBuilding) == "" {
		return nil, errors.New("default building is required")
	}
	if cfg.Location == nil {
		cfg.Location = time.Local
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	return &Service{source: source, repo: repo, cfg: cfg}, nil
}

func (s *Service) Run(ctx context.Context) (parserdomain.ImportStats, error) {
	rooms, err := s.source.ParseRooms(ctx)
	if err != nil {
		return parserdomain.ImportStats{}, fmt.Errorf("parse rooms: %w", err)
	}

	normalizedRooms := make([]parserdomain.RoomSelector, 0, len(rooms))
	for _, room := range rooms {
		roomName, building := normalizeRoom(room.Name, s.cfg.DefaultBuilding)
		normalizedRooms = append(normalizedRooms, parserdomain.RoomSelector{
			Name:     roomName,
			Building: building,
			Capacity: s.cfg.DefaultCapacity,
			FullURL:  room.FullURL,
		})
	}

	slots := make([]parserdomain.ScheduleSlot, 0)
	stats := parserdomain.ImportStats{RoomsSeen: len(rooms)}
	for _, room := range rooms {
		lessons, err := s.source.ParseLessonsRoom(ctx, room.FullURL)
		if err != nil {
			return parserdomain.ImportStats{}, fmt.Errorf("parse room %q: %w", room.Name, err)
		}
		stats.LessonsSeen += len(lessons)

		for _, lesson := range lessons {
			expanded, err := s.expand(room, lesson)
			if err != nil {
				return parserdomain.ImportStats{}, fmt.Errorf("expand lesson %q in room %q: %w", lesson.Subject, room.Name, err)
			}
			slots = append(slots, expanded...)
			stats.LessonsExpanded += len(expanded)
		}
	}

	importStats, err := s.repo.ReplaceParsedSchedule(ctx, normalizedRooms, slots)
	if err != nil {
		return parserdomain.ImportStats{}, fmt.Errorf("replace parsed schedule: %w", err)
	}
	importStats.RoomsSeen = stats.RoomsSeen
	importStats.LessonsSeen = stats.LessonsSeen
	importStats.LessonsExpanded = stats.LessonsExpanded

	return importStats, nil
}

func (s *Service) expand(room parserdomain.RoomSelector, lesson parserdomain.LessonRoom) ([]parserdomain.ScheduleSlot, error) {
	startHour, startMinute, err := parseClock(lesson.StartTime)
	if err != nil {
		return nil, err
	}

	weekday, ok := parseWeekday(lesson.Weekday)
	if !ok {
		return nil, fmt.Errorf("unknown weekday %q", lesson.Weekday)
	}

	now := s.cfg.Now().In(s.cfg.Location)
	monday := weekStart(now)
	slots := make([]parserdomain.ScheduleSlot, 0, s.cfg.WeeksAhead)
	for week := 0; week < s.cfg.WeeksAhead; week++ {
		date := monday.AddDate(0, 0, week*7+weekdayOffset(weekday))
		if !matchesWeek(lesson.Week, date) {
			continue
		}

		start := time.Date(date.Year(), date.Month(), date.Day(), startHour, startMinute, 0, 0, s.cfg.Location)
		end := start.Add(defaultLessonDuration)
		roomName, building := normalizeRoom(room.Name, s.cfg.DefaultBuilding)

		slots = append(slots, parserdomain.ScheduleSlot{
			RoomName:     roomName,
			Building:     building,
			Capacity:     s.cfg.DefaultCapacity,
			Subject:      lesson.Subject,
			LessonType:   lesson.LessonType,
			Teacher:      lesson.Teacher,
			GroupNumbers: append([]string(nil), lesson.GroupNumbers...),
			Week:         lesson.Week,
			StartTime:    start,
			EndTime:      end,
		})
	}

	return slots, nil
}

func parseClock(value string) (int, int, error) {
	match := timePattern.FindStringSubmatch(value)
	if match == nil {
		return 0, 0, fmt.Errorf("cannot parse lesson start time %q", value)
	}

	hour, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse lesson hour %q: %w", value, err)
	}
	minute, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, fmt.Errorf("parse lesson minute %q: %w", value, err)
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("lesson start time out of range %q", value)
	}
	return hour, minute, nil
}

func parseWeekday(value string) (time.Weekday, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimSuffix(normalized, ".")

	switch {
	case strings.HasPrefix(normalized, "пн"), strings.HasPrefix(normalized, "пон"):
		return time.Monday, true
	case strings.HasPrefix(normalized, "вт"), strings.HasPrefix(normalized, "вто"):
		return time.Tuesday, true
	case strings.HasPrefix(normalized, "ср"), strings.HasPrefix(normalized, "сре"):
		return time.Wednesday, true
	case strings.HasPrefix(normalized, "чт"), strings.HasPrefix(normalized, "чет"):
		return time.Thursday, true
	case strings.HasPrefix(normalized, "пт"), strings.HasPrefix(normalized, "пят"):
		return time.Friday, true
	case strings.HasPrefix(normalized, "сб"), strings.HasPrefix(normalized, "суб"):
		return time.Saturday, true
	case strings.HasPrefix(normalized, "вс"), strings.HasPrefix(normalized, "вос"):
		return time.Sunday, true
	default:
		return 0, false
	}
}

func weekStart(t time.Time) time.Time {
	dayOffset := int(t.Weekday() - time.Monday)
	if dayOffset < 0 {
		dayOffset = 6
	}
	date := t.AddDate(0, 0, -dayOffset)
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, t.Location())
}

func weekdayOffset(day time.Weekday) int {
	if day == time.Sunday {
		return 6
	}
	return int(day - time.Monday)
}

func matchesWeek(week string, date time.Time) bool {
	normalized := strings.ToLower(strings.TrimSpace(week))
	if normalized == "" {
		return true
	}

	_, isoWeek := date.ISOWeek()
	switch {
	case strings.Contains(normalized, "числ"), strings.Contains(normalized, "нечет"), strings.Contains(normalized, "неч"):
		return isoWeek%2 != 0
	case strings.Contains(normalized, "знам"), strings.Contains(normalized, "чет"):
		return isoWeek%2 == 0
	default:
		return true
	}
}

func normalizeRoom(raw string, defaultBuilding string) (string, string) {
	room := strings.TrimSpace(raw)
	if room == "" {
		return "unknown", defaultBuilding
	}

	open := strings.LastIndex(room, "(")
	close := strings.LastIndex(room, ")")
	if open > 0 && close > open {
		building := strings.TrimSpace(room[open+1 : close])
		name := strings.TrimSpace(room[:open])
		if name != "" && building != "" {
			return name, building
		}
	}

	return room, defaultBuilding
}
