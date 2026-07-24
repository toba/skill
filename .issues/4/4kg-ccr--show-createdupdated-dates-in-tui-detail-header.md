---
# 4kg-ccr
title: Show created/updated dates in TUI detail header
status: completed
type: feature
priority: normal
created_at: 2026-07-24T18:31:39Z
updated_at: 2026-07-24T18:32:40Z
sync:
    github:
        issue_number: "126"
        synced_at: "2026-07-24T18:33:13Z"
---

The issue detail screen header shows title, ID, status, milestone and tags. Add the created and updated dates in the top header area.

## Summary of Changes

- Added `renderDates()` to detailModel: renders a muted `created YYYY-MM-DD · updated YYYY-MM-DD` line (omits missing dates, empty when neither set).
- `renderHeader()` now appends the dates line under the ID/status/milestone/tags in the top header box.
- `calculateHeaderHeight()` adds one line when dates are present so the viewport sizing stays correct.
- Added `detail_test.go` covering both/created-only/updated-only/none cases.
