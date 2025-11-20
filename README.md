# AnchorCore

一个基于 [sing-box](https://github.com/SagerNet/sing-box) 的修改版本，针对自用需求进行了简化和定制。

## 构建产物

- **命令行工具**：`anchor` (Linux/Windows, amd64/arm64)
- **Android 库**：`anchorcore.aar`
- **Apple 库**：`anchorcore.xcframework` (iOS/macOS)

## 主要修改

- 重命名为 AnchorCore/anchor
- 精简构建目标（仅保留 Linux、Windows、Android、iOS/macOS）
- 移除不需要的平台和打包格式

## 协议

本项目遵循 GPLv3 协议，详见上游项目 [sing-box](https://github.com/SagerNet/sing-box) 的协议说明。