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
① 探测（series / ship|amend 推断、列出可选合入线）
    ↓
② 【确认门闸 A】展示并让用户选择/确认：要推哪些线（及 weekly 日期）
       未确认 → 禁止 push / 建 Issue / 建 PR
    ↓
③ 多线：fetch → 分支 → patch → commit -s → push fork
    ↓
④ 每条线：建空白 Issue
    ↓
⑤ 【确认门闸 B】PR 模版问卷：活跃分支 Y/N + 变更来源类型 + 跟踪号
       未齐 → 禁止建 PR / start build
    ↓
⑥ 按模版填 body → POST 建 PR（Fixes #Issue）→ 评论 start build
    ↓
⑦ 输出总表（远端分支 / Issue / PR / gate）
```

用户说「通顺」只做 review；说「start build / gate」且 PR 已在则只做评论。

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

## 提问策略

| 项 | 可否静默 | 规则 |
|----|----------|------|
| ship vs amend | 可推断 | 有 `{series}_*` / last-ship → amend；否则 ship；冲突再问 |
| series | 用户已写则用 | 「1029 ship」→ 1029；没有则问 |
| **要推哪些线** | **不可静默** | **门闸 A：必须展示并等用户确认**（见下） |
| weekly 日期 | — | 用户勾了 weekly 才问 |
| **活跃分支 Y/N（模版）** | **不可静默** | **门闸 B：明确勾选**；可默认=门闸 A 已选线为 Y，但须展示给用户改 |
| **变更来源类型** | **不可静默** | **门闸 B：四选一**，禁止猜 |
| **跟踪号** | **不可静默** | **门闸 B：纯数字**，禁止编造 |
| Issue / PR / start build | 默认开 | 用户说「先别…」则跳过对应步 |

### 门闸 A — 推送前（强制）

探测后**先发确认块，等用户回复再 push**。推荐格式：

```text
准备 series=1029，模式=ship（新开）
staged：<文件列表摘要>
fork remote：hearn

请选择要推的合入线（可多选，回复如：master,sta,70）：
- [ ] master → 分支 1029_master ← origin/master
- [ ] sta → 1029_sta ← origin/OpenHarmony_feature_sta_20260331
- [ ] 70 → 1029_70 ← origin/OpenHarmony-7.0-Release
- [ ] weekly → 1029_701 ← origin/weekly_YYYYMMDD（需补日期）

建议（来自配置，仅供参考，不能当已确认）：master,sta,70
```

用户也可以说「就建议的三条」/「全要除了 weekly」——必须有一句明确选择。  
**未确认线集合 → 禁止 fetch/push/建 Issue。**

### 门闸 B — 建 PR 前（强制）

push + Issue 完成后，再发模版问卷（可与门闸 A 合并成一次，若用户开场已答齐则可跳过对应项）：

```text
PR 模版确认（docs）：

1) 活跃分支同步（Y/N，可改）：
   - master: Y/N
   - sta (OpenHarmony_feature_sta_20260331): Y/N
   - weekly_YYYYMMDD: Y/N
   - OpenHarmony-7.0-Release: Y/N
   （默认：门闸 A 已推的线=Y，未推=N；仍须用户确认或改口）

2) 文档变更来源（四选一，原因需唯一）：
   - 新增需求 / 变更需求 / 问题修复 / 主动优化
   - 跟踪编号（纯数字，不带前缀）：？

3) 是否继续：建 PR + start build？（默认是；说「先别」则停）
```

**类型或跟踪号缺失 → 禁止 POST PR / start build。** 禁止用「fix bug」或空 body 凑合建 PR。

---

## ① 探测

```bash
git remote -v
git diff --cached --stat
git branch --list "${SERIES}_*"
test -f .blueship.json && cat .blueship.json
```

无 staged → 停，请用户 `git add`。  
staged 几乎为空（如只多空行）→ **警告**，用户确认再进门闸 A。

探测结束后立刻出门闸 A，**不要**直接推。

---

## ② 多线推 fork（核心机械步骤）

**仅在门闸 A 通过后执行。** 只处理用户确认的线。

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

**仅在门闸 B 通过后执行。** 活跃分支 Y/N、变更来源类型、跟踪号一律用用户确认值，禁止自行默认成「全 Y」或臆造类型。

1. 读仓内模版（优先）：

   - `.gitcode/PULL_REQUEST_TEMPLATE.zh-CN.md`
   - 其次 `.gitcode/PULL_REQUEST_TEMPLATE.md` / `.gitee/...` / `.github/...`

2. **按门闸 B 答案填模版**（不要交空 body——API **不会**自动套模版）：

   - 活跃分支：按用户 Y/N 填写（weekly 写清具体日期分支名）
   - 文档变更来源：只填用户选中的那一类 + 跟踪号
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
