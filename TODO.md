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
- [ ] Implement sitemap fetch (`topics*.xml.gz`)
- [ ] Implement gzip + XML parsing to topic slugs (preserve order)
- [ ] Implement case-insensitive search
- [ ] Implement CLI `zenn-topics <query>`
- [ ] Add fixtures/tests (parser/search/CLI behavior)
- [ ] Run verification (`mise run check`)
