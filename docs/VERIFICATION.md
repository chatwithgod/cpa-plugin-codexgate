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

商店收录申请：[PR #284](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store/pull/284)。该 PR 曾于 2026-10-24（Asia/Shanghai）按用户要求临时关闭，以调整 README、配图和双语简介；后续收录继续使用同一个 PR。最新审核状态请查看 PR。上述安装器验证不等于官方商店已经收录；合并后仍需确认官方目录可检索并完成安装。

## 本机图片工具鉴权热修复：0.1.1-dev.1

2026-10-24（Asia/Shanghai），排查内置生图工具返回的 `401 Invalid API key`：同一份当前 Codex 登录凭证访问 `/v1/models` 返回 200，但访问 `/v1/images/generations` 和 `/v1/images/edits` 返回 401。原有 CPA API Key 能通过图片接口鉴权；故意提交不完整 JSON 时，接口返回本地参数错误 400，没有调用上游模型。

根因是 CodexGate 的路由允许列表遗漏了这两个 Images API 入口。修复仅增加这两个路径的 POST 请求，不开放其他图片路径或方法，不改变凭证、有效期、账号检查和管理接口保护。

- Windows amd64 本地开发包 `0.1.1-dev.1`：单元测试、vet、race 和隔离 CPA 8.0.38 宿主检查通过。
- 新增隔离检查覆盖两个图片接口：当前本机凭证和原 API Key 均进入参数检查（400），无凭证仍返回 401。原有普通请求、WebSocket 握手、凭证轮换/退出和插件启停检查继续通过。
- 用户授权后，使用管理 API 热切换本机插件，保留 `0.1.0-dev.3` 原库及回退备份，没有重启 CPA/Codex。其他运行配置及受保护文件保持不变。
- 按用户授权仅执行 **1 次真实内置生图工具调用**，成功生成“白底蓝色圆形”测试图片。图像编辑接口验证的是鉴权，没有另做真实编辑调用。
- 新 DLL SHA-256：`a70fd652e6ae5754437bf46e1adc010e6588af5c7c0fc836a2cb76e94e697854`。

本地证据：`.runs/image-auth-0.1.1-dev.1-smoke.json`、`.runs/image-auth-install-report.json`、`.runs/image-auth-real-tool-check.json`。该开发包只做 Windows 本机验证，不单独发布到 GitHub Release 或插件商店；正式版平台验证另记，不改写上面的 v0.1.0 记录。

## 正式版 0.1.1：图片接口鉴权修复

2026-10-24（Asia/Shanghai），[v0.1.1](https://github.com/chatwithgod/cpa-plugin-codexgate/releases/tag/v0.1.1) 已公开发布，源代码提交为 `320636b37f95ae5a0426268622d6f3df183c0a0c`。[正式发布 CI](https://github.com/chatwithgod/cpa-plugin-codexgate/actions/runs/40026948901) 的五个平台及发布步骤全部通过：

| 平台 | 原生构建/单元/vet/race | 商店 SDK 安装器 | 官方 CPA 宿主（含图片鉴权与 WS 握手） |
| --- | --- | --- | --- |
| Windows amd64 | 通过 | 通过 | 通过 |
| Linux amd64 | 通过 | 通过 | 通过 |
| Linux arm64 | 通过 | 通过 | 通过 |
| macOS amd64 | 通过 | 通过 | 通过 |
| macOS arm64 | 通过 | 通过 | 通过 |

每个平台均新增验证 `POST /v1/images/generations` 和 `POST /v1/images/edits`：无凭证返回 401；当前登录和原 API Key 均通过鉴权，随后由故意不完整的 JSON 触发本地参数错误 400。这些检查没有调用图片模型。原有凭证轮换、退出、启停、并发回调、管理接口保护和 WebSocket 握手检查继续通过；两种 Linux 架构符合 GLIBC 2.17 基线。

发布后下载了全部 5 个公开 ZIP，核对 SHA-256、压缩包结构、内置构建记录和动态库校验值，均与 [verification.json](https://github.com/chatwithgod/cpa-plugin-codexgate/releases/download/v0.1.1/verification.json) 一致。另用固定版本的 CPA SDK 安装器从真实 GitHub Release 下载 Windows 包，安装到新的隔离目录，并通过隔离 CPA 8.0.38 的完整宿主检查。

本地报告：`.runs/published-v0.1.1/download-verification.json`、`.runs/published-v0.1.1/live-install.json`、`.runs/published-v0.1.1/live-installed-host-smoke.json`。正式 Windows DLL SHA-256 为 `e2c5d88b57da7f3f2c7d98024449b48e36e08dc10a9292bb86f89d5a8cc6c835`。本轮发布没有新增真实模型调用；本机生产插件保持此前已验证的 `0.1.1-dev.1`，隔离测试进程已停止。

原商店 [PR #284](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store/pull/284) 的标题及中英介绍已更新到 v0.1.1，仍只新增 `codexgate` 注册条目。该条目按最新 Release 获取版本，无需另加版本字段。记录时 PR 开放待审核，尚未正式上架。
