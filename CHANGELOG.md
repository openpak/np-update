# Changelog: np-update

## 0.2.0 - 2026-11-07

- Game patches pass through to Sony (owner decision 2026-11-07, openpak/prds#158): `/tpl/` goes over HTTPS to the request's `*.dl.playstation.net` host with SCEI DNAS Root 05 pinned (Sony's leaf is SHA-1, checked by hand), `/tppkg/` streams over HTTP with Range support, nothing cached. Logs title and offered versions. Sony unreachable or a foreign host answers 404 (no update). System firmware update hosts never reach this service.

## 0.1.0

- First release: every title patch list request answers "no update available".
