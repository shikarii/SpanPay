package orchestra

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	ErrMissingKey = errors.New("Idempotency-Key header is required")
	ErrConflict   = errors.New("idempotency key in use with different request body")
	ErrInProgress = errors.New("request with this idempotency key is already in progress")
	ErrKeyExpired = errors.New("idempotency key has expired")
)

const DefaultKeyTTL = 24 * time.Hour

// DBTX abstracts *sql.DB and *sql.Tx.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// CachedResponse holds the stored response for a completed idempotent request.
type CachedResponse struct {
	StatusCode int
	Body       []byte
}

// IdempotencyResult describes what the caller should do after checking a key.
type IdempotencyResult struct {
	// IsNew is true when this is the first request for this key.
	IsNew bool
	// Cached is non-nil when a completed response exists for this key.
	Cached *CachedResponse
}

// RequestHash computes the SHA-256 hex digest of a request body.
func RequestHash(body []byte) string {
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

// Check looks up an idempotency key and determines the appropriate action.
//
// Returns:
//   - IsNew=true: no prior request; caller should call Begin then process.
//   - Cached non-nil: completed request; caller returns the cached response.
//   - ErrConflict: key exists but request hash differs (422).
//   - ErrInProgress: another request is processing with this key (409).
func Check(ctx context.Context, db DBTX, merchantID, key, hash string) (*IdempotencyResult, error) {
	if key == "" {
		return nil, ErrMissingKey
	}

	var status, storedHash string
	var responseCode sql.NullInt32
	var responseBody []byte
	var expiresAt sql.NullTime

	err := db.QueryRowContext(ctx,
		`SELECT status, request_hash, response_code, response_body, expires_at
		 FROM idempotency_keys
		 WHERE merchant_id = $1 AND idempotency_key = $2`,
		merchantID, key,
	).Scan(&status, &storedHash, &responseCode, &responseBody, &expiresAt)

	if errors.Is(err, sql.ErrNoRows) {
		return &IdempotencyResult{IsNew: true}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("idempotency check: %w", err)
	}

	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		return nil, ErrKeyExpired
	}

	if storedHash != hash {
		return nil, ErrConflict
	}

	if status == "IN_PROGRESS" {
		return nil, ErrInProgress
	}

	return &IdempotencyResult{
		Cached: &CachedResponse{
			StatusCode: int(responseCode.Int32),
			Body:       responseBody,
		},
	}, nil
}

// Begin inserts the key as IN_PROGRESS. Call before processing the request.
func Begin(ctx context.Context, db DBTX, merchantID, key, hash string, ttl time.Duration) error {
	if ttl == 0 {
		ttl = DefaultKeyTTL
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO idempotency_keys (merchant_id, idempotency_key, request_hash, status, expires_at)
		 VALUES ($1, $2, $3, 'IN_PROGRESS', $4)`,
		merchantID, key, hash, time.Now().Add(ttl),
	)
	if err != nil {
		return fmt.Errorf("idempotency begin: %w", err)
	}
	return nil
}

// Complete marks the key as COMPLETED with the response to replay on duplicates.
func Complete(ctx context.Context, db DBTX, merchantID, key string, statusCode int, body []byte) error {
	_, err := db.ExecContext(ctx,
		`UPDATE idempotency_keys
		 SET status = 'COMPLETED', response_code = $3, response_body = $4
		 WHERE merchant_id = $1 AND idempotency_key = $2`,
		merchantID, key, statusCode, body,
	)
	if err != nil {
		return fmt.Errorf("idempotency complete: %w", err)
	}
	return nil
}

// Cleanup deletes expired idempotency keys. Intended for periodic background runs.
func Cleanup(ctx context.Context, db DBTX) (int64, error) {
	result, err := db.ExecContext(ctx,
		`DELETE FROM idempotency_keys WHERE expires_at < $1`,
		time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("idempotency cleanup: %w", err)
	}
	return result.RowsAffected()
}
