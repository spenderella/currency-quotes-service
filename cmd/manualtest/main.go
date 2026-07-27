package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/spenderella/currency-quotes-service/internal/config"
	"github.com/spenderella/currency-quotes-service/internal/db/postgres"
	"github.com/spenderella/currency-quotes-service/internal/repository"
)

func main() {
	cfg, err := config.New()
	must("load config", err)

	db, err := postgres.Connect(cfg.Postgres)
	must("connect db", err)
	defer db.Close()

	repo := repository.NewQuoteRepository(db)
	ctx := context.Background()

	step("1. CreateQuoteUpdate (EUR/MXN, key=manual-1)")
	id1, err := repo.CreateQuoteUpdate(ctx, "EUR", "MXN", "manual-1")
	must("create", err)
	fmt.Printf("  id=%s\n", id1)

	step("2. GetQuoteUpdateByID right after create (expect status=pending, zero Rate/FetchedAt)")
	q, err := repo.GetQuoteUpdateByID(ctx, id1)
	must("get by id", err)
	fmt.Printf("  %+v\n", q)

	step("3. GetPendingQuoteUpdates (expect id1 present)")
	pending, err := repo.GetPendingQuoteUpdates(ctx)
	must("get pending", err)
	fmt.Printf("  count=%d %+v\n", len(pending), pending)

	step("4. MarkInProgress(id1)")
	must("mark in progress", repo.MarkInProgress(ctx, id1))
	q, err = repo.GetQuoteUpdateByID(ctx, id1)
	must("get by id", err)
	fmt.Printf("  status=%s\n", q.Status)

	step("5. MarkDone(id1, 18.5, now)")
	rate := decimal.NewFromFloat(18.5)
	fetchedAt := time.Now().UTC()
	must("mark done", repo.MarkDone(ctx, id1, rate, fetchedAt))
	q, err = repo.GetQuoteUpdateByID(ctx, id1)
	must("get by id", err)
	fmt.Printf("  %+v\n", q)

	step("6. GetQuoteUpdateLatest(EUR, MXN) (expect same as step 5)")
	latest, err := repo.GetQuoteUpdateLatest(ctx, "EUR", "MXN")
	must("get latest", err)
	fmt.Printf("  %+v\n", latest)

	step("7. GetPendingQuoteUpdates again (expect id1 gone now)")
	pending, err = repo.GetPendingQuoteUpdates(ctx)
	must("get pending", err)
	fmt.Printf("  count=%d %+v\n", len(pending), pending)

	step("8. GetQuoteUpdateByID with random id (expect ErrNotFound)")
	_, err = repo.GetQuoteUpdateByID(ctx, uuid.New())
	fmt.Printf("  err=%v  is ErrNotFound=%v\n", err, errors.Is(err, repository.ErrNotFound))

	step("9. Second CreateQuoteUpdate (key=manual-2) then MarkFailed")
	id2, err := repo.CreateQuoteUpdate(ctx, "USD", "CAD", "manual-2")
	must("create 2", err)
	must("mark failed", repo.MarkFailed(ctx, id2, "provider timeout"))
	q, err = repo.GetQuoteUpdateByID(ctx, id2)
	must("get by id 2", err)
	fmt.Printf("  status=%s rate=%v fetchedAt=%v (note: error text not exposed by this method)\n", q.Status, q.Rate, q.FetchedAt)

	step("10. Idempotency check: CreateQuoteUpdate again with key=manual-1 (expect same id as step 1)")
	idAgain, err := repo.CreateQuoteUpdate(ctx, "EUR", "MXN", "manual-1")
	must("create again", err)
	fmt.Printf("  id=%s same_as_id1=%v\n", idAgain, idAgain == id1)

	fmt.Println("\nDONE")
}

func step(name string) {
	fmt.Println("\n--- " + name + " ---")
}

func must(op string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %v", op, err))
	}
}
