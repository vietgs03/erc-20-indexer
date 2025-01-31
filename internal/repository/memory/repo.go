package memory

import (
	"context"
	"sync"

	"github.com/vietgs03/erc-20-indexer/internal/abi"
)

type Repository struct {
	mu     *sync.Mutex
	events []*abi.Erc20Transfer
}

func NewRepository() *Repository {
	return &Repository{
		events: make([]*abi.Erc20Transfer, 0),
		mu:     &sync.Mutex{},
	}
}

func (r *Repository) SaveEvent(ctx context.Context, event *abi.Erc20Transfer) error {
	r.events = append(r.events, event)
	return nil
}
