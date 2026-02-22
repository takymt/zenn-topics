# TODO (Phase1 MVP)

## Goal

Implement `zenn-topics <query>` as a minimal Go CLI that fetches Zenn topic sitemap files (`topics*.xml.gz`), parses them, and prints matching topic slugs.

## Confirmed MVP Decisions

- Search target: topic slug (`https://zenn.dev/topics/<slug>`)
- Match rule: case-insensitive substring match
- Output format: one slug per line
- Result order: preserve source order (no sorting)

## Definition of Done

- `zenn-topics <query>` works end-to-end with live Zenn sitemap data
- Invalid/missing query returns a clear error (non-zero exit)
- Network/parse failures return a clear error (non-zero exit)
- Core public behavior is covered by pure, state-based tests
- `mise run check` passes

## Checklist

- [x] Confirm MVP behavior and DoD with user
- [x] Implement sitemap fetch (`topics*.xml.gz`)
- [x] Implement gzip + XML parsing to topic slugs (preserve order)
- [x] Implement case-insensitive search
- [x] Implement CLI `zenn-topics <query>`
- [x] Add fixtures/tests (parser/search/CLI behavior)
- [x] Run verification (`mise run check`)

## Phase2: Local Cache (TTL)

### Goal

Add a minimal local cache for parsed topic slugs so repeated searches do not fetch Zenn sitemaps every time.

### Scope

- Save parsed slugs to a JSON cache file on disk.
- Cache payload format includes `fetched_at` and `slugs`.
- Use a TTL (target range: 1 hour to 1 day; decide default during implementation).
- If cache is fresh, search from cache instead of fetching sitemaps.
- Add `--refresh` to force bypassing cache and refreshing it.

### Definition of Done

- CLI uses cache when a fresh cache file exists.
- CLI refreshes cache when cache is missing or expired.
- CLI supports `--refresh` to force a cache refresh.
- Cache read/write failures return a clear error (non-zero exit).
- Core cache behavior is covered by pure, state-based tests.
- `mise run check` passes.

### Checklist

- [x] Decide cache file path and default TTL
- [x] Implement cache JSON schema (`fetched_at`, `slugs`)
- [x] Implement cache read/write + TTL validation
- [x] Integrate cache into topic fetch flow
- [x] Add `--refresh` option
- [x] Add tests for cache hit/miss/expired behavior
- [x] Run verification (`mise run check`)

## Phase3: CLI Meta Flags

### Goal

Add basic CLI meta flags for help, version, and debug visibility.

### Scope

- Add `--help` / `-h`
- Add `--version` / `-V`
- Add `--verbose` / `-v`
- Print verbose logs to stderr so stdout remains script-friendly

### Definition of Done

- `--help` prints usage and exits successfully.
- `--version` prints the CLI version and exits successfully.
- `--verbose` prints diagnostic logs to stderr during normal execution.
- Core flag behavior is covered by tests.
- `mise run check` passes.

### Checklist

- [x] Implement `--help` / `-h`
- [x] Implement `--version` / `-V`
- [x] Implement `--verbose` / `-v`
- [x] Add tests for help/version/verbose behavior
- [x] Run verification (`mise run check`)
