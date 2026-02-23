---
name: zenn-suggest-topics
description: This skill should be used when the user asks to "suggest topics", "add topics", "update topics", "fill in topics", or wants to set the frontmatter topics field for a Zenn article markdown file.
argument-hint: <path/to/article.md>
allowed-tools: Read, Edit, Bash(zenn-topics:*)
---

You are updating the `topics:` field in a Zenn article's frontmatter with verified topic slugs.

## Step 1: Read the article

Read the file at `$ARGUMENTS` to get the full content (frontmatter + body).

## Step 2: Analyze content

From the title, headings, and body text, identify 5–10 candidate technology/concept keywords to query. Generate varied queries: specific names first, then broader concepts. Examples: `go`, `cli`, `sitemap`, `xml`, `typescript`, `react`, `docker`.

## Step 3: Verify via zenn-topics

For each candidate keyword, run:

```bash
zenn-topics <keyword>
```

Output is one slug per line, or `No topics matched query: <keyword>` if none match. Collect all unique slugs returned across all queries.

## Step 4: Select topics

From the collected slugs, pick the 1–5 that best match the article's subject matter. Prefer specificity over breadth. **Never pick a topic slug that was not returned by `zenn-topics`.**

## Step 5: Update frontmatter

Use `Edit` to rewrite the `topics:` line in the frontmatter:

- If `topics:` already exists: replace its value in place.
- If `topics:` is absent: insert it after the last existing frontmatter key (before the closing `---`).
- Format: `topics: [slug1, slug2]` (inline YAML array, no trailing comma).

**Do not modify any other part of the file.**
