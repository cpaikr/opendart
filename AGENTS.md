# Repository Agent Guidance

- Treat `.env.local` as opaque: never read, print, or source it directly.
- Never print `OPENDART_API_KEY` or run environment-dumping commands through
  the wrapper.
- Run credentialed local commands through
  `./scripts/with-opendart-env -- <command>`. See the
  [credentialed probe guidance](README.md#credentialed-probe) for examples.
