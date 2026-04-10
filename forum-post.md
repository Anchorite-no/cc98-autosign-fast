# [分享] 一个更轻更快的 CC98 自动签到工具：cc98-autosign-fast

之前我自己老是忘记 CC98 签到，所以最早写过一版浏览器自动化脚本。能用是能用，就是慢，而且整个链路比较重。于是就想写一个api版本，同时借鉴了一部分犬戎大佬的思路，做了深度流程优化。

默认都走webvpn所以不需要校网。核心请求步骤简化到四步，并加上了cookie缓存功能，所以速度极快。在校内签到三个账号用时在**0.15秒**左右，我在东京服务器运行用时在**1.5秒**左右，基本做到双击即签。喜欢的话可以在 GitHub 点一个 star。

项目地址：

- GitHub: https://github.com/Anchorite-no/cc98-autosign-fast
- Releases: https://github.com/Anchorite-no/cc98-autosign-fast/releases

目前发布包已经支持：

- Windows：解压后填写 `.env`，双击 `cc98-autosign-fast.exe`
- Linux：解压后填写 `.env`，执行 `./cc98-autosign-fast`

---

## 输出示例

```text
账号1(anchorite) ✅ 签到成功 · 🎁 1141财富值 · 📅 连续 30 天
账号2(example2) ✅ 签到成功 · 🎁 1155财富值 · 📅 连续 5 天
Cookie ✅ 命中
耗时 ⏱ 0.13s
```


## 核心功能

- 支持通过 WebVPN 自动签到
- 支持多账号顺序签到
- 支持缓存 WebVPN 登录态，加快后续启动速度
- 输出签到结果、今日奖励、连续签到天数和总耗时
- Windows 可以直接双击运行，Linux 可以直接在终端运行

---

## 配置方式

`.env` 里只需要填 WebVPN 账号和 CC98 账号密码。多账号也不用额外折腾，按顺序往下写就行。

```env
WEBVPN_USER=你的WebVPN账号
WEBVPN_PASS=你的WebVPN密码

CC98_ACCOUNT_COUNT=2

CC98_USER_1=第一个CC98账号
CC98_PASS_1=第一个CC98密码

CC98_USER_2=第二个CC98账号
CC98_PASS_2=第二个CC98密码
```

如果还有更多账号，就继续补：

```env
CC98_USER_3=第三个CC98账号
CC98_PASS_3=第三个CC98密码
```

---

## 请求链

### 1. WebVPN 认证链

冷启动时，WebVPN 这边只走最小登录流程：

1. `GET /login`
2. `POST /do-login`
3. 只有返回 `NEED_CONFIRM` 时才 `POST /do-confirm-login`

这里的 `do-confirm-login` 不是每次都会跑，只有 WebVPN 提示“已经在别处登录，需要确认是否顶掉旧会话”时才会触发。

热启动时，程序会先读取运行目录下的 `.webvpn-cookie-cache.json`。如果缓存可用，整段 WebVPN 登录会直接跳过。

如果首个账号的 `token` 请求被打回 WebVPN 登录页，程序会：

1. 清空当前 cookie
2. 重新执行一次 WebVPN 登录链
3. 只重试当前账号一次

### 2. CC98 业务链

WebVPN 会话建立完成后，每个账号都会走同一条业务链：

1. `POST connect/token`
2. `POST me/signin`
3. `GET me/signin`

这三步分别负责：

- `connect/token`：换取当前账号的 access token
- `me/signin`：执行签到
- `GET me/signin`：补充读取连续签到天数和今日奖励

---

## 为什么快

### WebVPN 这边为什么快

- 只保留最小登录链，不走主页跳转
- 不再请求 `user/info`
- 命中 cookie 缓存后，整段 WebVPN 登录直接跳过

### CC98 这边为什么快

- 不做动态 host 改写，直接走固定的 `connect/token` 和 `me/signin` 路由
- 多账号共享同一个 WebVPN 会话，不会为每个账号重复登录
- 热启动时只剩真正的业务请求

所以冷启动时主要耗时在 WebVPN 登录，热启动时主要耗时就在 CC98 的 `token / signin / sign-info` 这几步。

---

## 固定路径与缓存机制

这个版本当前依赖已经验证过的固定 WebVPN token/sign 路由。

程序会在运行目录写入 `.webvpn-cookie-cache.json`，里面缓存的是 WebVPN 登录态。当前缓存里真正关键的是：

- `wengine_vpn_ticketwebvpn_zju_edu_cn`

另外 `route` 在当前实现里使用固定默认值。
也正因为这样，命中缓存后可以直接跳过 WebVPN 登录链，速度会明显更快。

---

## 说明

- 普通用户优先直接使用 release
- 不要提交自己的 `.env`、`.webvpn-cookie-cache.json` 或任何真实账号密码
- 如果有使用中的问题，也欢迎提 issue 或直接交流
