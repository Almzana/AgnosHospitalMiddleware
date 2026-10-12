// Command server starts the Hospital Middleware API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Almzana/AgnosHospitalMiddleware/internal/apidocs"
	"github.com/Almzana/AgnosHospitalMiddleware/internal/auth"
	"github.com/Almzana/AgnosHospitalMiddleware/internal/config"
	"github.com/Almzana/AgnosHospitalMiddleware/internal/handler"
	"github.com/Almzana/AgnosHospitalMiddleware/internal/hospitalapi"
	"github.com/Almzana/AgnosHospitalMiddleware/internal/repository"
	"github.com/Almzana/AgnosHospitalMiddleware/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := connectDB(cfg.DatabaseURL)
	if err != nil {
		return err
	}

	// Dependency wiring: repositories -> services -> handlers.
	hospitals := repository.NewHospitalRepository(db)
	staffRepo := repository.NewStaffRepository(db)
	patientRepo := repository.NewPatientRepository(db)

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.TokenTTL)
	his := hospitalapi.NewClient(cfg.HISTimeout)

	authSvc := service.NewAuthService(staffRepo, hospitals, tokens)
	patientSvc := service.NewPatientService(patientRepo, hospitals, his, logger)

	gin.SetMode(gin.ReleaseMode)
	router := handler.NewRouter(
		handler.NewStaffHandler(authSvc, logger),
		handler.NewPatientHandler(patientSvc, logger),
		tokens,
	)
	apidocs.Register(router)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM (docker stop).
	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-stop:
		logger.Info("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
	return nil
}

// connectDB retries for a while because Postgres may still be starting inside docker compose.
func connectDB(dsn string) (*gorm.DB, error) {
	var (
		db  *gorm.DB
		err error
	)
	for attempt := 1; attempt <= 10; attempt++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			return db, nil
		}
		time.Sleep(2 * time.Second)
	}
	return nil, err
}
