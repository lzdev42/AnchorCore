# LibAnchor API 使用文档

本文档基于 `Anchorcore.framework`（通过 `gomobile` 从 `github.com/sagernet/sing-box` 编译），提供在 iOS/macOS 应用中集成 VPN 核心的完整指南。

## 目录

- [概述](#概述)
- [核心概念](#核心概念)
- [快速开始](#快速开始)
- [功能详解](#功能详解)
  - [1. 启动与停止服务](#1-启动与停止服务)
  - [2. 路由模式切换](#2-路由模式切换)
  - [3. 流量统计](#3-流量统计)
  - [4. 节点管理](#4-节点管理)
  - [5. 实时日志](#5-实时日志)
  - [6. 连接管理](#6-连接管理)
  - [7. 配置管理](#7-配置管理)
- [完整示例](#完整示例)
- [常见问题](#常见问题)

---

## 概述

LibAnchor 采用 **Service-Client 架构**：
- **`AnchorcoreBoxService`**：VPN 核心服务，负责流量转发和生命周期管理
- **`AnchorcoreCommandClient`**：控制客户端，用于与运行中的服务通信（获取数据、发送指令）

## 核心概念

### 路由模式（Clash Mode）

LibAnchor 支持三种路由模式，通过 `setClashMode` 动态切换：

| 模式 | 说明 | 行为 |
|------|------|------|
| **Rule** | 规则分流模式 | 按照配置文件中的规则自动分流（默认） |
| **Global** | 全局代理模式 | 所有流量强制走代理，忽略规则 |
| **Direct** | 全局直连模式 | 所有流量强制直连，不走代理 |

**实现原理**：在配置文件的 `route.rules` 中添加 `clash_mode` 字段的规则，运行时切换模式会改变规则匹配行为。

---

## 快速开始

### 1. 配置文件示例

```json
{
  "log": {
    "level": "info",
    "timestamp": true
  },
  "dns": {
    "servers": [
      {
        "tag": "google",
        "address": "https://8.8.8.8/dns-query"
      }
    ]
  },
  "route": {
    "rules": [
      {
        "clash_mode": "Global",
        "outbound": "proxy"
      },
      {
        "clash_mode": "Direct",
        "outbound": "direct"
      },
      {
        "geoip": "cn",
        "outbound": "direct"
      },
      {
        "domain_suffix": [".cn"],
        "outbound": "direct"
      }
    ],
    "final": "proxy"
  },
  "outbounds": [
    {
      "type": "vmess",
      "tag": "proxy",
      "server": "example.com",
      "server_port": 443,
      "uuid": "your-uuid-here",
      "security": "auto"
    },
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}
```

### 2. 初始化代码

```swift
import Anchorcore

class VPNManager {
    var service: AnchorcoreBoxService?
    var commandClient: AnchorcoreCommandClient?
    let clientHandler = ClientHandler()
    
    func initialize() {
        // 1. 启动服务
        startVPN(configContent: loadConfigJSON())
        
        // 2. 连接控制客户端
        setupCommandClient()
    }
}
```

---

## 功能详解

### 1. 启动与停止服务

#### 启动服务

```swift
func startVPN(configContent: String) {
    var error: NSError?
    
    // 创建服务实例
    guard let service = AnchorcoreNewService(
        configContent,          // JSON 配置字符串
        nil,                    // platformInterface (通常传 nil)
        &error
    ) else {
        print("创建服务失败: \(error?.localizedDescription ?? "未知错误")")
        return
    }
    self.service = service
    
    // 启动服务
    if !service.start(&error) {
        print("启动失败: \(error?.localizedDescription ?? "未知错误")")
        return
    }
    
    print("VPN 启动成功")
}
```

#### 停止服务

```swift
func stopVPN() {
    var error: NSError?
    service?.close(&error)
    service = nil
    print("VPN 已停止")
}
```

#### 重启服务

```swift
func restartVPN(newConfig: String) {
    stopVPN()
    Thread.sleep(forTimeInterval: 0.5) // 可选：等待资源释放
    startVPN(configContent: newConfig)
}
```

---

### 2. 路由模式切换

#### 设置控制客户端

```swift
class ClientHandler: NSObject, AnchorcoreCommandClientHandler {
    var onModeChanged: ((String) -> Void)?
    var supportedModes: [String] = []
    
    func connected() {
        print("✅ 已连接到核心")
    }
    
    func disconnected(_ message: String?) {
        print("❌ 断开连接: \(message ?? "")")
    }
    
    // 接收模式列表
    func initializeClashMode(_ modeList: AnchorcoreStringIterator?, currentMode: String?) {
        supportedModes.removeAll()
        while modeList?.hasNext() == true {
            if let mode = modeList?.next() {
                supportedModes.append(mode)
                print("支持模式: \(mode)")
            }
        }
        print("当前模式: \(currentMode ?? "未知")")
        if let mode = currentMode {
            onModeChanged?(mode)
        }
    }
    
    // 模式变更通知
    func updateClashMode(_ newMode: String?) {
        guard let mode = newMode else { return }
        print("🔄 模式已切换为: \(mode)")
        onModeChanged?(mode)
    }
    
    // 其他必需的协议方法（空实现）
    func clearLogs() {}
    func writeConnections(_ message: AnchorcoreConnections?) {}
    func writeGroups(_ message: AnchorcoreOutboundGroupIterator?) {}
    func writeLogs(_ messageList: AnchorcoreStringIterator?) {}
    func writeStatus(_ message: AnchorcoreStatusMessage?) {}
}
```

#### 切换模式

```swift
func setupCommandClient() {
    let options = AnchorcoreCommandClientOptions()
    options.command = AnchorcoreCommandClashMode // 订阅模式变更
    options.statusInterval = 0
    
    commandClient = AnchorcoreNewCommandClient(clientHandler, options)
    
    var error: NSError?
    if commandClient?.connect(&error) == true {
        print("控制客户端连接成功")
    } else {
        print("连接失败: \(error?.localizedDescription ?? "")")
    }
}

// 切换到规则模式
func switchToRuleMode() {
    var error: NSError?
    commandClient?.setClashMode("Rule", error: &error)
}

// 切换到全局代理模式
func switchToGlobalMode() {
    var error: NSError?
    commandClient?.setClashMode("Global", error: &error)
}

// 切换到全局直连模式
func switchToDirectMode() {
    var error: NSError?
    commandClient?.setClashMode("Direct", error: &error)
}
```

---

### 3. 流量统计

#### 订阅流量数据

```swift
func setupCommandClient() {
    let options = AnchorcoreCommandClientOptions()
    options.command = AnchorcoreCommandStatus  // 订阅状态更新
    options.statusInterval = 1000              // 每 1000ms 更新一次
    
    commandClient = AnchorcoreNewCommandClient(clientHandler, options)
    // ... 连接逻辑
}
```

#### 接收流量回调

```swift
class ClientHandler: NSObject, AnchorcoreCommandClientHandler {
    var onTrafficUpdate: ((Int64, Int64, Int64, Int64) -> Void)?
    
    func writeStatus(_ message: AnchorcoreStatusMessage?) {
        guard let msg = message else { return }
        
        let upSpeed = msg.uplink          // 上传速度 (bytes/s)
        let downSpeed = msg.downlink      // 下载速度 (bytes/s)
        let totalUp = msg.uplinkTotal     // 总上传 (bytes)
        let totalDown = msg.downlinkTotal // 总下载 (bytes)
        
        onTrafficUpdate?(upSpeed, downSpeed, totalUp, totalDown)
        
        // 格式化输出
        let upStr = AnchorcoreFormatBytes(upSpeed)
        let downStr = AnchorcoreFormatBytes(downSpeed)
        print("↑ \(upStr)/s  ↓ \(downStr)/s")
    }
}
```

#### 格式化工具函数

```swift
// 使用 LibAnchor 提供的格式化函数
let formattedSpeed = AnchorcoreFormatBytes(1024000) // "1.00 MB"
let formattedMemory = AnchorcoreFormatMemoryBytes(512000000) // "512 MB"
```

---

### 4. 节点管理

#### 订阅节点分组

```swift
func setupCommandClient() {
    let options = AnchorcoreCommandClientOptions()
    options.command = AnchorcoreCommandGroup  // 订阅分组更新
    
    commandClient = AnchorcoreNewCommandClient(clientHandler, options)
    // ... 连接逻辑
}
```

#### 接收分组数据

```swift
class ClientHandler: NSObject, AnchorcoreCommandClientHandler {
    var onGroupsUpdate: (([OutboundGroupModel]) -> Void)?
    
    func writeGroups(_ iterator: AnchorcoreOutboundGroupIterator?) {
        var groups: [OutboundGroupModel] = []
        
        while iterator?.hasNext() == true {
            guard let group = iterator?.next() else { continue }
            
            var items: [OutboundItemModel] = []
            let itemsIterator = group.getItems()
            while itemsIterator?.hasNext() == true {
                guard let item = itemsIterator?.next() else { continue }
                items.append(OutboundItemModel(
                    tag: item.tag,
                    type: item.type,
                    delay: Int(item.urlTestDelay)
                ))
            }
            
            groups.append(OutboundGroupModel(
                tag: group.tag,
                type: group.type,
                selected: group.selected,
                items: items
            ))
        }
        
        onGroupsUpdate?(groups)
    }
}

// 数据模型
struct OutboundGroupModel {
    let tag: String
    let type: String
    let selected: String
    let items: [OutboundItemModel]
}

struct OutboundItemModel {
    let tag: String
    let type: String
    let delay: Int
}
```

#### 选择节点

```swift
func selectNode(groupTag: String, nodeTag: String) {
    var error: NSError?
    commandClient?.selectOutbound(groupTag, outboundTag: nodeTag, error: &error)
    if let err = error {
        print("选择节点失败: \(err.localizedDescription)")
    }
}

// 示例：切换 "Proxy" 组到 "US-Server-1" 节点
selectNode(groupTag: "Proxy", nodeTag: "US-Server-1")
```

#### 延迟测试

```swift
func testGroupLatency(groupTag: String) {
    var error: NSError?
    commandClient?.urlTest(groupTag, error: &error)
    // 测试结果会通过 writeGroups 回调返回，查看 urlTestDelay 字段
}
```

---

### 5. 实时日志

#### 订阅日志

```swift
func setupCommandClient() {
    let options = AnchorcoreCommandClientOptions()
    options.command = AnchorcoreCommandLog  // 订阅日志
    
    commandClient = AnchorcoreNewCommandClient(clientHandler, options)
    // ... 连接逻辑
}
```

#### 接收日志

```swift
class ClientHandler: NSObject, AnchorcoreCommandClientHandler {
    var onLogReceived: (([String]) -> Void)?
    
    func writeLogs(_ iterator: AnchorcoreStringIterator?) {
        var logs: [String] = []
        while iterator?.hasNext() == true {
            if let log = iterator?.next() {
                logs.append(log)
            }
        }
        onLogReceived?(logs)
    }
    
    func clearLogs() {
        // UI 清空日志时调用
        print("日志已清空")
    }
}
```

---

### 6. 连接管理

#### 订阅连接信息

```swift
func setupCommandClient() {
    let options = AnchorcoreCommandClientOptions()
    options.command = AnchorcoreCommandConnections  // 订阅连接列表
    
    commandClient = AnchorcoreNewCommandClient(clientHandler, options)
    // ... 连接逻辑
}
```

#### 接收连接列表

```swift
class ClientHandler: NSObject, AnchorcoreCommandClientHandler {
    var onConnectionsUpdate: (([ConnectionModel]) -> Void)?
    
    func writeConnections(_ connections: AnchorcoreConnections?) {
        var connList: [ConnectionModel] = []
        
        let iterator = connections?.iterator()
        while iterator?.hasNext() == true {
            guard let conn = iterator?.next() else { continue }
            connList.append(ConnectionModel(
                id: conn.id_,
                source: conn.source,
                destination: conn.displayDestination(),
                protocol: conn.protocol,
                rule: conn.rule,
                chain: conn.chain(),
                uplink: conn.uplink,
                downlink: conn.downlink
            ))
        }
        
        onConnectionsUpdate?(connList)
    }
}

struct ConnectionModel {
    let id: String
    let source: String
    let destination: String
    let protocol: String
    let rule: String
    let chain: AnchorcoreStringIterator?
    let uplink: Int64
    let downlink: Int64
}
```

#### 断开连接

```swift
// 断开指定连接
func closeConnection(id: String) {
    var error: NSError?
    commandClient?.closeConnection(id, error: &error)
}

// 断开所有连接
func closeAllConnections() {
    var error: NSError?
    commandClient?.closeConnections(&error)
}
```

---

### 7. 配置管理

#### 验证配置

```swift
func validateConfig(_ jsonString: String) -> Bool {
    var error: NSError?
    let isValid = AnchorcoreCheckConfig(jsonString, &error)
    if !isValid {
        print("配置验证失败: \(error?.localizedDescription ?? "")")
    }
    return isValid
}
```

#### 格式化配置

```swift
func formatConfig(_ jsonString: String) -> String? {
    var error: NSError?
    let formatted = AnchorcoreFormatConfig(jsonString, &error)
    return formatted?.value
}
```

#### 应用新配置

```swift
func applyNewConfig(_ newConfigJSON: String) {
    // 1. 验证配置
    guard validateConfig(newConfigJSON) else {
        print("配置无效，拒绝应用")
        return
    }
    
    // 2. 断开客户端
    var error: NSError?
    commandClient?.disconnect(&error)
    
    // 3. 重启服务
    stopVPN()
    Thread.sleep(forTimeInterval: 0.5)
    startVPN(configContent: newConfigJSON)
    
    // 4. 重新连接客户端
    Thread.sleep(forTimeInterval: 1.0)
    setupCommandClient()
}
```

---

## 完整示例

```swift
import Anchorcore

class VPNManager: NSObject {
    static let shared = VPNManager()
    
    var service: AnchorcoreBoxService?
    var commandClient: AnchorcoreCommandClient?
    let clientHandler = ClientHandler()
    
    private override init() {
        super.init()
        setupCallbacks()
    }
    
    func start() {
        let config = loadConfigFromFile()
        startVPN(configContent: config)
        setupCommandClient()
    }
    
    private func startVPN(configContent: String) {
        var error: NSError?
        guard let service = AnchorcoreNewService(configContent, nil, &error) else {
            print("创建服务失败")
            return
        }
        self.service = service
        
        if !service.start(&error) {
            print("启动失败")
            return
        }
    }
    
    private func setupCommandClient() {
        let options = AnchorcoreCommandClientOptions()
        options.command = AnchorcoreCommandStatus 
                        | AnchorcoreCommandGroup 
                        | AnchorcoreCommandLog 
                        | AnchorcoreCommandConnections
                        | AnchorcoreCommandClashMode
        options.statusInterval = 1000
        
        commandClient = AnchorcoreNewCommandClient(clientHandler, options)
        
        var error: NSError?
        commandClient?.connect(&error)
    }
    
    private func setupCallbacks() {
        clientHandler.onTrafficUpdate = { up, down, totalUp, totalDown in
            NotificationCenter.default.post(
                name: .trafficUpdated,
                object: nil,
                userInfo: ["up": up, "down": down]
            )
        }
        
        clientHandler.onModeChanged = { mode in
            NotificationCenter.default.post(
                name: .modeChanged,
                object: mode
            )
        }
    }
    
    func switchMode(to mode: String) {
        var error: NSError?
        commandClient?.setClashMode(mode, error: &error)
    }
    
    func stop() {
        var error: NSError?
        commandClient?.disconnect(&error)
        service?.close(&error)
        service = nil
    }
    
    private func loadConfigFromFile() -> String {
        // 从 Bundle 或文件系统加载配置
        return """
        {
          "route": {
            "rules": [
              {"clash_mode": "Global", "outbound": "proxy"},
              {"clash_mode": "Direct", "outbound": "direct"}
            ]
          },
          "outbounds": [
            {"type": "direct", "tag": "direct"}
          ]
        }
        """
    }
}

// 通知名称扩展
extension Notification.Name {
    static let trafficUpdated = Notification.Name("trafficUpdated")
    static let modeChanged = Notification.Name("modeChanged")
}
```

---

## 常见问题

### Q1: 如何处理服务崩溃？
```swift
// 监听服务错误
var errorMessage = AnchorcoreReadServiceError(&error)
if let err = errorMessage {
    print("服务错误: \(err.value)")
}
```

### Q2: 如何在后台保持连接？
```swift
// 使用 pause/wake 机制
func applicationDidEnterBackground() {
    service?.pause()
}

func applicationWillEnterForeground() {
    service?.wake()
}
```

### Q3: 模式切换后如何确认生效？
通过 `ClientHandler.updateClashMode` 回调确认：
```swift
func updateClashMode(_ newMode: String?) {
    print("模式已切换: \(newMode ?? "")")
    // 更新 UI 状态
}
```

### Q4: 如何获取版本信息？
```swift
let version = AnchorcoreVersion()
print("LibAnchor 版本: \(version)")
```

### Q5: CommandClient 断线重连？
```swift
func reconnectClient() {
    var error: NSError?
    commandClient?.disconnect(&error)
    Thread.sleep(forTimeInterval: 1.0)
    commandClient?.connect(&error)
}
```

---

## API 参考表

| 功能 | API | 说明 |
|------|-----|------|
| **服务管理** | `AnchorcoreNewService` | 创建服务实例 |
| | `service.start()` | 启动服务 |
| | `service.close()` | 停止服务 |
| | `service.pause()` | 暂停（后台） |
| | `service.wake()` | 唤醒 |
| **模式切换** | `client.setClashMode()` | 切换路由模式 |
| **节点管理** | `client.selectOutbound()` | 选择节点 |
| | `client.urlTest()` | 延迟测试 |
| **连接管理** | `client.closeConnection()` | 断开单个连接 |
| | `client.closeConnections()` | 断开所有连接 |
| **配置** | `AnchorcoreCheckConfig()` | 验证配置 |
| | `AnchorcoreFormatConfig()` | 格式化配置 |
| **工具函数** | `AnchorcoreFormatBytes()` | 格式化流量 |
| | `AnchorcoreVersion()` | 获取版本 |

---

**文档版本**: v1.0  
**更新日期**: 2025-11-21  
**适用版本**: sing-box 1.8.0+
