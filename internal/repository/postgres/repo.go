package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/vietgs03/erc-20-indexer/internal/abi"
)

type Repository struct {
	db        *sql.DB
	batchSize int
	batchCh   chan *abi.Erc20Transfer
	done      chan struct{}
	workers   int // Số worker xử lý batch
	stats     struct {
		processedEvents uint64
		failedEvents    uint64
		batchesSaved    uint64
		errors          uint64
	}
}

type Transfer struct {
	TxHash      string
	BlockNumber uint64
	From        string
	To          string
	Value       string
	Timestamp   time.Time
}

func NewRepository(dsn string) (*Repository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Set connection pool limits
	db.SetMaxOpenConns(25) // Tăng số lượng connections
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	r := &Repository{
		db:        db,
		batchSize: 5000,                                 // Tăng batch size
		batchCh:   make(chan *abi.Erc20Transfer, 50000), // Tăng buffer size
		done:      make(chan struct{}),
		workers:   5, // Số worker xử lý batch
	}

	// Start multiple batch writers
	for i := 0; i < r.workers; i++ {
		go r.batchWriter()
	}
	return r, nil
}

func runMigrations(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS transfers (
		id SERIAL PRIMARY KEY,
		tx_hash VARCHAR(66) NOT NULL,
		block_number BIGINT NOT NULL,
		from_address VARCHAR(42) NOT NULL,
		to_address VARCHAR(42) NOT NULL,
		value NUMERIC NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_transfers_block_number ON transfers(block_number);
	CREATE INDEX IF NOT EXISTS idx_transfers_from_address ON transfers(from_address);
	CREATE INDEX IF NOT EXISTS idx_transfers_to_address ON transfers(to_address);
	`

	_, err := db.Exec(query)
	return err
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) batchWriter() {
	var batch []*abi.Erc20Transfer
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case transfer := <-r.batchCh:
			batch = append(batch, transfer)
			if len(batch) >= r.batchSize {
				r.saveBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				r.saveBatch(batch)
				batch = batch[:0]
			}
		case <-r.done:
			if len(batch) > 0 {
				r.saveBatch(batch)
			}
			return
		}
	}
}

func (r *Repository) saveBatch(batch []*abi.Erc20Transfer) {
	// Add retry logic
	maxRetries := 3
	for retry := 0; retry < maxRetries; retry++ {
		if err := r.executeBatch(batch); err != nil {
			log.Error().Err(err).Int("retry", retry).Msg("failed to save batch")
			time.Sleep(time.Second * time.Duration(retry+1))
			continue
		}
		return
	}
}

func (r *Repository) executeBatch(batch []*abi.Erc20Transfer) error {
	if len(batch) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		log.Error().Err(err).Msg("failed to begin transaction")
		return err
	}

	stmt, err := tx.Prepare(pq.CopyIn("transfers", "tx_hash", "block_number", "from_address", "to_address", "value", "timestamp"))
	if err != nil {
		log.Error().Err(err).Msg("failed to prepare statement")
		tx.Rollback()
		return err
	}

	for _, transfer := range batch {
		_, err := stmt.Exec(
			transfer.Raw.TxHash.Hex(),
			transfer.Raw.BlockNumber,
			transfer.From.Hex(),
			transfer.To.Hex(),
			transfer.Value.String(),
			time.Now(),
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to execute statement")
			tx.Rollback()
			return err
		}
	}

	if err := stmt.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close statement")
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("failed to commit transaction")
		return err
	}
	return nil
}

func (r *Repository) SaveEvent(ctx context.Context, event *abi.Erc20Transfer) error {
	select {
	case r.batchCh <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Repository) logStats() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		log.Info().
			Uint64("processed_events", atomic.LoadUint64(&r.stats.processedEvents)).
			Uint64("failed_events", atomic.LoadUint64(&r.stats.failedEvents)).
			Uint64("batches_saved", atomic.LoadUint64(&r.stats.batchesSaved)).
			Uint64("errors", atomic.LoadUint64(&r.stats.errors)).
			Msg("repository stats")
	}
}
