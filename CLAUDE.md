# fintechdemo

Claude Code loads this file at the start of every session. It records what we are building, the decisions already made, and where we are. Update the "Progress" section whenever a step is finished.

## What we are building

A production-grade Nigerian digital bank and P2P payments platform, written in Go as microservices, designed for millions of users. It is also a teaching project: David is learning how a senior fintech backend engineer designs and builds such a system, and must be able to defend every decision in an interview.

Product scope:

- Onboarding: registration, email OTP (no SMS), BVN/NIN KYC, customer tiers, devices, PIN.
- Authentication: password, PIN, passkey/biometric (public keys only), access and refresh tokens, sessions, lockout.
- Accounts on a double-entry ledger: available, ledger, reserved and pending balances; statements.
- Same-bank P2P transfers (atomic, idempotent).
- Interbank transfers over a NIBSS NIP simulator that we build ourselves (decided 2026-10-11), with timeout handling, status queries, reversal and reconciliation. A Paystack adapter may be added later behind the same interface.
- Charges: transfer fees, VAT on fees, statutory levies (see "Charges").
- Savings (flexible, goals, autosave, disciplined, non-interest option), loans, airtime, data, TV, betting funding.
- Kafka events through a transactional outbox, background workers, notifications, fraud and risk, observability.

Priority order when goals conflict: Security → Financial correctness → Reliability → Regulatory compliance → Observability → User experience → Performance → Scale.

## How to work with David

