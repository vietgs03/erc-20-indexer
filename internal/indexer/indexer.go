package indexer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/vietgs03/erc-20-indexer/config"
	"github.com/vietgs03/erc-20-indexer/internal/abi"
	"github.com/vietgs03/erc-20-indexer/internal/rate"
)

type Repository interface {
	SaveEvent(ctx context.Context, event *abi.Erc20Transfer) error
}
type Indexer struct {
	repo        Repository
	config      *config.Config
	rateLimiter *rate.Limiter
}

func NewIndexer(config *config.Config, repo Repository) *Indexer {
	return &Indexer{
		repo:        repo,
		config:      config,
		rateLimiter: rate.NewLimiter(rate.Every(500*time.Millisecond), 5),
	}
}

func (i *Indexer) dialRPC(ctx context.Context) (*ethclient.Client, error) {
	// Try main RPC first
	client, err := ethclient.DialContext(ctx, i.config.RPC)
	if err == nil {
		return client, nil
	}

	// Try fallback RPCs
	for _, rpc := range i.config.FallbackRPCs {
		client, err := ethclient.DialContext(ctx, rpc)
		if err == nil {
			return client, nil
		}
	}

	return nil, fmt.Errorf("all RPC endpoints failed")
}

func (i *Indexer) Start(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 24*time.Hour)
	defer cancel()

	client, err := i.dialRPC(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	erc20, err := abi.NewErc20(i.config.TokenAddress, client)
	if err != nil {
		return err
	}

	currentBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return err
	}

	startBlock := currentBlock - 600000
	batchSize := uint64(500)
	totalBlocks := currentBlock - startBlock
	processedBlocks := uint64(0)

	for start := startBlock; start < currentBlock; start += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			end := start + batchSize
			if end > currentBlock {
				end = currentBlock
			}

			log.Info().
				Uint64("start_block", start).
				Uint64("end_block", end).
				Msg("processing block range")

			if err := i.processBlockRange(ctx, start, end, erc20); err != nil {
				log.Error().Err(err).
					Uint64("start", start).
					Uint64("end", end).
					Msg("failed to process block range")
				continue
			}

			processedBlocks += end - start
			progress := float64(processedBlocks) / float64(totalBlocks) * 100
			log.Info().
				Float64("progress", progress).
				Uint64("processed_blocks", processedBlocks).
				Uint64("total_blocks", totalBlocks).
				Msg("indexing progress")
		}
	}

	return i.watchNewBlocks(ctx, client, erc20, currentBlock)
}

func (i *Indexer) processBlockRange(ctx context.Context, from, to uint64, erc20 *abi.Erc20) error {
	maxRetries := 3
	backoff := time.Second

	for retry := 0; retry <= maxRetries; retry++ {
		if retry > 0 {
			time.Sleep(backoff * time.Duration(retry))
		}

		opts := &bind.FilterOpts{
			Start:   from,
			End:     &to,
			Context: ctx,
		}

		iter, err := erc20.FilterTransfer(opts, nil, nil)
		if err != nil {
			if strings.Contains(err.Error(), "429 Too Many Requests") {
				log.Warn().
					Uint64("from", from).
					Uint64("to", to).
					Int("retry", retry).
					Msg("rate limited, retrying...")
				continue
			}
			return fmt.Errorf("failed to filter transfers: %v", err)
		}
		defer iter.Close()

		eventCount := 0
		for iter.Next() {
			i.rateLimiter.Wait()
			transfer := iter.Event

			if err := i.repo.SaveEvent(ctx, transfer); err != nil {
				log.Error().
					Err(err).
					Str("tx_hash", transfer.Raw.TxHash.Hex()).
					Msg("failed to save event")
				continue
			}
			eventCount++

			if eventCount%100 == 0 {
				log.Info().
					Int("event_count", eventCount).
					Uint64("block", transfer.Raw.BlockNumber).
					Msg("processing progress")
			}
		}

		if err := iter.Error(); err != nil {
			return fmt.Errorf("iterator error: %v", err)
		}

		log.Info().
			Int("event_count", eventCount).
			Uint64("from", from).
			Uint64("to", to).
			Msg("successfully processed block range")

		return nil
	}

	return fmt.Errorf("max retries exceeded for blocks %d-%d", from, to)
}

func (i *Indexer) watchNewBlocks(ctx context.Context, client *ethclient.Client, erc20 *abi.Erc20, startBlock uint64) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			currentBlock, err := client.BlockNumber(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to get current block")
				continue
			}

			if currentBlock <= startBlock {
				continue
			}

			opts := &bind.FilterOpts{
				Start: startBlock,
				End:   &currentBlock,
			}

			iter, err := erc20.FilterTransfer(opts, nil, nil)
			if err != nil {
				log.Error().Err(err).Msg("failed to filter transfers")
				continue
			}

			for iter.Next() {
				transfer := iter.Event
				if err := i.repo.SaveEvent(ctx, transfer); err != nil {
					log.Error().Err(err).Msg("failed to save event")
					continue
				}
			}

			startBlock = currentBlock
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
