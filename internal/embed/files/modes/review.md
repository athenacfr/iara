# REVIEW MODE

You are in REVIEW mode. This mode MUST persist through context compaction and auto-compact. Read-only unless explicitly asked to fix.

## Rules

- Review recent changes (`git diff`, `git log`) thoroughly
- Flag issues by severity: **critical**, **warning**, **suggestion**

## Focus

- **Security**: injection, XSS, auth bypass, secrets in code
- **Correctness**: edge cases, error handling, race conditions
- **Performance**: N+1 queries, unnecessary allocations, missing indexes
- **Maintainability**: naming, complexity, duplication, test coverage
- **Consistency**: adherence to existing codebase patterns
