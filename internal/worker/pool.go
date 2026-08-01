package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/spenderella/currency-quotes-service/internal/domain"
)

// IQuoteService is the subset of QuoteService the pool depends on.
type IQuoteService interface {
	ClaimPendingQuoteUpdates(ctx context.Context, limit int) ([]domain.Quote, error)
	RecoverPendingUpdates(ctx context.Context) ([]domain.Quote, error)
	ProcessClaimedTask(ctx context.Context, task domain.Quote) error
}

// Pool runs a ticker that claims pending quote updates and dispatches them to
// a fixed-size pool of worker goroutines. The claim batch size equals the
// pool size: never claim more than can be started on immediately.
type Pool struct {
	quoteService IQuoteService
	logger       *slog.Logger
	size         int
	tickInterval time.Duration
	taskTimeout  time.Duration
}

func New(quoteService IQuoteService, logger *slog.Logger, size int, tickInterval, taskTimeout time.Duration) *Pool {
	return &Pool{
		quoteService: quoteService,
		logger:       logger,
		size:         size,
		tickInterval: tickInterval,
		taskTimeout:  taskTimeout,
	}
}

// Run recovers any tasks left in_progress by a crash, then starts the ticker
// and worker goroutines. It blocks until ctx is cancelled, then stops
// claiming new work, lets in-flight tasks finish, and returns.
func (p *Pool) Run(ctx context.Context) error {
	tasks := make(chan domain.Quote, p.size)

	var wg sync.WaitGroup
	for range p.size {
		wg.Go(func() {
			for task := range tasks {
				p.process(task)
			}
		})
	}

	recovered, err := p.quoteService.RecoverPendingUpdates(ctx)
	if err != nil {
		if ctx.Err() == nil {
			p.logger.ErrorContext(ctx, "worker: recover pending updates", "error", err)
		}
	} else {
		p.dispatch(ctx, tasks, recovered)
	}

	ticker := time.NewTicker(p.tickInterval)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			// Keep claiming without waiting for the next tick.
			for {
				claimed, err := p.quoteService.ClaimPendingQuoteUpdates(ctx, p.size)
				if err != nil {
					if ctx.Err() == nil {
						p.logger.ErrorContext(ctx, "worker: claim pending updates", "error", err)
					}
					break
				}
				p.dispatch(ctx, tasks, claimed)
				if len(claimed) < p.size {
					break
				}
				if ctx.Err() != nil {
					break loop
				}
			}
		}
	}

	close(tasks)
	wg.Wait()
	return nil
}

// dispatch pushes claimed tasks onto the channel, giving up if ctx is
// cancelled while waiting for a free worker.
func (p *Pool) dispatch(ctx context.Context, tasks chan<- domain.Quote, batch []domain.Quote) {
	for _, task := range batch {
		select {
		case tasks <- task:
		case <-ctx.Done():
			return
		}
	}
}

// process resolves a single claimed task using its own bounded context, so
// cancelling Run's ctx during shutdown doesn't abort work already in flight.
func (p *Pool) process(task domain.Quote) {
	ctx, cancel := context.WithTimeout(context.Background(), p.taskTimeout)
	defer cancel()

	if err := p.quoteService.ProcessClaimedTask(ctx, task); err != nil {
		p.logger.ErrorContext(ctx, "worker: process claimed task", "id", task.ID, "error", err)
	}
}
