# cc98-autosign-fast

一个面向 CC98 的轻量自动签到工具。

当前主实现是 Go 版，发布包会直接提供：

- Windows：`cc98-autosign-fast.exe` + `.env`
- Linux：`cc98-autosign-fast` + `.env`

## 使用方式

### Windows

1. 到 GitHub Releases 下载 Windows 发布包
2. 解压后填写 `.env`
3. 双击 `cc98-autosign-fast.exe`
4. 程序会自动签到，并在窗口里显示结果

### Linux

1. 到 GitHub Releases 下载 Linux 发布包
2. 解压后填写 `.env`
3. 在终端执行 `./cc98-autosign-fast`
4. 程序会输出签到结果后直接退出

如果 `.env` 不存在，程序会自动生成一份模板，并提示你填写后重新运行。

## `.env` 多账号示例

```env
WEBVPN_USER=你的WebVPN账号
WEBVPN_PASS=你的WebVPN密码

CC98_ACCOUNT_COUNT=2

CC98_USER_1=第一个CC98账号
CC98_PASS_1=第一个CC98密码

CC98_USER_2=第二个CC98账号
CC98_PASS_2=第二个CC98密码
```

如果你还有更多账号，就继续往下写：

```env
CC98_USER_3=第三个CC98账号
CC98_PASS_3=第三个CC98密码
```

## 输出示例

```text
账号1(anchorite) ✅ 签到成功 · 🎁 1141财富值 · 📅 连续 30 天
账号2(example2) ✅ 签到成功 · 🎁 1155财富值 · 📅 连续 5 天
Cookie ✅ 命中
耗时 ⏱ 0.13s
```

## 请求链

当前实现把请求分成两段：WebVPN 认证链和 CC98 业务链。

### WebVPN 认证链

冷启动时，WebVPN 部分只做最小登录流程：

1. `GET /login`
2. `POST /do-login`
3. 只有返回 `NEED_CONFIRM` 时才 `POST /do-confirm-login`

热启动时，会先读取运行目录下的 `.webvpn-cookie-cache.json`。如果缓存可用，整段 WebVPN 登录会被直接跳过。

如果首个账号的 `token` 请求被打回 WebVPN 登录页，程序会：

1. 清空当前 cookie
2. 重新执行一次 WebVPN 认证链
3. 只重试当前账号一次

### CC98 业务链

WebVPN 会话建立完成后，每个账号都会走同一条业务链：

1. `POST connect/token`
2. `POST me/signin`
3. `GET me/signin`

其中：

- `connect/token` 用来换取当前账号的 access token
- `me/signin` 用来执行签到
- `GET me/signin` 用来补充读取连续签到天数和今日奖励

## 为什么快

### WebVPN 部分为什么快

- WebVPN 只保留最小登录链，不走主页跳转
- 不再请求 `user/info`
- 命中 cookie 缓存后，整段 WebVPN 登录会被直接跳过

### CC98 部分为什么快

- 不做动态 host 改写，直接走固定的 `connect/token` 和 `me/signin` 路由
- 多账号共享同一个 WebVPN 会话，不会为每个账号重复登录
- 热启动时只剩真正的业务请求

所以冷启动时主要耗时在 WebVPN 登录，热启动时则主要耗时在 CC98 的 token / signin / sign-info 这几步。

## 固定路径与缓存机制

- 当前实现依赖已经验证过的固定 WebVPN token/sign 路由
- 程序会在运行目录写入 `.webvpn-cookie-cache.json`
- 缓存里真正关键的是 `wengine_vpn_ticketwebvpn_zju_edu_cn`
- `route` 使用程序内置的固定默认值

这也是当前实现能够在命中缓存后明显提速的原因。

## 仓库结构

- `src/`
  - Go 主实现与测试
- `python-reference/`
  - 当前 Python 参考实现，仅用于协议参考与调试
- `.github/workflows/release.yml`
  - GitHub Actions 自动构建并发布 release

## 本地构建

Windows:

```powershell
./build-release.ps1
```

Linux:

```bash
bash ./build-release.sh
```

构建完成后，产物会生成到本地 `dist/` 目录，但这个目录不会进入 git。

## 说明

- 不要提交 `.env`、`.webvpn-cookie-cache.json`、`dist/` 或任何真实账号密码
- `python-reference/` 不是主发布入口，普通用户优先使用 Go 版 release
