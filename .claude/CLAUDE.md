# Rules

- Always run both unit tests (`make test`) and e2e tests (`make e2e`) after making changes
- When TUI rendering changes (keybar, layout, screen flow), update e2e snapshots with `make e2e-update` before running `make e2e`
