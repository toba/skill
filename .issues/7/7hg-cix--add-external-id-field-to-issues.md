---
# 7hg-cix
title: Add external_id field to issues
status: completed
type: feature
priority: normal
created_at: 2026-07-26T03:17:02Z
updated_at: 2026-07-26T03:23:02Z
sync:
    github:
        issue_number: "127"
        synced_at: "2026-07-26T03:26:36Z"
---

Allow issues to store an external ID in frontmatter (e.g. reference to an ID in another tracking system). Wire through the issue model (frontmatter parse/render), GraphQL schema + resolvers (create/update), CLI flags, and display.

## Summary of Changes

Added an optional `external_id` frontmatter field to issues for referencing an issue in another system (e.g. a Jira key or legacy ticket number).

- `internal/todo/issue/issue.go`: `ExternalID` field + frontmatter parse/render (round-trip covered by tests)
- `internal/todo/graph/schema.graphqls` + regenerated code: `externalId` on `Issue`, `CreateIssueInput`, `UpdateIssueInput`
- `internal/todo/graph/schema.resolvers.go`: set on create/update; empty string clears it on update
- `cmd/todo_create.go` / `cmd/todo_update.go`: `--external-id` flag
- `cmd/todo_show.go`: `ext:` in the show header; `internal/todo/tui/detail.go`: `ext` in the detail dates line
- `cmd/todo_prompt.tmpl`: documented the flag for agents
- Tests: model round-trip + resolver create/update/clear
