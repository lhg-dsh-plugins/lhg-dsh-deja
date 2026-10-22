# BUGFIX

## 标题

`deja files` / `deja how` 的范围行把带连字符的项目叫成一个不存在的名字（上游 issue #5321）：项目目录 `proj/my-app` 在结果行里写作 `proj/my-app`，页脚与 `--json` 的 `project` 字段却写作 `my/app` —— 一个任何行都不显示、`--project` 也不接受的名字。

版本：上游 0.23.0 → 本版 0.23.1（patch +1）。

## 证据链（修复前）

新增回归测试 `cmd/deja/files_scope_name_test.go` 在上游 HEAD `7a85f1524253babf6603dc5bf91d88bfaffc0dbd` 上三条全红，原始输出：

```
files touched while working on "release" — 1 session
  …/001/proj/my-app/notes.txt                              1 session
— in my/app only; `--all-projects` looks at every project

$ make release
  ran once in 1 session · last Jul 1
— in my/app only; `--all-projects` looks at every project

files --json: project = "my/app", want "proj/my-app"
```

## 根因

`howScopeName`（cmd/deja/how.go）直接返回项目名候选列表的第一项。候选由 `digest.ProjectNameCandidates` 生成，第一项（无别名时）是 `sources.PlainProjectName(cwd)`，即 Claude 编码目录名经 `decodeProjectBase` 的虚线启发式读回：`my-app` 的每个 `-` 都被当作 `/`，得到 `my/app`。而结果行携带的项目名来自 `cwdProjectName`，是 parent/base 形式 `proj/my-app`（有别名文件时为别名）。页脚/"project" 字段因此与行名对不上，且 `deja files release --project my/app` 按 issue 实测查不到任何会话。

## 改动文件

- `cmd/deja/how.go`：`howScopeName(scope)` 改为 `howScopeName(scope, cwd)` —— 候选多于一项时，若别名文件命名了该目录且它正是首项候选，返回别名；否则返回候选列表中与 cwd 的 parent/base 形式（`Base(Dir(cwd)) + "/" + Base(cwd)`）相同的那一项；都对不上时回退到原行为（首项）。`runHow` 相应捕获 `cwd` 并传给 `writeHowJSON` 与 `projectScopeHint`。
- `cmd/deja/files.go`：`runFiles` 捕获 `cwd`，三处 `--json` 的 `Project` 字段与 `projectScopeHint` 调用改传新签名；`projectScopeHint(scope, cwd, widened)` 签名同步。
- `cmd/deja/main.go`：blame 的 "no session in %s mentions" 一行改传 `howCwd()`。
- `cmd/deja/mcp_orient.go`：orient 的 scope 名改传已有的 `cwd` 变量。
- `cmd/deja/files_scope_name_test.go`（新增）：三条回归测试 —— `files` 页脚、`how` 页脚、`files --json` 的 `project` 字段，均断言名字与行名一致（`proj/my-app`），且页脚不得出现 `my/app`。
- `package.json`、`extensions/dsh/package.json`：version 0.23.0 → 0.23.1，repository.url 改指本 fork 仓库。其余字段不动。

`--project` 显式给定时 scope 只有一项，仍原样返回用户输入的名字（issue 中的对照行为不
变）；`--all-projects` 时 scope 为空，仍返回 ""。

## 复验结果

- 新增三条回归测试：修复前全红（输出见上），修复后全绿。
- `go test ./cmd/deja/ -count=1`（全套件，-v 落盘核对）：3775 个测试通过，2 个失败 —— `TestBriefNamesWhyABuildCouldNotRun` 与 `TestDoctorStoreStates/denied`。用 `git stash -u` 回到 pristine 上游树单跑同两条，结果同样失败：两者都是沙箱以 root 运行的环境口径（权限拒绝类断言对 root 不成立、构建失败口径不同），非本次引入。
- `go build ./...`、`go vet ./cmd/deja/`、`gofmt -l`（改动文件）均干净。
- DSH 插件 JS 测试：`node --test extensions/dsh/test/*.test.js` 39/39 通过（与基线一致，本次未改 JS）。

## 验证边界

- 版本戳：上游的版本号由其发布流程在约 20 处渠道清单（claude-plugin、codex-plugin、gemini-extension.json 等）统一打戳。本版只按 fork 口径 bump 了 DSH 插件的两个清单（仓库根 `package.json` 与 `extensions/dsh/package.json`）；其余渠道清单仍为 0.23.0，与上游发布工具下次统一打戳前一致地滞后，不影响本插件的功能与测试（仓库内无跨清单版本一致性测试断言这两处必须与其余渠道同步）。
- WorkBuddy 聊天文件夹这类 cwd 经 `projectRoot` 改写后再取候选的边角情形：若 cwd 原始 parent/base 形式不在候选列表中，`howScopeName` 回退到首项候选，即与修复前行为一致，不构成回归。
- 未实测 Windows 路径形态（候选中的 `C:` 驱动器形式）；修复逻辑只用 `path/filepath` 的 Base/Dir 组合与候选列表成员判断，不新增平台假设。
