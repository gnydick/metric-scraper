# Config

## Bad config at startup

- Dictated by the owner (Gabe), 2026-10-01: "a bad config shouldn't literally crash the service. it should be a failure from invalidated config"
- A bad config never panics the service.
- Loading validates the config.
- A config that fails validation makes startup fail.
- That failure is a reported validation failure, not a crash.
- Ticket: #17 (A bad or incomplete config crashes the scraper instead of being refused at startup).

## Glossary

- Bad config: a config file that is missing a field, has a field of the wrong type, or holds a value the service does not know. It is what validation rejects.
- Crash: a panic with a stack trace. It is the outcome this specification rules out.
- Validation failure: the error that loading returns for a bad config. It is the outcome this specification requires in place of a crash.
