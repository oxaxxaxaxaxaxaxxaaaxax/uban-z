package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	bookingserver "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/bookingserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/events/noop"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/events/rabbitmq"
	bookinghttp "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/http"
	bookingpostgres "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/postgres"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/parser/httpparser"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/config"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/port"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/service"
	parserdomain "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/domain"
	parserservice "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/service"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/logging"
)

const dbConnectTimeout = 10 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("config load failed", slog.Any("err", err))
		os.Exit(1)
	}

	logger, err := logging.New(cfg.LogLevel)
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("logger init failed", slog.Any("err", err))
		os.Exit(1)
	}

	pool, err := openPostgres(cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres connect failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to postgres")

	publisher, closePublisher, err := buildPublisher(cfg, logger)
	if err != nil {
		logger.Error("event publisher init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer closePublisher()

	store := bookingpostgres.NewStoreFromPool(pool)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	importStatus := newImportStatusTracker()
	startStartupScheduleImport(ctx, cfg, store, logger, importStatus)

	useCase := service.New(store, store, publisher)
	handler := bookinghttp.NewHandler(useCase, logger)
	mux := http.NewServeMux()
	mux.Handle("GET /parser/status", importStatus)
	mux.Handle("/", bookingserver.Handler(handler))

	router := httpx.Chain(
		mux,
		httpx.ParseToken([]byte(cfg.JWTSecret)),
		httpx.RequestID,
		httpx.RecoverPanic(logger),
		httpx.AccessLog(logger),
	)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: cfg.ShutdownTimeout,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown failed", slog.Any("err", err))
		}
	}()

	logger.Info("booking service starting",
		slog.String("addr", server.Addr),
		slog.String("log_level", cfg.LogLevel),
		slog.Bool("events_enabled", cfg.EventsEnabled),
	)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed", slog.Any("err", err))
		os.Exit(1)
	}
}

func startStartupScheduleImport(ctx context.Context, cfg config.Config, store *bookingpostgres.Store, logger *slog.Logger, status *importStatusTracker) {
	logger.Info("parser startup import started")
	status.markRunning()
	go func() {
		stats, err := importStartupSchedule(ctx, cfg, store, logger)
		if err != nil {
			status.markFailed(err)
			logger.Error("parser startup import failed", slog.Any("err", err))
			return
		}
		status.markReady(stats)
	}()
}

type importStatusTracker struct {
	mu          sync.RWMutex
	status      string
	err         string
	stats       parserdomain.ImportStats
	startedAt   time.Time
	completedAt *time.Time
}

type importStatusResponse struct {
	Status      string                    `json:"status"`
	Error       string                    `json:"error,omitempty"`
	Stats       *parserdomain.ImportStats `json:"stats,omitempty"`
	StartedAt   string                    `json:"started_at,omitempty"`
	CompletedAt string                    `json:"completed_at,omitempty"`
}

func newImportStatusTracker() *importStatusTracker {
	return &importStatusTracker{status: "pending"}
}

func (s *importStatusTracker) markRunning() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = "running"
	s.err = ""
	s.stats = parserdomain.ImportStats{}
	s.startedAt = time.Now().UTC()
	s.completedAt = nil
}

func (s *importStatusTracker) markReady(stats parserdomain.ImportStats) {
	s.mu.Lock()
	defer s.mu.Unlock()

	completedAt := time.Now().UTC()
	s.status = "ready"
	s.err = ""
	s.stats = stats
	s.completedAt = &completedAt
}

func (s *importStatusTracker) markFailed(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	completedAt := time.Now().UTC()
	s.status = "failed"
	s.err = err.Error()
	s.completedAt = &completedAt
}

func (s *importStatusTracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := importStatusResponse{
		Status:    s.status,
		Error:     s.err,
		StartedAt: formatOptionalTime(s.startedAt),
	}
	if s.completedAt != nil {
		response.CompletedAt = s.completedAt.Format(time.RFC3339)
	}
	if s.status == "ready" {
		stats := s.stats
		response.Stats = &stats
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func buildPublisher(cfg config.Config, logger *slog.Logger) (port.EventPublisher, func(), error) {
	if !cfg.EventsEnabled {
		logger.Info("event publishing disabled")
		return noop.Publisher{}, func() {}, nil
	}

	pub, err := rabbitmq.NewPublisher(cfg.RabbitMQURL, cfg.RabbitMQExchange, logger)
	if err != nil {
		return nil, nil, err
	}
	logger.Info("connected to rabbitmq",
		slog.String("exchange", cfg.RabbitMQExchange),
	)
	return pub, func() {
		if err := pub.Close(); err != nil {
			logger.Warn("rabbitmq publisher close", slog.Any("err", err))
		}
	}, nil
}

func openPostgres(url string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbConnectTimeout)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func importStartupSchedule(ctx context.Context, cfg config.Config, store *bookingpostgres.Store, logger *slog.Logger) (parserdomain.ImportStats, error) {
	location, err := time.LoadLocation(cfg.ParserTimezone)
	if err != nil {
		return parserdomain.ImportStats{}, err
	}

	source, err := httpparser.New(cfg.ParserBaseURL, cfg.ParserTimeout)
	if err != nil {
		return parserdomain.ImportStats{}, err
	}

	parser, err := parserservice.New(source, store, parserservice.Config{
		WeeksAhead:      cfg.ParserWeeksAhead,
		DefaultBuilding: cfg.ParserDefaultBuilding,
		DefaultCapacity: cfg.ParserDefaultCapacity,
		Location:        location,
	})
	if err != nil {
		return parserdomain.ImportStats{}, err
	}

	stats, err := parser.Run(ctx)
	if err != nil {
		return parserdomain.ImportStats{}, err
	}

	logger.Info("parser startup import completed",
		slog.Int("rooms_seen", stats.RoomsSeen),
		slog.Int("rooms_imported", stats.RoomsImported),
		slog.Int("lessons_seen", stats.LessonsSeen),
		slog.Int("lessons_expanded", stats.LessonsExpanded),
		slog.Int("lessons_imported", stats.LessonsImported),
		slog.Int("lessons_skipped", stats.LessonsSkipped),
	)
	return stats, nil
}
