-- Add merchant_id scoping and processing status to idempotency_keys.
-- The initial migration created the table without these fields.

ALTER TABLE idempotency_keys DROP CONSTRAINT idempotency_keys_pkey;

ALTER TABLE idempotency_keys
    ADD COLUMN merchant_id UUID NOT NULL,
    ADD COLUMN status TEXT NOT NULL DEFAULT 'COMPLETED';

ALTER TABLE idempotency_keys
    ADD PRIMARY KEY (merchant_id, idempotency_key);

ALTER TABLE idempotency_keys
    ALTER COLUMN response_code DROP NOT NULL,
    ALTER COLUMN response_body DROP NOT NULL;

ALTER TABLE idempotency_keys
    ADD CONSTRAINT idempotency_keys_status_check
    CHECK (status IN ('IN_PROGRESS', 'COMPLETED'));
