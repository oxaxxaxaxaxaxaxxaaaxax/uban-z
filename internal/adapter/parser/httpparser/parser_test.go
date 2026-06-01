package httpparser_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/parser/httpparser"
)

func TestParser_ParseRoomsAndRoomLessons(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/room", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`
			<html><body>
				<a class="tutors_item" href="/room/101">3107 (Новый корпус)</a>
			</body></html>
		`))
	})
	mux.HandleFunc("/room/101", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`
			<html><body>
				<table class="time-table">
					<tr><th></th><th>Понедельник</th><th>Вторник</th></tr>
					<tr>
						<td>09:00</td>
						<td>
							<div class="cell">
								<span class="type" title="Лекция">лек.</span>
								<div class="subject">Мат анализ</div>
								<a class="tutor">Иванов И.И.</a>
								<div class="groups"><a class="group">22201</a><a class="group">22202</a></div>
								<div class="week">числ.</div>
							</div>
							<div class="cell">
								<div class="subject">Забронировано на кафедру</div>
							</div>
						</td>
						<td></td>
					</tr>
				</table>
			</body></html>
		`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	parser, err := httpparser.New(server.URL, time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rooms, err := parser.ParseRooms(context.Background())
	if err != nil {
		t.Fatalf("ParseRooms: %v", err)
	}
	if len(rooms) != 1 {
		t.Fatalf("rooms len = %d, want 1", len(rooms))
	}
	if rooms[0].Name != "3107 (Новый корпус)" {
		t.Fatalf("room name = %q", rooms[0].Name)
	}
	if rooms[0].FullURL != server.URL+"/room/101" {
		t.Fatalf("room url = %q", rooms[0].FullURL)
	}

	lessons, err := parser.ParseLessonsRoom(context.Background(), rooms[0].FullURL)
	if err != nil {
		t.Fatalf("ParseLessonsRoom: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("lessons len = %d, want 1", len(lessons))
	}
	got := lessons[0]
	if got.Subject != "Мат анализ" || got.LessonType != "лек." || got.Teacher != "Иванов И.И." {
		t.Fatalf("lesson = %+v", got)
	}
	if got.StartTime != "09:00" || got.Weekday != "Понедельник" || got.Week != "числ." {
		t.Fatalf("lesson time fields = %+v", got)
	}
	if len(got.GroupNumbers) != 2 || got.GroupNumbers[0] != "22201" || got.GroupNumbers[1] != "22202" {
		t.Fatalf("groups = %v", got.GroupNumbers)
	}
}
