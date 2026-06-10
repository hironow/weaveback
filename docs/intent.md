# Intent

**Last updated:** 2026-06-10
**Requester:** hironow
**Status:** DRAFT — AI が README / git 履歴から起草。requester 未確認
**Work unit:** weaveback — Go client library and CLI for the Weave Feedback API

## Goal

Provide a Go client library (`pkg/weave/`) for the Weave (wandb.ai) Feedback
API — with authentication, retry, and helper functions on top of
oapi-codegen-generated code — and a CLI (`cmd/weaveback/`) for feedback
operations (create, query, replace, purge) with stdin pipe support.

## Success Criteria

- `just build`, `just test`, and `just lint` pass (justfile recipes; unit, integration, and e2e test suites exist under `tests/`)
- CLI feedback subcommands work in both argument mode (#6) and stdin pipe mode (#11), backed by E2E tests against the built binary (#13, fixed in #18)
- Contract tests pass against the cherry-picked OpenAPI spec with Basic auth (#12, #14)
- Beyond these: 未定義 — Open Questions 参照

## Scope

### In scope

- Client wrapper with auth, retry, and error handling (`pkg/weave/`, MY-395) plus helpers like AddReaction/AddNote (MY-396)
- Code generation from the cherry-picked Weave OpenAPI feedback endpoints (`api/`, `pkg/weave/gen/`, MY-393/MY-394)
- CLI feedback subcommands with `--weave-ref` required flag on create/replace (MY-397, MY-398, MY-414)

### Out of scope (Non-goals)

- 未確認 — the repo covers only the Feedback API endpoints (the OpenAPI spec was cherry-picked to feedback endpoints in MY-393); whether other Weave API surfaces are intended later is not stated

## Constraints

- Auth is HTTP Basic (not Bearer) — corrected in MY-399/MY-415 and reflected in the OpenAPI securitySchemes
- Client construction uses constructor injection instead of implicit `os.Getenv` (MY-416)
- Integration tests require `WANDB_API_KEY` (`just test-integration`)
- Generated code under `pkg/weave/gen/` is produced via `just generate` (oapi-codegen) — do not hand-edit

## Open Questions

- [ ] requester による本ドラフトのレビュー
- [ ] `just fetch-spec` is a stub ("not yet configured (see MY-393)") — how should the OpenAPI spec be refreshed from upstream?
- [ ] Product-level success criteria and intended consumers of the library/CLI — not stated in README
- [ ] Deadlines or milestone targets — none found in the repo
- [ ] Whether non-feedback Weave API endpoints are ever in scope
