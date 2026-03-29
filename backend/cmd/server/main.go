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

	suggestionRepo := repository.NewSQLiteSuggestionRepository(db)
	suggestionSvc := service.NewSuggestionService(suggestionRepo)
	suggestionHandler := handler.NewSuggestionHandler(suggestionSvc)

	userRepo := repository.NewSQLiteUserRepository(db)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authSvc, cfg.IsProduction)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// CORS는 전역으로 적용: OPTIONS preflight가 JWTMiddleware에 도달하지 않도록
	r.Use(handler.CORSMiddleware)

	r.Options("/*", func(w http.ResponseWriter, r *http.Request) {})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	r.Route("/api/v1", func(r chi.Router) {
		// 인증 불필요 라우트: 회원가입·로그인·토큰 갱신·로그아웃
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		// 인증 필요 라우트: JWT 쿠키 검증 미들웨어 적용
		r.Group(func(r chi.Router) {
			r.Use(handler.JWTMiddleware(cfg.JWTSecret))

			r.Route("/logs", func(r chi.Router) {
				r.Post("/", logHandler.CreateLog)
				r.Get("/", logHandler.ListLogs)
				r.Get("/{id}", logHandler.GetLog)
				r.Put("/{id}", logHandler.UpdateLog)
				r.Delete("/{id}", logHandler.DeleteLog)
			})

			r.Route("/suggestions", func(r chi.Router) {
				r.Get("/tags", suggestionHandler.GetTagSuggestions)
				r.Get("/companions", suggestionHandler.GetCompanionSuggestions)
			})
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
