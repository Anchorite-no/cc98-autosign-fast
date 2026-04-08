# Python 参考实现

这里保留的是当前协议验证后的 Python 轻量实现，主要用于：

- 对照 Go 版逻辑
- 协议调试
- 临时排查问题

普通用户不建议优先使用这里的脚本，优先使用仓库 release 中的 Go 成品。

## 文件说明

- `webvpn-fixed-api.py`：Python 主脚本
- `requirements.txt`：依赖列表
- `.env.example`：最小配置模板
- `run-webvpn-fixed-api.ps1` / `run-webvpn-fixed-api.sh`：本地启动脚本
