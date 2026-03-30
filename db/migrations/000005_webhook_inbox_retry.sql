-- Add retry tracking to webhook_inbox for poison event handling.

ALTER TABLE webhook_inbox
    ADD COLUMN retry_count INT NOT NULL DEFAULT 0;
