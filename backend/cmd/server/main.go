package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/golang-migrate/migrate/v4"
	migratesqlite3 "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"

	coffeedb "coffee-of-the-day/backend/db"
	"coffee-of-the-day/backend/config"
	"coffee-of-the-day/backend/internal/handler"
	"coffee-of-the-day/backend/internal/repository"
	"coffee-of-the-day/backend/internal/service"
	"coffee-of-the-day/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("설정 오류: %v", err)
	}

	// _foreign_keys=on: SQLite는 연결마다 외래키 강제를 별도로 활성화해야 한다.
	// _journal_mode=WAL: 읽기/쓰기 동시성을 높이고, Litestream 복제에 필수다.
	db, err := sql.Open("sqlite3", cfg.DBPath+"?_foreign_keys=on&_journal_mode=WAL")
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
		// rate limit: IP 기준 1분에 20회 초과 시 429 반환 (brute-force 방어)
		r.Route("/auth", func(r chi.Router) {
			r.Use(httprate.LimitByIP(20, 1*time.Minute))
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
			// /me는 JWT가 필요하므로 인증 그룹에서 별도로 등록한다
		})

		// 인증 필요 라우트: JWT 쿠키 검증 미들웨어 적용
		r.Group(func(r chi.Router) {
			r.Use(handler.JWTMiddleware(cfg.JWTSecret))

			r.Get("/auth/me", authHandler.Me)

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

	// /api/v1 이외의 모든 경로를 React SPA로 fallback한다.
	// React Router가 클라이언트 사이드에서 라우팅을 처리한다.
	r.Handle("/*", web.Handler())

	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 별도 고루틴에서 서버를 시작해 메인 고루틴이 시그널 수신을 기다릴 수 있도록 한다.
	go func() {
		log.Printf("server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// SIGTERM(컨테이너 종료), SIGINT(Ctrl+C) 수신 대기
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
	log.Println("shutting down server...")

	// 진행 중인 요청이 완료될 때까지 최대 30초 대기
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}

func runMigrations(db *sql.DB) error {
	driver, err := migratesqlite3.WithInstance(db, &migratesqlite3.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	// iofs 소스: embed된 SQL 파일을 사용해 실행 위치에 무관하게 마이그레이션을 실행한다.
	src, err := iofs.New(coffeedb.MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
