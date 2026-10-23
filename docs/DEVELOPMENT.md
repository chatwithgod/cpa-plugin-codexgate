# Build and release

## Pinned inputs

`scripts/toolchains.json` pins Go 1.26.8, CPA 8.0.38 archives and SHA-256 values, LLVM-MinGW 20260922, Xcode 16.4, and manylinux2014 image digests. CI action revisions are pinned in `.github/workflows/release.yml`.

Python 3.11+ and Git are required for scripts. No pip packages are needed. The Go SDK remains pinned in `go.mod`; plugin-store imports are verification tooling, not a second plugin service.

## Native checks

```sh
python scripts/ci.py
```

Windows needs the pinned LLVM-MinGW compiler (or pass `--cc` to its `x86_64-w64-mingw32-gcc.exe`); macOS CI selects Xcode 16.4 and `CC=clang`. `--go` and `--core` accept explicit local paths. CPA is only launched with an isolated temporary config, synthetic login, empty upstream-account directory and random loopback port.

Linux release jobs run natively on amd64 and arm64 inside the pinned manylinux2014 images, including the real official CPA host smoke. `readelf` rejects libraries requiring GLIBC newer than 2.17. A modern Linux compiler may produce a newer GLIBC requirement; use the workflow container for a portable release.

Individual steps:

```sh
python scripts/build.py
python scripts/package_release.py
python scripts/smoke_cpa.py --core /path/to/cli-proxy-api --library /path/to/codexgate.so --expected-version 0.1.0
```

`build.py` checks the Go version and native architecture, runs tests/vet/race, then writes a version/platform-specific library. Existing outputs are never overwritten. `build.ps1` and `package.py` are retained development helpers; store releases use the Python pipeline.

## What the release gate proves

1. Native tests, vet and race checks pass.
2. ZIP layout and checksums follow the store specification.
3. The pinned CPA SDK's actual release installer downloads local in-memory fixtures, rejects a mismatched checksum, parses the ZIP, and installs the library to a new temporary directory. This check cannot reach external endpoints.
4. The official CPA 8.0.38 binary loads that installed library and exercises authentication, rotation/logout/expiry, enable/disable and WebSocket handshake with synthetic credentials.
5. Only redacted summaries and packages are uploaded as artifacts. Raw test configs, login files, and host logs stay out of releases.

This proves loading and authentication, not a full Codex UI or upstream-model session on every OS. Manual acceptance is recorded separately.

## Publish

1. Update `VERSION`, changelog, README status, and the verification record.
2. Commit a clean tree; let all five main-branch jobs pass.
3. Push a matching `v<version>` tag. The tag workflow rebuilds and verifies all targets.
4. `assemble_release.py` requires matching version/source commit, clean sources, successful host/installer evidence, valid archives/checksums, and both Linux GLIBC reports.
5. Upload all assets to a draft release before publishing: `codexgate_<version>_<goos>_<goarch>.zip`, `checksums.txt`, and `verification.json`.
6. Validate published downloads, then submit `store/registry-entry.json` to the official store in a PR changing only `registry.json`. A pending PR is not a listing.

Never replace a published artifact silently; use a new version. Do not restart or modify a user's running CPA/Codex during CI or publishing.
