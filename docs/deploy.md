# Deploy

```sh
GOWORK=off go run .        # listens on NP_HTTP_ADDR, default :8534
go test ./...
```

Quadlet: `deploy/openpak-np-update.container`; image `ghcr.io/openpak/np-update`. No secrets, no database, no TLS:
[`np-account`](https://github.com/openpak/np-account) terminates firmware TLS and forwards `/tpl/` here when
`NP_UPDATE_URL=http://openpak-np-update:8534` is set.
