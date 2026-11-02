# Protocol

Firmware 4.93. VSH builds the title patch list URL in `FUN_00281734` (vsh.elf 0x281734):

`https://a0.ww.np.dl.playstation.net/tpl/np/<titleid>/<titleid>-ver.xml`

A `200` reply is parsed as
`<titlepatch titleid=".." status="alive"><tag name=".." popup=".." signoff=".."><package version=".." size=".." sha1sum=".." url=".." ps3_system_ver=".."/></tag></titlepatch>`
(parser 0x2826xx-0x2832b8). A body of 0 bytes, or 64 KiB or more (`0x80028eda`), is not a list.

"No update" replies:

| Reply | VSH behaviour |
| --- | --- |
| `404` or `403` | `if (status - 0x193 < 2)` branches to 0x281f18: no data, success, no error shown |
| `200` with `Content-Length: 0` | `if (size == 0)` branches to the same 0x281f18 |
| any other non-200 status | error `0x80028ed9` |
| `200`, not parsable | error `0x80028ed5`/`0x80028ed6` |

np-update answers `404` with an empty body to every `GET /tpl/...`, the status Sony serves for titles without a
patch. OpenPak never hosts Sony or game patch packages, so no list is ever returned.
