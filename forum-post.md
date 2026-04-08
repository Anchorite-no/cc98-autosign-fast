# [分享] 一个更轻量的 CC98 自动签到工具：cc98-autosign-fast

最近把自己在用的 CC98 自动签到工具整理成了一个单独的公开仓库，想简单分享一下。

它的目标不是做一个“大而全”的签到框架，而是尽量把事情做简单：

- 校外可用
- 配置简单
- 双击就能跑
- 速度尽量快

目前项目地址：

- GitHub 仓库：[cc98-autosign-fast](https://github.com/Anchorite-no/cc98-autosign-fast)
- Releases 下载页：[v0.1.1 及后续版本](https://github.com/Anchorite-no/cc98-autosign-fast/releases)

---

## 这个工具解决了什么问题

之前我自己折腾 CC98 自动签到时，最大的痛点主要有几个：

1. 很多方案更偏“脚本”，不是“成品”
2. 校外走 WebVPN 时，流程比较绕
3. 普通用户使用门槛高，经常要装 Python、装依赖、开命令行
4. 一些实现逻辑比较重，实际签到路径不够短

这个工具主要就是想把这些问题压下来。

---

## 这个版本的特点

- Windows 版是 `exe + .env`
- 下载 release 后，填好 `.env` 就可以直接双击运行
- 支持多个 CC98 账号
- 支持 WebVPN
- 支持 cookie 缓存复用
- 输出会直接显示：
  - 是否签到成功
  - 今日奖励
  - 连续签到天数
  - 是否命中 cookie 缓存
  - 总耗时

相较于很多现有的 CC98 autosign，我自己感觉它更偏向：

- 轻量
- 直接
- 易分享
- 更适合普通用户

---

## 使用方式

### Windows

1. 到 release 页面下载 Windows 压缩包
2. 解压
3. 填写同目录下的 `.env`
4. 双击 `cc98-autosign-fast.exe`

### Linux

1. 到 release 页面下载 Linux 压缩包
2. 解压
3. 填写同目录下的 `.env`
4. 在终端执行：

```bash
./cc98-autosign-fast
```

---

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

如果你有更多账号，就继续往下写：

```env
CC98_USER_3=第三个CC98账号
CC98_PASS_3=第三个CC98密码
```

---

## 运行效果示例

```text
账号1 ✅ 签到成功 · 🎁 1141财富值 · 📅 连续 30 天
账号2 ✅ 签到成功 · 🎁 1155财富值 · 📅 连续 5 天
Cookie ✅ 命中
耗时 ⏱ 0.13s
```

---

## 图片预留

### 图片 1：项目主页 / Releases 页面

![项目主页截图]()

### 图片 2：Windows 解压后的目录结构

![Windows 目录截图]()

### 图片 3：填写 `.env` 示例

![env 示例截图]()

### 图片 4：实际运行结果

![运行结果截图]()

---

## 一些说明

- 这是一个个人整理和分享的小工具
- 目前主打的是“轻量、够用、易分发”
- 公开仓库里同时保留了 Go 主实现和 Python 参考实现
- 如果后续 WebVPN 路由策略变化，可能需要跟着调整

---

## 欢迎反馈

如果你在使用过程中遇到问题，或者想提一些改进建议，欢迎直接提 issue 或留言交流。

如果这个工具对你有帮助，也欢迎点个 star。
