# SKILLS.md — Slash Commands, Recipes & Anti-Patterns

**For:** Common workflows, recipes, and things to avoid.

## Slash Commands

### /scribe
**Refresh scaffold documentation** (CLAUDE.md, AGENTS.md, SKILLS.md) based on latest codebase state.

Usage: `/scribe` after substantive changes.  
Output: Updated docs, proposed git commit.

### /code-review
**Review a changeset** for adherence to CLAUDE.md layout, test coverage, and lint compliance.

Usage: `/code-review [commit-sha]` or `/code-review branch:feat/my-feature`.  
Output: Checklist of findings, improvement suggestions.

## Recipes

### Adding a New Adapter (Complete Walkthrough)

1. **Create the adapter directory:**
   ```bash
   mkdir -p internal/manager/<adapter-id>
   ```

2. **Implement parse.go** with parsing logic:
   ```go
   package <adapter-id>

   // ParseList parses output of `<adapter> list` command
   func ParseList(output string) ([]Package, error) { ... }

   // ParseSearch parses output of `<adapter> search <query>` command
   func ParseSearch(output string) ([]Package, error) { ... }
   ```

3. **Implement cmd.go** with command builders:
   ```go
   // CommandList returns the Spec for listing installed packages
   func CommandList() *spec.Cmd { ... }

   // CommandSearch returns the Spec for searching packages
   func CommandSearch(query string) *spec.Cmd { ... }
   ```

4. **Create integration_test.go** with fixtures:
   - Capture real adapter output → `internal/spec/fixtures/<adapter-id>/`
   - Write golden tests that compare parser output against fixtures.

5. **Wire into cmd/pkgr/main.go:**
   ```go
   import "<adapter-id> internal/manager/<adapter-id>"
   
   // Add to manager registry in init()
   ```

6. **Run tests and linting:**
   ```bash
   make test
   make lint
   ```

7. **Commit:**
   ```bash
   git add internal/manager/<adapter-id>/ cmd/pkgr/main.go
   git commit -m "feat(adapter): add <adapter-id> support"
   ```

### Refreshing Golden Fixtures

When parser behavior changes (e.g., handling a new field):

1. **Run the real adapter:**
   ```bash
   brew list --json > internal/spec/fixtures/brew/list_output.json
   ```

2. **Update the test:**
   ```go
   // internal/manager/brew/parse_test.go
   func TestParseList(t *testing.T) {
       output := readFixture(t, "list_output.json")
       pkgs, err := ParseList(output)
       // Update assertions if behavior changed
   }
   ```

3. **Verify:** `make test` (all green).

4. **Commit:**
   ```bash
   git add internal/spec/fixtures/brew/list_output.json internal/manager/brew/parse_test.go
   git commit -m "chore(fixtures): refresh brew list output"
   ```

### Updating Version

1. **Edit internal/config/version.go:**
   ```go
   const Version = "0.2.0"  // e.g., 0.1.0 → 0.2.0
   ```

2. **Verify build:**
   ```bash
   make build
   ./pkgr version  # Should show "pkgr 0.2.0"
   ```

3. **Commit:**
   ```bash
   git add internal/config/version.go
   git commit -m "chore(release): bump version to 0.2.0"
   ```

## Anti-Patterns (Don't Do These)

### ❌ PM-Specific Adapter Libraries

**Bad:** Importing `gopkg.in/yaml.v2` inside `internal/manager/brew/` for custom YAML parsing.

**Why:** Couples adapter to external library; breaks modularity; hard to update library version without affecting other adapters.

**Good:** Define a shared parser interface in `internal/parser/`, implement per-adapter, keep external deps minimal and centralized.

### ❌ Parsing Without Golden Fixtures

**Bad:** Writing a parser and assuming it handles all real-world output formats.

**Why:** Parser behavior degrades over time as adapters change their output format. No regression safety net.

**Good:** Capture real output → golden fixture → test against fixture. Fixture becomes the source of truth.

### ❌ Caching Mutating Operations

**Bad:** Caching the result of `npm install` or `brew upgrade`.

**Why:** Cache stale; next run uses outdated data; user expects fresh results from install operations.

**Good:** Cache read-only ops (list, search, info). Never cache write/mutate ops. Document in code: `// NOT cached: install/upgrade/remove`.
