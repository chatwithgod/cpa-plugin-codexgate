# 验证记录

首次 Windows 开发版验证：2026-10-23（Asia/Shanghai）。后续平台验证见下方 CI 记录。

## 已通过

开发版 `0.1.0-dev.3`，Windows amd64，Go 1.26.8，CPA 8.0.38（提交 `0db19025`）。

- `go test ./...`：通过。
- `go vet ./...`：通过。
- `go test -race ./internal/...`：通过。
- 原生 DLL 被 CPA 注册，插件有效启用，使用非独占入口鉴权。
- 不配置 `auth-file`，自动读取隔离 `CODEX_HOME/auth.json`。
- 没有凭证、其他 JWT、过期 token、已退出的登录、损坏的登录文件均返回 401。
- 原有 CPA API Key 仍返回 200，包括退出 Codex 和禁用插件之后。
- 当前本机登录 token 返回 200；替换登录文件后，新 token 生效，旧 token 立即返回 401。
- Responses WebSocket：缺少凭证时 401，有效凭证时握手 101，然后立即关闭，没有发送模型请求。
- 管理状态接口仍需要管理密钥，不能用 Codex token 访问。
- 通过 CPA 管理 API 禁用插件后，Codex token 被拒绝；重新启用后恢复。
- 测试日志不包含合成 token、测试 API Key 或测试管理密钥。
- 生产 Codex、ECPA、CPA 配置和真实登录文件的校验值未变化；隔离进程已停止。

完整隔离报告：`.runs/20261023-220825-bf6d42/summary.json`。

DLL：`build/0.1.0-dev.3/codexgate-v0.1.0-dev.3.dll`。

SHA-256：`29f756e03f2f41cdd3bc5c2de723962047f684713e6cc011918a39ab2f506df3`。

## 测试中修正的接入问题

- `dev.1`：CPA 要求源码仓库字段非空；已在后续版本补齐拟建地址，未创建或发布仓库。
- `dev.2`：管理注册传入的是 `/v0/management`，不是包含插件名的路径；已在 `dev.3` 修正。
- 隔离脚本曾原子替换整个配置文件，但未观察到本机 CPA 执行热重载。启停验证已改用 ECPA 同类的官方管理操作 `PATCH /v0/management/plugins/codexgate/enabled`。没有为此修改 CPA 核心。

旧版本 DLL 未覆盖。

## Windows 真实安装与客户端验收（开发版）

2026-10-23，用户授权安装 `0.1.0-dev.3` 到其 CPA 8.0.38，保留原 API Key 和其他插件。安装后验证原 API Key 与本机 Codex 登录凭证均可读取模型列表，无凭证仍返回 401。

随后用户开启 ECPA“使用 openai 提供商标识”并重启 Codex，明确反馈：**dot 通信成功，新聊天通信成功**。这是开发版的人工客户端验收，不是对所有平台、全部模型或所有传输路径的验收。

此前的隔离测试只用合成凭证和空上游账号目录，没有模型调用。正式版自动测试保持同样的隔离方式，不更新或重启上述生产实例。

## 正式版 0.1.0 平台验证

| 平台 | 构建/单元/vet/race | CPA 商店安装器 | 官方 CPA 8.0.38 宿主鉴权与 WS 握手 | 人工 Codex 客户端 |
| --- | --- | --- | --- | --- |
| Windows amd64 | 通过 | 通过 | 通过 | 开发版通过；正式版未另测 |
| Linux amd64 | 通过 | 通过 | 通过 | 未测 |
| Linux arm64 | 通过 | 通过 | 通过 | 未测 |
| macOS amd64 | 通过 | 通过 | 通过 | 未测 |
| macOS arm64 | 通过 | 通过 | 通过 | 未测 |

首次五端通过的代码提交：`77df3aa37415ef5b2bacd8579e1d6a2ae509fabc`；[完整 CI 运行及平台报告](https://github.com/chatwithgod/cpa-plugin-codexgate/actions/runs/39999645261)。五端均通过并发原生回调、原 API Key、当前登录、过期/轮换/退出、启停、管理接口保护和 WebSocket 握手检查。两种 Linux 架构均检查了 GLIBC 2.17 基线。

Intel macOS 首次实测暴露了宿主与 Go 动态库的 TLS/回调栈冲突，表现为 `runtime.cgocallback` / `unknown caller pc`。已增加仅用于该平台的 C 线程回调边界，保留相同鉴权逻辑；修复后官方 CPA 宿主和并发回调通过。细节见 [DEVELOPMENT.md](DEVELOPMENT.md)。

正式发布工作流要求五个平台的自动验证全部通过。正式 Release 的 `verification.json` 记录具体提交、工具链、动态库校验值、安装器与宿主检查结果。系统凭据库不在支持范围内，其他 CPA 版本未据此自动视为兼容。

鉴权发生在新的 HTTP 请求和 WebSocket 握手。已经建立的 WebSocket 连接不会由本插件主动断开；不能把握手测试理解成消息逐条重新鉴权。

## 正式发布后的下载与安装核对

[v0.1.0](https://github.com/chatwithgod/cpa-plugin-codexgate/releases/tag/v0.1.0) 已公开发布。发布提交为 `4c7a4e3ebf04cd876eed1ade22ee98c11ff423fa`；[正式发布 CI](https://github.com/chatwithgod/cpa-plugin-codexgate/actions/runs/40000618432) 再次通过上述五个平台的构建、单测、vet、race、商店 SDK 安装器及官方 CPA 宿主测试。

发布后另做了以下核对：

1. 下载全部 5 个公开 ZIP，核对 `checksums.txt`、ZIP 结构和动态库校验值，确认它们与 [verification.json](https://github.com/chatwithgod/cpa-plugin-codexgate/releases/download/v0.1.0/verification.json) 记录的受测产物一致。
2. 使用 CPA SDK 安装器从真实 GitHub Release 下载并安装 Windows 动态库，安装到全新的隔离目录，没有覆盖现有库。
3. 用这份安装后的动态库启动隔离 CPA 8.0.38，重新通过加载、并发回调、鉴权、WebSocket 握手、凭证轮换/退出、启停和管理接口保护检查。

这些自动测试没有调用真实模型，受保护的生产配置及登录文件校验值未变，隔离进程已停止。本机生产环境仍保留开发版 `0.1.0-dev.3`；正式版没有另做人工 Codex 客户端验收。

商店收录申请：[PR #284](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store/pull/284)，当前待审核。上述安装器验证不等于官方商店已经收录；合并后还需确认官方目录可检索并完成安装。
