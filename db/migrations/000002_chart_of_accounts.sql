-- Chart of accounts: core system-level ledger accounts.
-- Per-merchant accounts (e.g. Merchant_Wallet_Pending:merchant_abc)
-- will be created dynamically during merchant onboarding.

INSERT INTO ledger_accounts (id, name, account_type, normal_balance) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'Settlement_Pending',        'ASSET',    'DEBIT'),
    ('a0000000-0000-0000-0000-000000000002', 'Settlement_Assets',         'ASSET',    'DEBIT'),
    ('a0000000-0000-0000-0000-000000000003', 'Merchant_Wallet_Pending',   'LIABILITY','CREDIT'),
    ('a0000000-0000-0000-0000-000000000004', 'Merchant_Wallet_Available', 'LIABILITY','CREDIT'),
    ('a0000000-0000-0000-0000-000000000005', 'Merchant_Bank_Account',     'ASSET',    'DEBIT'),
    ('a0000000-0000-0000-0000-000000000006', 'Fee_Expense',               'EXPENSE',  'DEBIT'),
    ('a0000000-0000-0000-0000-000000000007', 'Inbound_Clearing',          'ASSET',    'DEBIT'),
    ('a0000000-0000-0000-0000-000000000008', 'Currency_Exchange_Loss',    'EXPENSE',  'DEBIT'),
    ('a0000000-0000-0000-0000-000000000009', 'Rounding_Expense',          'EXPENSE',  'DEBIT');
