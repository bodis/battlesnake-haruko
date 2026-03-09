# Next Iteration

Execute the next planned iteration from the roadmap. Follow these phases strictly in order. Stop and wait for user confirmation between phases.

## Phase 1: Identify & Plan

1. Read `ROADMAP.md` and find the first iteration with **Status: PLANNED**
2. Read `ENGINE.md` for current architecture context
3. Read all files listed in that iteration's "Files to Modify" table
4. Present a summary to the user:
   - Which iteration you found
   - What it does (1-2 sentences)
   - Implementation approach
   - Any concerns or questions about the plan
5. **STOP and wait for user review/approval before proceeding to Phase 2**

## Phase 2: Implement

1. Implement the iteration following the plan in ROADMAP.md
2. Ensure zero-alloc on hot path (use `sync.Pool`, stack arrays)
3. Run `go build ./...` and `go test ./...` — fix any failures
4. Run benchmarks: `go test -bench='BenchmarkEvaluate$|BenchmarkEvaluateLateGame|BenchmarkBRSNode' -benchmem -count=3 ./logic/`
5. Report benchmark results vs baseline (from MEMORY.md)
6. **STOP and wait for user review before proceeding to Phase 3**

## Phase 3: Test

1. Run A/B tests as specified in the iteration's Testing Plan section
2. Typically: `make compare PREV=snapshots/haruko-69c43bb N=100` (vs v32) and any other versions specified
3. Report results clearly:
   - Win rate vs each version
   - Whether it meets the success criteria (usually >55%)
4. If results are below target: discuss with user whether to tune weights, try alternatives, or declare dead end
5. **STOP and wait for user decision before proceeding to Phase 4**

## Phase 4: Document & Finalize

Only proceed here if user confirms the iteration is successful (or declares it a dead end).

### If successful:
1. Run `make snapshot` to create a snapshot of the new version
2. Move the completed iteration section from `ROADMAP.md` to `ROADMAP_FINISHED.md` (append at the end, before the snapshot log if applicable)
3. Update `ROADMAP.md`:
   - Update "Current State" table with new version info
   - Update snapshot log with new entry
4. Update `ENGINE.md`:
   - Add entry to version history table
   - Update eval signals section if signals changed
   - Update any architecture descriptions that changed
5. Update `CLAUDE.md`:
   - Update "Current state" section
6. Update memory file (`MEMORY.md`) with key findings
7. Commit all changes with descriptive message
8. Push to remote

### If dead end:
1. Revert any eval/search changes (keep infrastructure if useful)
2. Update the iteration status in `ROADMAP.md` to "DEAD END" with what was tried and why it failed
3. Update `ENGINE.md` dead ends section
4. Update `CLAUDE.md` current state
5. Update memory file with lessons learned
6. Commit and push
