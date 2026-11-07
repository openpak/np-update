# PS3 Title Update Server

OpenPak's PS3 game update check (title patch list) service.

| | |
| --- | --- |
| Network | PlayStation 3 |
| Role | Reverse-proxies the game patch list (`/tpl/...-ver.xml`) and patch packages (`/tppkg/...`) to Sony unchanged |
| Depends on | nothing |
| Used by | [`np-account`](https://github.com/openpak/np-account) forwards `/tpl/` here (`NP_UPDATE_URL`) |
| Image | `ghcr.io/openpak/np-update` |
| Licence | AGPL-3.0-only |

OpenPak does not host or cache patches; system firmware updates are never offered.

## Run

```sh
GOWORK=off go run .
```

## Docs

- [Protocol](docs/protocol.md)
- [Deploy](docs/deploy.md)

## Licence

AGPL-3.0-only. See [LICENSE](LICENSE).
