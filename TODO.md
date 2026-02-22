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

## Next Task: Local Cache (TTL)

### Goal

Add a minimal local cache for parsed topic slugs so repeated searches do not fetch Zenn sitemaps every time.

### Draft Scope

- Save parsed slugs to a JSON cache file on disk.
- Cache payload format includes `fetched_at` and `slugs`.
- Use a TTL (target range: 1 hour to 1 day; decide default during implementation).
- If cache is fresh, search from cache instead of fetching sitemaps.
- `--refresh` support is planned for a later step (not required in the first cache implementation).

### Definition of Done (Draft)

- CLI uses cache when a fresh cache file exists.
- CLI refreshes cache when cache is missing or expired.
- Cache read/write failures return a clear error (non-zero exit).
- Core cache behavior is covered by pure, state-based tests.
- `mise run check` passes.

### Checklist (Draft)

- [ ] Decide cache file path and default TTL
- [ ] Implement cache JSON schema (`fetched_at`, `slugs`)
- [ ] Implement cache read/write + TTL validation
- [ ] Integrate cache into topic fetch flow
- [ ] Add tests for cache hit/miss/expired behavior
- [ ] Run verification (`mise run check`)
- [ ] Add `--refresh` option (later)
