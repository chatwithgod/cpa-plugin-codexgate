# CodexGate

[English](README.md) · [下载版本](https://github.com/chatwithgod/cpa-plugin-codexgate/releases) · [验证记录](docs/VERIFICATION.md)

> **发布状态**：v0.1.0 已在 [GitHub Releases](https://github.com/chatwithgod/cpa-plugin-codexgate/releases/tag/v0.1.0) 正式发布。官方商店收录申请 [PR #284](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store/pull/284) 待审核，尚未上架，目前可按下方说明手动安装。五个平台均已通过原生 CI 和发布包核对。

CodexGate 是 CPA 的入口鉴权兼容插件。它让 Codex 保留原生 `openai` 提供商标识，同时模型请求继续走本地 CPA。在我们验证过的桌面环境中，这样能解决自定义提供商导致 **dot 入口不显示**的问题，但不会给账号开通原本没有的功能权限。

原有 CPA API Key 继续有效。插件不改请求和回复，不接管路由、Reflow 或上游账号，也不增加独立后台进程。这是独立的 MIT 开源项目，不是 OpenAI 或 CPA 官方插件。

## 怎么工作？

每次收到支持的请求，插件读取当前登录文件。只有 Bearer Token 与文件内的 `access_token` **精确匹配，且没有过期**才放行；请求带了 `ChatGPT-Account-ID` 时，还会检查账号是否匹配。

找文件的顺序是：

1. 插件设置里指定的 `auth-file` 绝对路径。
2. CPA 进程的 `CODEX_HOME/auth.json`。
3. CPA 运行用户目录下的 `~/.codex/auth.json`。

CPA 和 Codex 同机、同用户运行，且使用文件式 ChatGPT 登录时，通常不需要填写插件配置。登录刷新后，新 Token 从下一次请求起生效，旧 Token 失效；退出、过期或文件损坏时拒绝放行，原 API Key 不受影响。

目前只支持 `auth_mode=chatgpt` 的文件登录，不读取系统密钥链。服务账号和容器不自动拥有桌面用户的登录文件：容器可以只读挂载包含 `auth.json` 的目录，再填写容器内的绝对路径。不要只挂载单个文件，否则客户端原子替换文件后可能读不到更新；也不要把登录目录公开。

## 安装与使用顺序

商店收录后可以直接安装；收录前可从 [Releases](https://github.com/chatwithgod/cpa-plugin-codexgate/releases) 下载对应平台的 ZIP。

1. 确认 Codex 已通过 ChatGPT 登录并有本地登录文件；CPA 监听 `127.0.0.1`，保留至少一个原 API Key。
2. 把包内的 `codexgate.dll`、`codexgate.so` 或 `codexgate.dylib` 放到 CPA 插件目录的 `<goos>/<goarch>` 下。macOS 对应 `darwin`，Apple Silicon 对应 `arm64`。不要覆盖正在加载的库；手动替换时，先在空闲时间停止 CPA。
3. 在管理面板启用 CodexGate。若手动编辑 YAML，将下面配置**合并**进原配置，不要覆盖其他插件：

   ```yaml
   plugins:
     enabled: true
     configs:
       codexgate:
         enabled: true
         # auth-file: /absolute/path/to/auth.json
   ```

4. 使用 CPA 管理密钥查询 `GET /v0/management/codexgate/status`，确认 `enabled: true`、`login.ready: true`。状态接口不返回 Token 或账号 ID。
5. **ECPA 用户**打开“使用 openai 提供商标识”。**原版 CPA 用户**先备份 Codex 配置，再修改顶层字段：

   ```toml
   model_provider = "openai"
   openai_base_url = "http://127.0.0.1:8317/v1"
   ```

   地址以自己的 CPA 为准，不要放进 `[model_providers.openai]` 下，也不用改 `auth.json`。
6. 等 Codex 任务结束后重启，测试 dot 和新聊天，并确认 CPA 使用记录中能看到请求。

## 支持范围与回退

支持模型列表、Responses HTTP/SSE、WebSocket 握手、compact、alpha/search 及对应的 `/backend-api/codex` 路由。Chat Completions 等无关入口仍走 CPA 原有鉴权；管理接口仍需要管理密钥。

插件不验证 JWT 的密码学签名，不联网刷新登录，也不查询云端撤销状态；信任来自本机可信文件的精确匹配。已建立的 WebSocket 不会被主动断开，鉴权不是逐消息进行的。

回退时，**先**关闭 ECPA 的上述选项，或恢复原 CPA 提供商配置；空闲时重启 Codex，**再**禁用或卸载 CodexGate。不要只卸载插件却留下依赖它的登录鉴权配置。原 API Key 留着即可，不要把登录 Token 复制进静态密钥列表。

## 平台与验证

已通过原生宿主验证的平台为 Windows amd64、Linux amd64/arm64（GLIBC 2.17+）、macOS amd64/arm64。兼容基线为 CPA **8.0.38**、ABI 1/schema 6，Go **1.26.8**。Linux 的 `no-plugin` 版 CPA 和纯 musl 环境不适用。

2026-10-23，Windows 开发版 `0.1.0-dev.3` 已通过隔离鉴权测试，用户也确认重启后 dot、新聊天正常。正式版的多平台自动验证另见[验证记录](docs/VERIFICATION.md)、Actions 和 Release 中的 `verification.json`。

构建、发布流程见[开发说明](docs/DEVELOPMENT.md)，来源与许可见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
