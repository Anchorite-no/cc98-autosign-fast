# cc98-autosign-fast

一个面向 CC98 的轻量自动签到工具。

当前主实现是 Go 版，目标是提供一个可直接分发的 Windows 双击成品：

- `cc98-autosign-fast.exe`
- `.env`

## 使用方式

### 普通用户

1. 到 GitHub Releases 下载最新发布包
2. 解压后填写 `.env`
3. 双击 `cc98-autosign-fast.exe`
4. 程序会自动签到，并在窗口里显示结果

如果 `.env` 不存在，程序会自动生成一份模板，并提示你填写后重新运行。

### `.env` 多账号示例

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
账号1 ✅ 签到成功 · 🎁 1141财富值 · 📅 连续 30 天
账号2 ✅ 签到成功 · 🎁 1155财富值 · 📅 连续 5 天
Cookie ✅ 命中
耗时 ⏱ 0.13s
```

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

- 程序会在运行目录写入 `.webvpn-cookie-cache.json` 复用 WebVPN 登录态
- 不要提交 `.env`、`.webvpn-cookie-cache.json`、`dist/` 或任何真实账号密码
- `python-reference/` 不是主发布入口，普通用户优先使用 Go 版 release
