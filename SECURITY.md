# Security Policy

## Reporting a Vulnerability

Please report suspected vulnerabilities privately through GitHub Security Advisories:

- https://github.com/shikarii/SpanPay/security/advisories/new

Avoid posting exploit details publicly until a fix is available.

## Scope

Security-sensitive areas include:

- payment state transitions and idempotency handling
- webhook signature verification and inbox processing
- provider credentials and secret management
- ledger posting rules and reconciliation workflows
- dependency updates affecting HTTP, database, or crypto behavior

## Response targets

- Initial triage: within 7 days
- Mitigation plan: as soon as impact is confirmed
- Public disclosure: after remediation is available