- Teach step by step. David types the code himself; do not write project source files for him. Use tools to inspect, run and check his work.
- Each step is a full vertical slice (about a phase's worth: schema, domain, use case, adapters, tests). Smaller steps were rejected as too short several times.
- Every new service is built from scratch, with every file shown in full. Never tell him to `cp` files from another service or "copy and change these lines".
- For each step explain: what, why, alternatives, security, scalability, failure scenarios, database design, transaction boundaries, concurrency, duplicate prevention; then the files, tests, run commands and what was learned.
- No test code in the steps for now (David's instruction, 2026-10-11). Do not include `_test.go` files or test commands until he asks for tests again. Claude still verifies each step privately before presenting it.
- No mistakes that cause confusion: every command, path and file must be run exactly as written in a scratch copy first. One clear instruction per action; no alternatives, no "edit the copy" shortcuts.
- Build and run every step in a scratch copy before presenting it, and say what was and was not verified.
- Check his work at the start of each step by actually running it. He has said "next" before a step was working.
- Never invent regulatory values. Look up the current CBN / NIBSS / tax authority source first; values live in Policy tables, never in code.
- Challenge bad architecture the way a principal engineer would.

## Architecture decisions (settled)

- REST only at the edge (API Gateway). gRPC for all synchronous service-to-service calls. Kafka only for asynchronous facts, never on the money command path.
- Database per service on PostgreSQL, enforced by roles: `<svc>_owner` runs migrations, `<svc>_app` runs the service with SELECT and INSERT only unless a migration grants more.
- Account, Wallet and Ledger are ONE service and one database, so a P2P transfer is a single ACID transaction. Postings are append-only and are the source of truth; balances are projections updated in the same transaction. Corrections are reversal journals.
- Concurrency on balances: pessimistic row locks (`SELECT ... FOR UPDATE`) taken in account-ID order.
- Transfers are orchestrated by the Transfer service (explicit state machine), with idempotency keys under a unique constraint in Postgres.
- Interbank: hold funds, send to the rail (the NIP simulator), then capture on definite success or release on definite failure. A timeout means PENDING and a status query, never FAILED. The rail sits behind a `PaymentRail` interface so the provider can change without touching Transfer.
- Transactional outbox for every event; consumers are idempotent.
- Redis only for losable data: OTPs, rate limits, caches.
- Clean Architecture inside each service: `adapters → app → domain`; `cmd/` is wiring only.

Services, in build order: Ledger (skeleton), Identity, KYC, Policy (fees and limits), Transfer, Rail Gateway, NIP simulator, Notification, Reconciliation, API Gateway; later Savings, Loans, Bills, Fraud/Risk, Reporting.

## Charges

Not built yet. Designed as follows, and to be built with Policy and Transfer (Phases 9 to 10):

- Policy owns effective-dated tables: `transfer_fee_rules` (by transfer type, channel and amount band), `tax_rules` (VAT on fees), `levy_rules` (statutory transfer levy or stamp duty), `customer_tier_limits`.
- Transfer calls `Policy.QuoteCharges` before the customer authorises. The quote is stored on the transfer, so the customer is charged exactly what they were shown even if rules change mid-flight.
- The Ledger posts charges as separate lines in the same journal as the transfer:
  - debit customer: principal + fee + VAT + levy
  - credit recipient (same-bank) or the NIP settlement account (interbank): principal
  - credit fee income: fee
  - credit VAT payable: VAT
  - credit levy payable: levy
- A reversal returns principal and charges according to the reversal policy, as a new journal.
- Values to verify from official sources before loading any numbers: the customer transfer fee bands (CBN Guide to Charges), VAT rate on fees, the electronic transfer levy / stamp duty rule and who bears it, tier transaction and balance limits.

## External providers (David's choices, 2026-10-11)

Every provider sits behind an interface in the service that uses it, with an adapter per provider. Use test/sandbox keys only. Keys live in environment variables (a git-ignored `.env`), never in the repo, the Makefile or chat. Check each provider's current official API docs when its phase is built; do not write endpoints, fields or prices from memory.

| Need | Provider | Used by | Phase |
|---|---|---|---|
| Interbank transfers, name enquiry, status query | Our own NIBSS NIP simulator service (see below); Paystack optional later | Rail Gateway | 11 to 12 |
| BVN / NIN lookup, bank account verification, open banking, direct debit | Mono | KYC | 6 |
| Email (OTP) | Gmail account over SMTP with an app password | Identity OTP | 5 |
| Customer alerts (transfer received, debit, and so on) | Push through FCM, not email (Claude's reading of "we will use push notification for mail"; confirm with David) | Notification | 14 |
| Airtime, data, TV, bills | Flutterwave Bills Payment API (test key) | Bills | 17 |
| SMS | Not used at all (David's decision, 2026-10-11). Registration codes go by Gmail; anything that would otherwise be an SMS goes by push | n/a | n/a |

### NIBSS NIP simulator (decided 2026-10-11)

David proposed mocking NIP ourselves instead of using Paystack as the rail, and Claude agreed: a simulator lets us force every failure on demand (timeouts, unknown status, late success, duplicate callbacks), which a provider sandbox cannot.

- It is its own small service (`services/nipsim`) with its own database, reached over the network like a real third party. The Rail Gateway talks to it only through the `PaymentRail` interface.
- It implements the three NIP operations: Name Enquiry, Funds Transfer (single credit) and Transaction Status Query, keyed by a NIP-style session ID that the sender generates.
- It behaves like a real rail: realistic latency with jitter, configurable failure rates, definite failures with NIP-style response codes, ambiguous "processing" answers, requests that are processed but whose response is dropped (the timeout case), duplicate-session rejection, and scenario triggers (for example by account number or amount) so each case can be reproduced.
- Message shapes follow the publicly known NIP structure as closely as Claude can model it. The official NIBSS specification is only given to licensed institutions, so exact conformance is not claimed. If David obtains the spec, the simulator is updated to match it.
- Test keys and simulators are fine for learning. Holding real customer funds in production needs the right CBN licence or a licensed partner bank.

## Performance intent

- Same-bank transfer: synchronous, one database transaction in the Ledger, a handful of gRPC hops. Notifications and events are asynchronous and never delay the response.
- Interbank transfer: our side responds quickly with PROCESSING after the hold and dispatch; completion time is set by the rail and the receiving bank. The app is told the final state by push. Never report success before the rail confirms it.
- No latency has been measured yet. Budgets and load tests are Phase 20.

## Repository conventions

- Go workspace (`go.work`) with one module per directory: `pkg`, `gen`, `services/<name>`. There must be no `go.mod` in the repo root.
- Module paths: `github.com/sholana1/fintechdemo/<name>` for services (for example `.../ledger`, `.../identity`), `.../pkg`, `.../gen`. Services reference `pkg` and `gen` with `replace` directives.
- `proto/<service>/v1/*.proto` → `make proto` (buf) → `gen/`. Generated code is committed.
- `pkg/` holds infrastructure only (`grpcserver`, `pgdb`). Configuration stays per service.
- Service layout: `cmd/server`, `cmd/migrate`, `internal/{config,domain,app}`, `internal/adapters/{grpcapi,postgres,...}`, `migrations/` (goose SQL, embedded).
- Local ports: Ledger gRPC 50051, Identity gRPC 50052, Postgres 5432, Redis 6379, Kafka 9092.
- Make targets: `vet`, `test`, `test-integration`, `proto`, `infra-up`, `infra-down`, `infra-reset`, `migrate`, `run-<service>`.

## Progress

Done:

- Phase 1: architecture.
- Phase 2: monorepo, Ledger skeleton (config, gRPC health, graceful shutdown).
- Phase 3: Docker Compose (Postgres, Redis, Kafka), Ledger database roles, pool, migrations.

- Phase 4 part 1 (verified working 2026-10-11): shared `pkg/`, buf and `identity.proto`, Identity `RegisterUser`. Migrations apply for both services and a live `RegisterUser` call stores a user. Tests are deferred; the only test in the repo is `services/ledger/internal/config/config_test.go`. Note: the server constructor in `pkg/grpcserver` is named `NewServer` (not `New`).

- Phase 4 part 2 (verified working in the real repo 2026-10-11, uncommitted at the time; David tests gRPC with Postman by importing `proto/identity/v1/identity.proto`): Login, RefreshSession, Logout. Ed25519 JWT access tokens (15m), opaque refresh tokens stored as SHA-256 with single-use rotation and reuse detection (reuse revokes the session), `sessions` / `refresh_tokens` / `login_attempts` tables, Redis login lockout (Lua script) that fails closed, Redis revoked-session list for the gateway, `pkg/pgdb.DB` (`WithinTx`, `Conn`) as the shared transaction helper, Argon2id concurrency cap. Verified end to end in a scratch copy.

Next:
- Phase 5: onboarding. Email OTP by Gmail SMTP is the only registration check; a user becomes ACTIVE once the email is verified. The phone number is stored but not verified by code (no SMS); it is cross-checked against the BVN record in Phase 6. Devices: the first device is trusted by the email OTP; later devices are approved by a push to an already-trusted device once FCM exists (Phase 14), with email OTP as the fallback and as the only method until then. Transaction PIN with its own retry lock and a `VerifyPIN` call.
- Phase 6: KYC (Mono) and tiers. Phase 7 to 8: accounts and the double-entry ledger. Phase 9: same-bank P2P. Phase 10: fees and limits (Policy). Phase 11 to 12: NIP simulator service, interbank, timeout, reversal, reconciliation. Phase 13: outbox and Kafka. Phase 14: notifications (FCM push). Phase 15: savings. Phase 16: loans. Phase 17: airtime, data, TV, bills (Flutterwave). Phase 18: fraud and risk. Phase 19: observability. Phase 20: performance. Phase 21: production deployment.
