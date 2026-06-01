package httpparser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	parserdomain "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/domain"
)

const (
	defaultBaseURL = "https://table.nsu.ru"
	roomsPath      = "/room"
)

var timePattern = regexp.MustCompile(`\b\d{1,2}:\d{2}\b`)

type Parser struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func New(baseURL string, timeout time.Duration) (*Parser, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse parser base url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("parser base url must include scheme and host")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &Parser{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (p *Parser) ParseRooms(ctx context.Context) ([]parserdomain.RoomSelector, error) {
	doc, err := p.fetch(ctx, p.resolve(roomsPath))
	if err != nil {
		return nil, err
	}

	rooms := make([]parserdomain.RoomSelector, 0)
	doc.Find("a.tutors_item").Each(func(i int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Text())
		href, ok := s.Attr("href")
		if !ok || name == "" {
			return
		}
		rooms = append(rooms, parserdomain.RoomSelector{
			Name:    name,
			FullURL: p.resolve(href),
		})
	})

	return rooms, nil
}

func (p *Parser) ParseLessonsRoom(ctx context.Context, roomURL string) ([]parserdomain.LessonRoom, error) {
	doc, err := p.fetch(ctx, p.resolve(roomURL))
	if err != nil {
		return nil, err
	}

	table := doc.Find("table.time-table").First()
	if table.Length() == 0 {
		return nil, fmt.Errorf("time-table not found")
	}

	days := make([]string, 0)
	table.Find("tr").First().Find("th").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		days = append(days, strings.TrimSpace(s.Text()))
	})

	lessons := make([]parserdomain.LessonRoom, 0)
	table.Find("tr").Each(func(i int, tr *goquery.Selection) {
		if i == 0 {
			return
		}

		tds := tr.Find("td")
		if tds.Length() == 0 {
			return
		}
		startTime := strings.TrimSpace(tds.Eq(0).Text())
		startTime = extractStartTime(startTime)
		if startTime == "" {
			return
		}

		for col := 1; col < tds.Length(); col++ {
			weekdayIdx := col - 1
			if weekdayIdx < 0 || weekdayIdx >= len(days) {
				continue
			}
			weekday := days[weekdayIdx]
			td := tds.Eq(col)
			td.Find("div.cell").Each(func(i int, cell *goquery.Selection) {
				lessonType := strings.TrimSpace(cell.Find("span.type").First().Text())
				subject := strings.TrimSpace(cell.Find("div.subject").First().Text())
				if subject == "" || isReservedSubject(subject) {
					return
				}
				teacher := strings.TrimSpace(cell.Find("a.tutor").First().Text())

				groups := make([]string, 0)
				cell.Find("div.groups").Find("a.group").Each(func(i int, group *goquery.Selection) {
					groupName := strings.TrimSpace(group.Text())
					if groupName != "" {
						groups = append(groups, groupName)
					}
				})

				lessons = append(lessons, parserdomain.LessonRoom{
					LessonType:   lessonType,
					Subject:      subject,
					Teacher:      teacher,
					StartTime:    startTime,
					Weekday:      weekday,
					Room:         strings.TrimSpace(cell.Find("div.room a").First().Text()),
					Week:         strings.TrimSpace(cell.Find("div.week").First().Text()),
					GroupNumbers: groups,
				})
			})
		}
	})

	return lessons, nil
}

func extractStartTime(text string) string {
	return timePattern.FindString(text)
}

func isReservedSubject(subject string) bool {
	if strings.HasPrefix(strings.ToLower(subject), "забронировано") {
		return true
	}
	return false
}

func (p *Parser) fetch(ctx context.Context, target string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("new request %q: %w", target, err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", target, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("fetch %q: status %d: %s", target, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html %q: %w", target, err)
	}
	return doc, nil
}

func (p *Parser) resolve(href string) string {
	parsed, err := url.Parse(href)
	if err != nil {
		return href
	}
	return p.baseURL.ResolveReference(parsed).String()
}
