---
# q9y-85g
title: 'prime: omit todo guidance when .jig.yaml has no todo section'
status: completed
type: feature
priority: normal
created_at: 2026-07-29T04:36:54Z
updated_at: 2026-07-29T04:37:51Z
---

`jig prime` currently emits the full issue-tracking guide whenever any config file is found, because todoconfig.Load applies defaults and never signals whether the `todo:` section was actually present. Add a HasTodoSection detector and have prime output nothing when the config lacks a todo section (so projects that use jig only for cite/nope/brew/etc. don't get irrelevant todo instructions primed into agents).

## Summary of Changes

- Added `todoconfig.HasTodoSection(configPath)` which reports whether a config file actually contains a `todo:` section (legacy `.todo.yml` always counts; `.jig.yaml`/`.toba.yaml` require the top-level `todo:` key). `Load` still returns a defaulted config, so this is the only reliable presence signal.
- `jig prime` now returns early (no output) when a config file is found but has no todo section, so projects using jig only for cite/nope/brew/etc. don't prime agents with an irrelevant issue-tracking guide.
- Added `TestHasTodoSection` covering present/absent/empty-path/legacy cases.
