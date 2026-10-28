---
name: blueship
description: >-
  蓝区文档多线合入助手：按指令改文档、提交前通顺性建议（确认后才改）、多线建分支推 fork、
  主仓建空白 Issue、对已有 MR 评论 start build。Triggers on: blueship, /blueship,
  多线推送, 推 fork, 空白 Issue, start build（文档合入）, 文档多线.
---

# blueship

Skill 负责问答与确认；**可执行动作优先调用 CLI** `blueship`（见 `cli/`），禁止在能调 CLI 时手写一长串等价 git/curl。

## 职责

**做：** 依赖检查；按指令改文档（可先索引定位）；`review` 只建议不擅自改；多线落分支并推 **fork**；主仓建空白 Issue；对已有 MR 评 `start build`；迭代 amend 再推。

**不做：** 不替用户建 MR（人工 fork→主仓）；不在缺 remote/token/冲突未确认时硬推；不打印 token。

## CLI 位置

```bash
# 优先 PATH 中的 blueship；否则用本 skill 内预编译二进制
BLUESHIP_BIN="$(command -v blueship || true)"
if [ -z "$BLUESHIP_BIN" ]; then
  case "$(uname -s)-$(uname -m)" in
    Darwin-arm64) BLUESHIP_BIN="$HOME/.cursor/skills/blueship/cli/dist/blueship-darwin-arm64" ;;
    Darwin-x86_64) BLUESHIP_BIN="$HOME/.cursor/skills/blueship/cli/dist/blueship-darwin-amd64" ;;
    MINGW*|MSYS*|CYGWIN*|Windows_NT*) BLUESHIP_BIN="$HOME/.cursor/skills/blueship/cli/dist/blueship-windows-amd64.exe" ;;
    Linux-x86_64) BLUESHIP_BIN="$HOME/.cursor/skills/blueship/cli/dist/blueship-linux-amd64" ;;
    Linux-aarch64) BLUESHIP_BIN="$HOME/.cursor/skills/blueship/cli/dist/blueship-linux-arm64" ;;
  esac
fi
"$BLUESHIP_BIN" status --json
```

编译：`bash ~/.cursor/skills/blueship/cli/build.sh`

## 主仓白名单

仅下列 `owner/repo` 可作 upstream：

- `openharmony/docs`
- `openharmony/interface_sdk-js`
- `openharmony/interface_sdk_c`
- `HarmonyOS_Samples/guide-snippets`

fork = 同 repo 名、owner 不在白名单中的 remote。

## 项目配置（每仓一次）

无 `.blueship.json` 时先跑：

```bash
blueship init
```

写入：`upstream` / `fork` remote 名、`lines` 映射、是否默认建 Issue。缺双 remote 或主仓不在白名单 → **停**，提示如何 `git remote add`，不要静默添加除非用户明确要求。

## 每次必问

1. 新开 `ship` 还是 `amend`（或只要 `gate` / `review`）
2. 改动来源（默认 staged）
3. 上哪几条线（多选；可推荐配置里的 lines；含 weekly 再问日期）

Issue / start build：跟项目默认；用户说「先别」再关。

## 流程

### review（提交前）

1. `blueship status --json` 看工作区
2. 阅读 staged（或指定）diff，列出通顺/语法建议表：**位置 | 原文 | 问题 | 建议**
3. **不改文件**；等用户说「按建议全改 / 只改第 N 条 / 跳过」后再改

### ship（新开）

```bash
blueship ship --series <MMDD或名> --lines master,sta,70 [--weekly-date YYYYMMDD] [--message "..."]
# 默认 create-issue；不要 Issue 加 --no-issue
```

CLI 会：fetch 基线 → 建 `{series}_{line}` → 应用 staged 变更 → commit → push fork →（可选）主仓建空白 Issue。

Agent 输出表格：线 / 分支 / sha / issue。提醒用户人工建 MR。

### amend

```bash
blueship ship --amend --series <名> [--lines ...]
```

对已有 tip 打增量并 `--force-with-lease` 推 fork。算 Δ 时相对 **fork tip**，注意 tip 有而目标无的删除须向用户确认。

### gate

MR 已存在后：

```bash
blueship gate --series <名>
# 或
blueship gate --mr 168397,168398
```

评论字面量 `start build`。

### 按指令改文档

先定位再改；改完走 review → 用户确认 → ship。无把握时先列出将改路径请用户确认。

## Token

`git config --get gitcode.token` 或 `$GITCODE_TOKEN`（可 `source ~/.zshrc`）。用于 Issue 与 gate。勿打印。

## 状态驱动建议

优先：`blueship status --json`，按返回的 `state` / `suggestion` 提示用户下一步，不要假设上一轮对话进度。
