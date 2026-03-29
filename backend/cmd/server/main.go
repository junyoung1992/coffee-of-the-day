package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	migratesqlite3 "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"

	"coffee-of-the-day/backend/config"
	"coffee-of-the-day/backend/internal/handler"
	"coffee-of-the-day/backend/internal/repository"
	"coffee-of-the-day/backend/internal/service"
)

func main() {
	cfg := config.Load()

	// _foreign_keys=on: SQLite는 연결마다 외래키 강제를 별도로 활성화해야 한다.
	// DSN 수준에서 설정하면 connection pool의 모든 연결에 일괄 적용된다.
	db, err := sql.Open("sqlite3", cfg.DBPath+"?_foreign_keys=on")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// 의존성 연결: Repository → Service → Handler
	logRepo := repository.NewSQLiteLogRepository(db)
	logSvc := service.NewLogService(logRepo)
	logHandler := handler.NewLogHandler(logSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// CORS는 전역으로 적용: OPTIONS preflight가 UserIDMiddleware에 도달하지 않도록
	r.Use(handler.CORSMiddleware)

	// OPTIONS 와일드카드: CORS 미들웨어가 preflight를 처리할 수 있도록 라우트를 열어둔다
	r.Options("/*", func(w http.ResponseWriter, r *http.Request) {})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/logs", func(r chi.Router) {
			// UserID 미들웨어: POC에서 X-User-Id 헤더로 사용자 식별
			r.Use(handler.UserIDMiddleware)

			r.Post("/", logHandler.CreateLog)
			r.Get("/", logHandler.ListLogs)
			r.Get("/{id}", logHandler.GetLog)
			r.Put("/{id}", logHandler.UpdateLog)
			r.Delete("/{id}", logHandler.DeleteLog)
		})
	})

	addr := ":" + cfg.Port
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func runMigrations(db *sql.DB) error {
	driver, err := migratesqlite3.WithInstance(db, &migratesqlite3.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
