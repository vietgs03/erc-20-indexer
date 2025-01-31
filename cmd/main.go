package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vietgs03/erc-20-indexer/config"
	"github.com/vietgs03/erc-20-indexer/internal/indexer"
	"github.com/vietgs03/erc-20-indexer/internal/repository/postgres"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	context.AfterFunc(ctx, func() {
		log.Info().Msg("received interrupt signal, shutting down")
	})

	// Configure zerolog with proper timestamp
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	log.Info().Interface("config", cfg).Msg("loaded config")

	repo, err := postgres.NewRepository(cfg.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer repo.Close()

	indexer := indexer.NewIndexer(&cfg, repo)
	log.Info().Msg("starting indexer")
	if err := indexer.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start indexer")
	}
	log.Info().Msg("indexer stopped")
}
