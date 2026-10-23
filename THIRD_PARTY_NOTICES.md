# Third-party notices

CodexGate uses the MIT-licensed CLIProxyAPI plugin SDK, pinned to v8.0.38:

- https://github.com/router-for-me/CLIProxyAPI/tree/v8.0.38/sdk/pluginapi
- https://github.com/router-for-me/CLIProxyAPI/tree/v8.0.38/sdk/pluginabi

The native C ABI types and original shell in `cmd/codexgate/` are adapted from its Go native plugin example:

- https://github.com/router-for-me/CLIProxyAPI/blob/v8.0.38/examples/plugin/frontend-auth-exclusive/go/main.go

Unlike that example, CodexGate explicitly uses **non-exclusive** authentication and keeps CPA's normal API-key authentication in place. The token validation logic is implemented in this repository.

The upstream license is retained in `licenses/CLIProxyAPI-MIT.txt`. YAML configuration parsing uses `gopkg.in/yaml.v3` v3.0.1 under its MIT/Apache-2.0 licensing; see `licenses/go-yaml-LICENSE` and `licenses/go-yaml-NOTICE`.

The compiled libraries include the Go runtime and standard library; their BSD license is retained in `licenses/Go-LICENSE`. Windows releases are built with LLVM-MinGW 20260922; its compiler/runtime notices are retained in `licenses/LLVM-MinGW-LICENSE.txt`.

The release verification helper uses CPA's unmodified plugin-store installer from the same pinned SDK. This helper is build/test tooling and is not included as a second runtime plugin or service.
