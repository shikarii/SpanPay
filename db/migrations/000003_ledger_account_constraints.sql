-- Tighten ledger_accounts columns from free-text to checked enums.

ALTER TABLE ledger_accounts
    ADD CONSTRAINT ledger_accounts_type_check
    CHECK (account_type IN ('ASSET', 'LIABILITY', 'EQUITY', 'EXPENSE', 'REVENUE'));

ALTER TABLE ledger_accounts
    ADD CONSTRAINT ledger_accounts_normal_balance_check
    CHECK (normal_balance IN ('DEBIT', 'CREDIT'));
