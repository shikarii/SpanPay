# Data Model Specification

PostgreSQL is the authoritative system of record.
The initial migration in `db/migrations/000001_initial.sql` establishes the baseline.
This document is the reference for column semantics, constraints, and indexing strategy.

## Core Transactional Tables

### payments

The primary intent and high-level business entity.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PRIMARY KEY | Unique internal ID |
| merchant_id | UUID | NOT NULL, INDEX | Reference to the business |
| external_id | TEXT | NOT NULL, INDEX | Merchant's reference (e.g. Order ID) |
| amount | NUMERIC(19,4) | NOT NULL, CHECK > 0 | Total amount in base units |
| currency | CHAR(3) | NOT NULL | ISO 4217 code |
| status | TEXT | NOT NULL | CREATED, AUTHORIZED, CAPTURED, etc. |
| metadata | JSONB | NULLABLE | Extensible business data |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW | Audit timestamp |

### payment_attempts

Payments may have multiple attempts across different providers if retries are enabled.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PRIMARY KEY | Unique attempt ID |
| payment_id | UUID | NOT NULL, FK(payments) ON DELETE RESTRICT | Parent payment |
| provider_id | TEXT | NOT NULL | e.g. 'stripe', 'paypal' |
| provider_ref | TEXT | UNIQUE, INDEX | ID provided by the PSP |
| attempt_type | TEXT | NOT NULL | AUTH, CAPTURE, REFUND |
| status | TEXT | NOT NULL | SUCCESS, FAILED, PENDING |
| error_code | TEXT | NULLABLE | Normalized error code (see provider-interface.md) |
| raw_response | JSONB | NULLABLE | Exact response from the provider |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW | Audit timestamp |

Future constraint: unique partial index on (payment_id, attempt_type) WHERE status = 'SUCCESS'
to ensure only one successful capture or refund per payment.

## Ledger Tables

Three tables enforce a strict double-entry system.

### ledger_accounts

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PRIMARY KEY | Unique account ID |
| name | TEXT | NOT NULL | e.g. 'Settlement_Assets', 'Merchant_Wallet:abc' |
| account_type | TEXT | NOT NULL | ASSET, LIABILITY, EQUITY, EXPENSE, REVENUE |
| normal_balance | TEXT | NOT NULL | DEBIT or CREDIT |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW | Audit timestamp |

### ledger_transactions

Groups a set of balancing entries into one atomic unit.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PRIMARY KEY | Unique transaction ID |
| payment_id | UUID | FK(payments) ON DELETE RESTRICT | Related payment |
| reference | TEXT | NOT NULL | Human-readable description |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW | Audit timestamp |

### ledger_entries

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PRIMARY KEY | Unique entry ID |
| transaction_id | UUID | NOT NULL, FK(ledger_transactions) | Groups the balancing entries |
| account_id | UUID | NOT NULL, FK(ledger_accounts) | The targeted account |
| debit | NUMERIC(19,4) | NOT NULL, DEFAULT 0, CHECK >= 0 | Debit amount |
| credit | NUMERIC(19,4) | NOT NULL, DEFAULT 0, CHECK >= 0 | Credit amount |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW | Audit timestamp |
| CONSTRAINT | ledger_entries_single_side | CHECK | Either debit or credit > 0, not both |

## Reliability and Persistence Tables

### idempotency_keys

| Column | Type | Constraints | Description |
|---|---|---|---|
| idempotency_key | TEXT | PRIMARY KEY | Key from the request header |
| request_hash | TEXT | NOT NULL | SHA-256 hash of request payload |
| response_code | INT | NOT NULL | Cached HTTP status |
| response_body | JSONB | NOT NULL | Cached response body |
| expires_at | TIMESTAMPTZ | INDEX | Expiration time (24h recommended) |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW | Audit timestamp |

### webhook_inbox

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PRIMARY KEY | Internal ID |
| provider | TEXT | NOT NULL | e.g. 'stripe' |
| provider_evt_id | TEXT | UNIQUE | Provider's event ID for deduplication |
| raw_payload | JSONB | NOT NULL | Raw body as received |
| status | TEXT | NOT NULL | PENDING, PROCESSED, FAILED |
| received_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW | Ingestion timestamp |

### outbox_events

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PRIMARY KEY | Internal ID |
| topic | TEXT | NOT NULL | Event type |
| payload | JSONB | NOT NULL | Event data |
| status | TEXT | NOT NULL | PENDING, PUBLISHED, FAILED |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW | Creation timestamp |

### reconciliation_runs

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PRIMARY KEY | Internal ID |
| provider | TEXT | NOT NULL | Provider being reconciled |
| status | TEXT | NOT NULL | RUNNING, COMPLETED, FAILED |
| started_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW | Start timestamp |
| completed_at | TIMESTAMPTZ | NULLABLE | Completion timestamp |

## Indexing Strategy

- B-Tree indexes on all foreign keys and created_at columns.
- Unique partial indexes on payment_attempts(payment_id, attempt_type) WHERE status = 'SUCCESS'.
- Check constraints enforce non-negative amounts on all financial columns.
- Foreign keys use ON DELETE RESTRICT to prevent deletion of referenced records.
