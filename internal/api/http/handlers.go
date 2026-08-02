package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/spenderella/currency-quotes-service/internal/domain"
	"github.com/spenderella/currency-quotes-service/internal/service"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// createQuoteHandler accepts a currency pair, persists it as a pending
// update and returns its ID immediately — the actual fetch from the
// provider happens in the background worker.
func (s *Server) createQuoteHandler(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req CreateQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	base, quote, err := normalizeCurrencyPair(req.BaseCurrency, req.QuoteCurrency)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := s.quoteService.CreateQuoteUpdate(r.Context(), base, quote, idempotencyKey)
	if err != nil {
		s.handleServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusAccepted, CreateQuoteResponse{ID: id})
}

func (s *Server) getQuoteByIDHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id: "+err.Error())
		return
	}

	quote, err := s.quoteService.GetQuoteUpdateByID(r.Context(), id)
	if err != nil {
		s.handleServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, newQuoteResponse(quote))
}

func (s *Server) getLatestQuoteHandler(w http.ResponseWriter, r *http.Request) {
	base, quote, err := normalizeCurrencyPair(r.URL.Query().Get("base"), r.URL.Query().Get("quote"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	latest, err := s.quoteService.GetQuoteUpdateLatest(r.Context(), base, quote)
	if err != nil {
		s.handleServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, newQuoteResponse(latest))
}

func newQuoteResponse(q domain.Quote) QuoteResponse {
	return QuoteResponse{
		ID:            q.ID,
		BaseCurrency:  q.BaseCurrency,
		QuoteCurrency: q.QuoteCurrency,
		Status:        q.Status,
		Rate:          q.Rate,
		ProviderTime:  q.ProviderTime,
		FetchedAt:     q.FetchedAt,
	}
}

// handleServiceError maps known service errors to HTTP status codes and logs
// anything unexpected as a 500.
func (s *Server) handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrUnsupportedCurrency):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		s.logger.ErrorContext(r.Context(), "unhandled service error", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func normalizeCurrencyCode(code string) (string, error) {
	if len(code) != 3 {
		return "", fmt.Errorf("invalid currency code %q: expected a 3-letter ISO code, e.g. EUR", code)
	}
	return strings.ToUpper(code), nil
}

func normalizeCurrencyPair(baseRaw, quoteRaw string) (base, quote string, err error) {
	var errs []error

	base, baseErr := normalizeCurrencyCode(baseRaw)
	if baseErr != nil {
		errs = append(errs, baseErr)
	}
	quote, quoteErr := normalizeCurrencyCode(quoteRaw)
	if quoteErr != nil {
		errs = append(errs, quoteErr)
	}

	if err = errors.Join(errs...); err != nil {
		return "", "", err
	}
	return base, quote, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
