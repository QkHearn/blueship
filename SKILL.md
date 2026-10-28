---
name: blueship
description: >-
  蓝区文档多线合入（Skill 主编排）：staged → 多线推 fork → 按模版建 PR 并关联 Issue →
  评论 start build。Triggers on: blueship, /blueship, 「MMDD blueship」(如 1028 blueship),
  多线推送, 推 fork, 通顺, start build, gate, amend, 文档多线合入.
---

# blueship（Skill-first）

**编排以本 Skill 为准。** Agent 用 `git` + GitCode API（curl）跑完整链路；`cli/` 二进制仅作可选加速，**失败或行为怪异时忽略 CLI，按下文步骤直做**。  
运行合入时**不要读** `cli/*.go`（改工具本身除外）。

## 端到端流程

```text
staged diff
    ↓
① 探测 & 推断（series / ship|amend / lines）
    ↓
② 只问缺口（变更来源+跟踪号；weekly 日期；用户说「先别」则停 PR/gate）
    ↓
③ 多线：fetch 基线 → 建/切分支 → 打 patch → commit -s → push fork
    ↓
④ 每条线：建空白 Issue
    ↓
⑤ 读 PR 模版 → 填好 body → POST 建 PR（Fixes #Issue）
    ↓
⑥ 每条 PR 评论：start build
    ↓
⑦ 输出总表（远端分支 / Issue / PR / gate）
```

用户说「通顺」只做 review；说「start build / gate」且 PR 已在则只做 ⑥。

---

## 职责

| 做 | 不做 |
|----|------|
| 检查双 remote + token | 不打印 token |
| review 只建议，确认后才改文 | 缺 remote/token 不硬推 |
| 多线推 **fork** | 不推 upstream 主仓分支 |
| 按仓内模版建 PR + 关联 Issue | 不把无关改动夹进同一 PR |
| 评 `start build` | |

白名单 upstream：`openharmony/docs` · `openharmony/interface_sdk-js` · `openharmony/interface_sdk_c` · `HarmonyOS_Samples/guide-snippets`  
fork = 同 repo 名、owner 不在白名单。

---

## 配置（每仓一次）

根目录 `.blueship.json`（无则根据 remote **直接写入**，不必死磕交互式 `init`）：

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
  "create_issue": true,
  "create_pr": true,
  "start_build": true
}
```

Token：`git config --get gitcode.token` 或 `$GITCODE_TOKEN`（可 `source ~/.zshrc`）。

API 基址：`https://gitcode.com/api/v5`，头：`PRIVATE-TOKEN: <token>`。

---

## 提问策略（能推断就别问）

| 项 | 默认 | 才问 |
|----|------|------|
| ship vs amend | 有 `{series}_*` 或 last-ship → amend；否则 ship | 多 series 且用户没点名 |
| series | 用户「1028 blueship」 | 完全没给 |
| lines | 配置里非 weekly 键；默认 `master,sta,70` | 用户要改集合 |
| weekly 日期 | — | 线含 weekly |
| **变更来源 + 跟踪号** | — | **建 PR 前必问一次**（除非本轮已说清） |
| Issue / PR / start build | 配置默认全开 | 用户说「先别 / 先不建 PR / 先别 start build」 |

变更来源（模版要求唯一）：`新增需求` | `变更需求` | `问题修复` | `主动优化` + **纯数字**跟踪号。

---

## ① 探测

```bash
git remote -v
git diff --cached --stat
git branch --list "${SERIES}_*"
test -f .blueship.json && cat .blueship.json
# 可选：blueship status --json
```

无 staged → 停，请用户 `git add`。  
staged 若几乎为空（例如只多空行）→ **先警告**，用户确认再继续。

---

## ② 多线推 fork（核心机械步骤）

设：`UP`=upstream remote，`FK`=fork remote，`SERIES`，每条 `LINE` → `BASE`（来自配置），本地/远端分支名：

- 普通线：`BR=${SERIES}_${LINE}`（如 `1028_master`）
- weekly：`BR=${SERIES}_701`

**新开 ship（每条线）：**

```bash
git fetch "$UP" "$BASE"
git checkout -B "$BR" "$UP/$BASE"
git diff --cached > /tmp/blueship.patch   # 在切分支前于原分支导出一次即可复用
git apply --3way /tmp/blueship.patch || git apply /tmp/blueship.patch
git add -A
git commit -s -m "${MSG:-fix bug}"
git push -u "$FK" "$BR:$BR"
```

