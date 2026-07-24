---
# fir-05f
title: Update all Go dependencies
status: completed
type: task
priority: normal
created_at: 2026-07-24T16:44:43Z
updated_at: 2026-07-24T16:48:07Z
sync:
    github:
        issue_number: "125"
        synced_at: "2026-07-24T17:02:23Z"
---

Update module deps to latest and verify build/test/vet.

## Summary of Changes
Ran `go get -u ./...` + `go mod tidy`, then resolved two version-skew breakages the blanket upgrade introduced:

- **bleve stack**: the upgrade bumped `RoaringBitmap/roaring/v2`, `bleve_index_api`, `scorch_segment_api`, `zapx/v17`, and `go-faiss` past what `bleve/v2 v2.6.0` (latest) supports, breaking compilation (`.Value` arity change). Pinned those five back to the versions bleve v2.6.0 declares (roaring v2.14.5, bleve_index_api v1.3.11, scorch_segment_api v2.4.7, zapx/v17 v17.1.2, go-faiss v1.1.0).
- **gqlgen**: upgraded v0.17.90 -> v0.17.94; regenerated `generated.go` via `go generate ./...` (old generated code referenced removed `CollectedField.Deferrable`). Re-applied gocritic `paramTypeCombine` to two regenerated resolver signatures to keep lint clean.

Verified: `go build`, `go vet ./...`, `go test ./...` (all pass), `scripts/lint.sh` (0 issues).
