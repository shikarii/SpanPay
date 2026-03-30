# Contributing

Thanks for improving SpanPay.

## Workflow

1. Branch from the latest `develop`.
2. Keep scope tight and commit coherent units.
3. Run validation locally before pushing.
4. Open a PR to `develop` with a linked issue URL, risk notes, and validation evidence.
5. Merge only after required checks pass.

## Local validation

```bash
make typecheck
make test
make build
```

## Engineering rules

- Follow `AGENTS.md`.
- Keep behavior changes covered by tests.
- Avoid architectural changes without ADRs where required.
- Treat ledger correctness, idempotency, and reconciliation as first-class concerns.
