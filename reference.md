# blueship reference

## Allowlisted upstream

- `openharmony/docs`
- `openharmony/interface_sdk-js`
- `openharmony/interface_sdk_c`
- `HarmonyOS_Samples/guide-snippets`

## Config (`.blueship.json`)

Created by `blueship init` at repo root.

## CLI binaries

Built by `cli/build.sh` into `cli/dist/`:

- `blueship-darwin-arm64`
- `blueship-darwin-amd64`
- `blueship-windows-amd64.exe`
- `blueship-linux-amd64`
- `blueship-linux-arm64`

## Token

`git config --get gitcode.token` or `GITCODE_TOKEN`.

## APIs

- `POST /api/v5/repos/{owner}/{repo}/issues` body `{"title","body":"-"}`
- `POST /api/v5/repos/{owner}/{repo}/pulls/{n}/comments` body `{"body":"start build"}`
