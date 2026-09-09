# DX65 Device Mint UI Implementation Plan

> **For agentic workers:** Inline execution (user: 确认后写计划并开工).

**Goal:** Auth Sessions panel mint form (show tokens once) + `make device-session-smoke`.

**Architecture:** Frontend API wrapper + SpacePage form; smoke = go TestAuthSessionDevice* + FE markers + evidence md (oidc-console style).

**Tech Stack:** React, vitest, bash Makefile.

## Global Constraints

- Do not call `setAuthSession` with device tokens
- No backend contract change
- No new tables

---

### Task 1: API + UI + tests
### Task 2: smoke script + make + docs