**amend：** 在已有 `$BR`（或 `git fetch "$FK" "$BR"` 后 checkout）上对 tip 打增量，`commit --amend --no-edit -s`，再 `git push --force-with-lease "$FK" "$BR:$BR"`。Δ 相对 fork tip；涉及 tip 有/工作区无的删除须确认。

切分支前务必：**先把 staged patch 存到临时文件**（checkout 会弄丢 index 时需可重放）。

每条线记下：`remote_ref=$FK/$BR`、`sha`、浏览器 `https://gitcode.com/<forkOwner>/docs/tree/$BR`。

---

## ③ 建空白 Issue

```bash
curl -sS -X POST "https://gitcode.com/api/v5/repos/${OWNER}/${REPO}/issues" \
  -H "PRIVATE-TOKEN: $TOKEN" -H "Content-Type: application/json" \
  -d "{\"title\":\"${MSG} [${BASE} / ${BR}]\",\"body\":\"-\"}"
```

`body` 必须非空（用 `"-"`）。记下 `number` / `html_url`。  
用户说「先别建 Issue」则跳过；无 Issue 时 PR body 不写 `Fixes #`。

---

## ④ 按模版建 PR

1. 读仓内模版（优先）：

   - `.gitcode/PULL_REQUEST_TEMPLATE.zh-CN.md`
   - 其次 `.gitcode/PULL_REQUEST_TEMPLATE.md` / `.gitee/...` / `.github/...`

2. **Agent 填模版**（不要交空 body——API **不会**自动套模版）：

   - 活跃分支：本次 `lines` 涉及的勾 **Y**，其余 **N**（weekly 写清具体 `weekly_YYYYMMDD`）
   - 文档变更来源：只填用户选定的那一类 + 跟踪号
   - 自检 / 示例代码：按改动事实勾；无示例可写「不涉及」
   - 文末加：`Fixes #<issue_number>`（有 Issue 时；关联必须写在 **PR body**，评论无效）

3. 创建：

```bash
curl -sS -X POST "https://gitcode.com/api/v5/repos/${OWNER}/${REPO}/pulls" \
  -H "PRIVATE-TOKEN: $TOKEN" -H "Content-Type: application/json" \
  -d "{
    \"title\": \"${MSG} [${BASE} / ${BR}]\",
    \"head\": \"${FORK_OWNER}:${BR}\",
    \"base\": \"${BASE}\",
    \"body\": \"<填好的模版 Markdown>\",
    \"fork_path\": \"${FORK_OWNER}/${REPO}\"
  }"
```

- 409 且提示已有 MR → 取已有 `!N`，必要时 `PATCH` 更新 body（补模版 / Fixes）。
- 504 后用 list/`head` 查是否已创建，避免重复。

---

## ⑤ start build

```bash
curl -sS -X POST \
  "https://gitcode.com/api/v5/repos/${OWNER}/${REPO}/pulls/${PR}/comments" \
  -H "PRIVATE-TOKEN: $TOKEN" -H "Content-Type: application/json" \
  -d '{"body":"start build"}'
```

正文必须是字面量 `start build`。用户说先别则跳过。

---

## ⑥ 输出（必须）

| 线 | 远端分支 | SHA | Issue | PR | start build |
|----|----------|-----|-------|----|-------------|
| master | `hearn/1028_master` | `abc…` | #… url | !… url | ok / skipped |

附上每个 `remote_url` / PR 链接。提醒：门禁结果在 PR 页看。

落盘：写 `.blueship-last-ship.json`（series、lines、branches、issues、prs），供下次 amend / gate。

---

## review（可选前置）

读 staged diff → 表：**位置 | 原文 | 问题 | 建议** → **不改文件**，等用户「按建议改 / 跳过」后再改，再进合入。

---

## 可选 CLI

路径：`~/.cursor/skills/blueship/cli/dist/blueship-<os-arch>` 或 PATH 中 `blueship`。  
仅当 Agent 想省事时可调用 `ship` / `gate` / `status`；**创建 PR、填模版、关联 Issue 以本 Skill 步骤为准**（CLI 若尚未覆盖 PR 不要干等）。  
编译：`bash ~/.cursor/skills/blueship/cli/build.sh`。

---

## 触发词

| 说法 | 行为 |
|------|------|
| `blueship` / `/blueship` | 全流程（status → 推断 → …） |
| `1028 blueship` | series=1028 全流程 |
| `通顺` | 仅 review |
| `amend` / `补一刀` | 强制 amend 后再 ④⑤（若需） |
| `start build` / `gate` | 仅对已有 PR 评门禁 |
| `先别 PR` / `先别 start build` | 对应阶段跳过 |
