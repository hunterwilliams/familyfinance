package main

import (
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hunterwilliams/familyfinance/internal/config"
	"github.com/hunterwilliams/familyfinance/internal/handler"
	appmigrate "github.com/hunterwilliams/familyfinance/internal/migrate"
	"github.com/hunterwilliams/familyfinance/internal/repository"
	"github.com/hunterwilliams/familyfinance/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.Load()

	if err := appmigrate.Up(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		log.Fatal().Err(err).Msg("migration failed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("database connection failed")
	}
	defer pool.Close()

	repo := repository.New(pool)
	svc := service.New(repo)
	if _, err := svc.EnsureHousehold(ctx); err != nil {
		log.Fatal().Err(err).Msg("household setup failed")
	}

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.MaxMultipartMemory = 12 << 20
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	h := handler.New(svc, cfg)
	h.Register(r)

	log.Info().Str("port", cfg.Port).Msg("starting server")
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
