# blueship

蓝区文档 **多线合入** 工具：Cursor Skill + 本地 CLI。

把一次文档改动同步到多条合入线 → 推到 **fork** → 在主仓建空白 Issue →（你人工建 MR 后）评论 `start build`。

支持的主仓（白名单）：

- `openharmony/docs`
- `openharmony/interface_sdk-js`
- `openharmony/interface_sdk_c`
- `HarmonyOS_Samples/guide-snippets`

---

## 依赖

- `git`
- 当前仓库已配置 **主仓 remote**（白名单内）+ **fork remote**（你自己的同名仓）
- 建 Issue / 评门禁需要 GitCode Token：
  ```bash
  git config --global gitcode.token <YOUR_TOKEN>
  # 或
  export GITCODE_TOKEN=<YOUR_TOKEN>
  ```

---

## 安装 CLI

### 方式 A：本机编译（推荐）

需要 [Go](https://go.dev/dl/) 1.22+。

```bash
git clone git@github.com:QkHearn/blueship.git
cd blueship/cli
chmod +x build.sh
./build.sh
```

产物在 `cli/dist/`：

| 文件 | 平台 |
|------|------|
| `blueship-darwin-arm64` | macOS Apple Silicon |
| `blueship-darwin-amd64` | macOS Intel |
| `blueship-windows-amd64.exe` | Windows x64 |
| `blueship-linux-amd64` / `arm64` | Linux |

把对应二进制拷到 `PATH`，或：

```bash
# macOS arm64 示例
sudo cp dist/blueship-darwin-arm64 /usr/local/bin/blueship
```

### 方式 B：Cursor Skill

将本仓库放到 Cursor 个人 skills，或软链：

```bash
ln -s /path/to/blueship ~/.cursor/skills/blueship
```

Skill 说明见根目录 [`SKILL.md`](./SKILL.md)。Agent 会优先调用 `blueship` CLI。

---

## 快速开始

在目标 git 仓库根目录：

```bash
# 1. 配置本仓（问一次）：选 upstream / fork、合入线
blueship init

# 2. 改文档并 git add（staged）
git add path/to/doc.md

# 3. （可选）让 Agent 按 Skill 做通顺性 review；确认后再改

# 4. 多线推送 + 建 Issue
blueship ship --series 1028 --lines master,sta,70

# 含周线时：
blueship ship --series 1028 --lines master,sta,70,weekly --weekly-date 20261012

# 5. 你在 GitCode 上从 fork 分支向主仓建 MR

# 6. 触发门禁
blueship gate --series 1028
# 或
blueship gate --mr 168397,168398
```

查看状态与建议：

```bash
blueship status
blueship status --json
```

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
