# CodexGate

[English](README.md) · [插件商店](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store) · [手动下载](https://github.com/chatwithgod/cpa-plugin-codexgate/releases) · [开发文档](docs/DEVELOPMENT.md)

**解决接入 CPA 后，dot 入口消失的问题。**

CodexGate 是一个 CPA（CLIProxyAPI）插件，用来解决因自定义提供商标识导致的 **dot 入口不显示**问题，配合 ECPA 的 **“使用 openai 提供商标识”**功能使用。

只打开这个开关，可能会出现 dot 入口恢复了、请求却失败的情况：Codex 使用的是本机 ChatGPT 登录，而 CPA 原本接受的是 API Key。CodexGate 补上这部分支持，让 CPA 能识别本机当前有效的 Codex 登录。

模型请求仍然走 CPA，原有 API Key 继续可用。**不需要复制登录 Token，不需要配置模型列表，也不需要额外运行一个程序。**

## 使用前提

- CPA 和 Codex 在同一台电脑、同一个系统用户下运行。
- Codex 已通过 ChatGPT 账号登录，并在本地保存了 `auth.json` 登录文件。

插件会自动读取登录文件，通常不需要手写配置。目前支持文件式登录，不读取系统凭据库；容器、远程部署等情况请看[开发文档](docs/DEVELOPMENT.md#login-file-and-deployment)。

## 安装与启用

CodexGate 已收录至 [CPA 官方插件商店](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store)，推荐优先通过商店安装。

记住这个顺序：

**先安装并启用插件 → 再打开“使用 openai 提供商标识” → 重启 Codex。**

### 1. 商店安装（推荐）

1. 在 CPA / ECPA 插件商店中搜索 **CodexGate**（若未搜到，可先刷新商店列表）。
2. 点击安装，并确认在插件管理中处于启用状态。

原有的 API Key 正常保留即可，无需改动。

### 2. 在 ECPA 中开启设置

确认 CodexGate 已启用，再按图中的顺序操作：

1. 点击左侧的 **智能体配置**，选中 **Codex**。
2. 进入顶部的 **配置管理**。
3. 打开 **使用 openai 提供商标识**。

![ECPA 设置步骤：智能体配置、配置管理、使用 openai 提供商标识](docs/images/ecpa-openai-provider-zh-v1.png)

### 3. 重启 Codex

完全退出并重新打开 Codex 客户端，检查：

- dot 入口是否出现，并能正常回复。
- 新聊天是否能正常回复。
- CPA 使用记录中是否能看到请求。

## 手动安装（备用）

<details>
<summary>展开手动下载与安装步骤</summary>

如果无法通过商店安装，可以从 [Releases 下载正式版 v0.1.1](https://github.com/chatwithgod/cpa-plugin-codexgate/releases/tag/v0.1.1)。按你的电脑选择安装包：

| 系统 | 安装包 |
|---|---|
| Windows 64 位 | `codexgate_0.1.1_windows_amd64.zip` |
| Linux x86_64 | `codexgate_0.1.1_linux_amd64.zip` |
| Linux ARM64 | `codexgate_0.1.1_linux_arm64.zip` |
| macOS Intel | `codexgate_0.1.1_darwin_amd64.zip` |
| macOS Apple Silicon | `codexgate_0.1.1_darwin_arm64.zip` |

等当前任务结束后，先停止 CPA。

解压 ZIP，把里面的插件文件放进 CPA 的 `plugins` 对应子目录；缺少的文件夹可以自行创建：

| 系统 | `plugins` 内的文件位置 |
|---|---|
| Windows 64 位 | `windows/amd64/codexgate.dll` |
| Linux x86_64 | `linux/amd64/codexgate.so` |
| Linux ARM64 | `linux/arm64/codexgate.so` |
| macOS Intel | `darwin/amd64/codexgate.dylib` |
| macOS Apple Silicon | `darwin/arm64/codexgate.dylib` |

重新启动 CPA，在插件管理中找到并启用 **CodexGate**，再继续上面的 ECPA 设置和重启 Codex 步骤。

</details>

## 没有使用 ECPA？

直接使用原版 CPA 也可以，只是需要手动修改 **Codex 的配置文件**，设置提供商标识和 CPA 接口地址。

具体方式见[手动配置说明](docs/DEVELOPMENT.md#manual-codex-configuration)。使用 ECPA 开关的用户无需手动修改。

## 停用与卸载

按下面的顺序恢复即可：

1. 关闭 ECPA 的“使用 openai 提供商标识”，或恢复原来的 Codex 提供商配置。
2. 重启 Codex。
3. 在 CPA 插件管理中禁用 CodexGate。

如果还要删除插件文件，先停止 CPA，再删除 **CodexGate 对应的文件**。原有 API Key 不受影响。

## 开发文档与许可

- [开发与配置说明](docs/DEVELOPMENT.md)
- [测试记录](docs/VERIFICATION.md)
- [第三方来源与许可](THIRD_PARTY_NOTICES.md)

本项目采用 [MIT 开源协议](LICENSE)。
