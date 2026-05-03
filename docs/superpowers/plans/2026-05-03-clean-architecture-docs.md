# Clean Architecture Docs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update `README.md` and `ARCHITECTURE.md` so experienced backend developers can understand how this Go REST API demonstrates clean architecture.

**Architecture:** Keep README as a concise orientation and make ARCHITECTURE.md the deeper technical walkthrough. Preserve the existing repo structure and documentation files; this is a documentation-only change.

**Tech Stack:** Markdown documentation for a Go REST API using chi, GORM, PostgreSQL, JWT, goose, and OpenAPI/oapi-codegen.

---

## File Structure

- Modify: `README.md` — clarify educational goal, summarize clean architecture layers, and direct readers to the architecture deep dive.
- Modify: `ARCHITECTURE.md` — expand the existing architecture explanation with dependency direction, boundary DTOs, request flow, domain aggregate behavior, repository responsibilities, testing strategy, and trade-offs.

---

### Task 1: Update README orientation

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Rewrite the opening description**

Replace the short product-focused description with one that says the project is an educational clean-architecture example for experienced backend developers.

- [ ] **Step 2: Add a clean architecture overview section**

Add a section after the Tech Stack that explains:
- handlers translate HTTP/OpenAPI concerns;
- services coordinate application flow;
- domain files own business rules;
- repositories own persistence;
- DTO/datamodel separation is intentional;
- `ARCHITECTURE.md` contains the deep dive.

- [ ] **Step 3: Preserve practical setup/API content**

Keep the existing setup, configuration, API, development, project structure, and CLI sections, adjusting wording only where it reinforces the educational framing.

---

### Task 2: Expand ARCHITECTURE.md deep dive

**Files:**
- Modify: `ARCHITECTURE.md`

- [ ] **Step 1: Reframe the document purpose**

Update the introduction to say the document is a clean-architecture walkthrough, not just a file map.

- [ ] **Step 2: Add clean architecture principles**

Add a section explaining the dependency rule used here: HTTP and database are outer details; services and domain model are inner application code; dependencies point inward through DTO boundaries.

- [ ] **Step 3: Deepen package/layer explanations**

Expand the existing package shape section with concrete responsibilities and what each layer must not do.

- [ ] **Step 4: Deepen request/data flow**

Keep the current numbered task request flow, but make the transformations between HTTP models, service DTOs, record DTOs, datamodels, and domain aggregates explicit.

- [ ] **Step 5: Deepen task aggregate explanation**

Explain why task activity generation belongs in `internal/task/domain.go` instead of `repository.go`, and how that keeps persistence focused on atomic storage.

- [ ] **Step 6: Add trade-offs and testing strategy**

Add sections explaining the cost of explicit mapping, why it is acceptable for an educational API, and how tests map to the layers.

---

### Task 3: Verify documentation quality

**Files:**
- Verify: `README.md`
- Verify: `ARCHITECTURE.md`

- [ ] **Step 1: Read both files end-to-end**

Check for duplicated sections, stale contradictions, and unclear wording.

- [ ] **Step 2: Run a lightweight markdown check**

Run: `grep -R "TBD\|TODO\|implement later" README.md ARCHITECTURE.md`
Expected: no output.

- [ ] **Step 3: Review git diff**

Run: `git diff -- README.md ARCHITECTURE.md`
Expected: only documentation changes to the two requested files.
