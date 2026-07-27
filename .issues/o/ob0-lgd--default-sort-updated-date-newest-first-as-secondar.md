---
# ob0-lgd
title: 'Default sort: updated date newest-first as secondary (after status)'
status: completed
type: feature
priority: normal
created_at: 2026-07-27T23:30:51Z
updated_at: 2026-07-27T23:48:33Z
---

Change the default issue sort (CompareByStatusPriorityAndType) so ordering is status → updated-date (newest first) → priority → type → title. Currently updated date is not part of the default ordering.

## Summary of Changes

Inserted updated date (newest first) as the secondary key in the default sort (`CompareByStatusPriorityAndType`). New order: status → updated↓ → priority → type → title.

- Added `compareUpdatedDesc` helper (nil updated dates sort last, equal dates fall through to lower keys).
- Applies everywhere the default sort is used (CLI list/archive, TUI list/detail).
- Added subtests pinning the new secondary ordering, tie-fallthrough to priority, and nil-date handling.

Refined: secondary key is `updated_at ?? created_at` (activity date) — use updated_at, fall back to created_at when unset; newest first. Issues with neither date sort last.
