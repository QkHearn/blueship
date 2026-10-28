# blueship

蓝区文档 **多线合入**：以 **Cursor Skill 主编排**（`git` + GitCode API），CLI 仅为可选加速。

完整链路：

**staged → 多线推 fork → 按仓内模版建 PR（关联 Issue）→ 评论 `start build`**

支持的主仓（白名单）：

- `openharmony/docs`
- `openharmony/interface_sdk-js`
- `openharmony/interface_sdk_c`
- `HarmonyOS_Samples/guide-snippets`

流程细节见 [`SKILL.md`](./SKILL.md)。

---

## 依赖

- `git` + Cursor（装好本 Skill）
- 双 remote：白名单 **upstream** + 自己的 **fork**
- GitCode Token（Issue / PR / start build）：
  ```bash
  git config --global gitcode.token <YOUR_TOKEN>
  # 或
  export GITCODE_TOKEN=<YOUR_TOKEN>
  ```

---

## 推荐用法（Skill）

```bash
# 1. 安装 Skill（一次）
git clone git@github.com:QkHearn/blueship.git
ln -s "$(pwd)/blueship" ~/.cursor/skills/blueship

# 2. 目标仓准备（一次）：双 remote + 可选 .blueship.json
# 3. 改文档并 stage
git add path/to/doc.md

# 4. 在 Cursor 对话里说：
#    1028 blueship
# Agent 会：推断 ship/amend → 多线推 fork → 问变更来源/跟踪号
#          → 填 PR 模版建 PR → Fixes #Issue → start build
```

触发词：`blueship` / `1028 blueship` / `通顺` / `start build` / `amend`。

---

## 可选：CLI

预编译在 `cli/dist/`（多平台）。适合本机脚本；**建 PR / 填模版以 Skill 为准**。

```bash
chmod +x cli/dist/blueship-darwin-arm64
sudo cp cli/dist/blueship-darwin-arm64 /usr/local/bin/blueship
blueship status --json
# 仅推 fork + Issue（不含按模版建 PR）：
blueship ship --series 1028 --lines master,sta,70
blueship gate --series 1028
```

本机编译：`cd cli && ./build.sh`。

---

## 命令说明

| 命令 | 作用 |
|------|------|
| `blueship init` | 写入 `.blueship.json`（upstream/fork/lines/是否建 Issue） |
| `blueship status [--json]` | 探测配置、staged、remote、token，给出下一步建议 |
| `blueship ship --series NAME --lines a,b,c` | 按主仓基线建 `{series}_{line}`，应用 **staged** diff，推 fork；默认可建空白 Issue |
| `blueship ship --amend --series NAME --lines …` | 在已有分支上 amend + `--force-with-lease` 推 fork |
| `blueship gate --series NAME` | 查找该 series 前缀的打开 MR，评论 `start build` |
| `blueship gate --mr N,N` | 对指定 MR 评论 `start build` |
| `blueship version` | 版本号 |

常用 flag：

- `--message "..."`：commit / Issue 标题前缀（默认 `fix bug`）
- `--no-issue`：ship 时不建 Issue
- `--weekly-date YYYYMMDD`：展开配置里的 `weekly_{date}` 基线

`docs` 仓默认线映射（`init` 后可改 `.blueship.json`）：

| 线名 | 主仓分支 | 本地/fork 分支名 |
|------|----------|------------------|
| `master` | `master` | `{series}_master` |
| `sta` | `OpenHarmony_feature_sta_20260331` | `{series}_sta` |
| `70` | `OpenHarmony-7.0-Release` | `{series}_70` |
| `weekly` | `weekly_{date}` | `{series}_701` |

---

## 配置示例 `.blueship.json`

```json
{
  "upstream": "origin",
  "fork": "hearn",
  "upstream_repo": "openharmony/docs",
  "lines": {
    "master": "master",
    "sta": "OpenHarmony_feature_sta_20260331",
    "70": "OpenHarmony-7.0-Release",
    "weekly": "weekly_{date}"
  },
  "create_issue": true
}
```

- **upstream**：主仓 remote 名（URL 须在白名单）
- **fork**：你的 fork remote 名
- **MR 由人工创建**；CLI 不建 MR

---

## 退出码

| 码 | 含义 |
|----|------|
| 0 | 成功 |
| 2 | 缺依赖/配置（remote、token、`.blueship.json`） |
| 3 | 需要用户补充参数或确认（无 staged、缺 series 等） |

---

## 开发

```bash
cd cli
go build -o dist/blueship .
./build.sh   # 交叉编译多平台
```

---

## 职责边界（给使用者）

| 你做 | blueship 做 |
|------|-------------|
| 写文档、建 GitCode MR | 多线分支、推 fork、空白 Issue、`start build` |
| 确认通顺性修改 | Skill 侧 review 只建议不擅自改 |
| 补齐双 remote / token | `status` / `init` 提示缺什么 |
