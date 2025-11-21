# Clash API 使用文档

本文档详细说明如何通过 **Clash RESTful API** 控制 AnchorCore 命令行版本（`anchor run`），实现远程管理和实时监控。

## 目录

- [概述](#概述)
- [配置启用](#配置启用)
- [认证方式](#认证方式)
- [API 端点详解](#api-端点详解)
  - [1. 系统信息](#1-系统信息)
  - [2. 路由模式切换](#2-路由模式切换)
  - [3. 流量统计](#3-流量统计)
  - [4. 节点管理](#4-节点管理)
  - [5. 实时日志](#5-实时日志)
  - [6. 连接管理](#6-连接管理)
  - [7. 延迟测试](#7-延迟测试)
- [完整示例](#完整示例)
- [错误处理](#错误处理)

---

## 概述

Clash API 是一个兼容 Clash Premium/Meta 的 RESTful API，运行在独立的 HTTP 服务器上。所有端点返回 JSON 格式数据。

**特性**：
- ✅ 完全兼容 Clash Dashboard
- ✅ 支持 WebSocket 实时推送
- ✅ CORS 跨域支持
- ✅ Bearer Token 认证

---

## 配置启用

在配置文件中添加 `experimental.clash_api` 字段：

```json
{
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:9090",
      "external_ui": "/path/to/clash-dashboard",
      "external_ui_download_url": "https://github.com/MetaCubeX/metacubexd/archive/refs/heads/gh-pages.zip",
      "external_ui_download_detour": "proxy",
      "secret": "your-secret-token",
      "default_mode": "Rule"
    }
  }
}
```

**参数说明**：
- `external_controller`：API 监听地址（必填）
- `secret`：API 访问密钥（可选，强烈建议设置）
- `default_mode`：默认路由模式（`Rule`/`Global`/`Direct`）
- `external_ui`：Web Dashboard 路径（可选）

---

## 认证方式

所有请求需要在 HTTP Header 中携带 Token（如果配置了 `secret`）：

```bash
Authorization: Bearer your-secret-token
```

**示例**：
```bash
curl -H "Authorization: Bearer your-secret-token" \
     http://127.0.0.1:9090/version
```

---

## API 端点详解

### 1. 系统信息

#### 获取版本信息
```http
GET /version
```

**响应**：
```json
{
  "version": "sing-box 1.8.0",
  "premium": true,
  "meta": true
}
```

**示例**：
```bash
curl http://127.0.0.1:9090/version
```

---

### 2. 路由模式切换

#### 获取当前配置
```http
GET /configs
```

**响应**：
```json
{
  "mode": "Rule",
  "mode-list": ["Rule", "Global", "Direct"],
  "log-level": "info"
}
```

#### 切换路由模式
```http
PATCH /configs
Content-Type: application/json
```

**请求体**：
```json
{
  "mode": "Global"
}
```

**响应**：`204 No Content`

**模式说明**：
| 模式 | 行为 |
|------|------|
| `Rule` | 按规则分流 |
| `Global` | 全局代理 |
| `Direct` | 全局直连 |

**示例**：
```bash
# 切换到全局代理模式
curl -X PATCH http://127.0.0.1:9090/configs \
     -H "Content-Type: application/json" \
     -d '{"mode":"Global"}'

# 切换到规则模式
curl -X PATCH http://127.0.0.1:9090/configs \
     -H "Content-Type: application/json" \
     -d '{"mode":"Rule"}'

# 切换到全局直连模式
curl -X PATCH http://127.0.0.1:9090/configs \
     -H "Content-Type: application/json" \
     -d '{"mode":"Direct"}'
```

---

### 3. 流量统计

#### 实时流量（WebSocket）
```http
GET /traffic
Upgrade: websocket
```

**响应格式**（每秒推送一次）：
```json
{
  "up": 12800,
  "down": 204800
}
```

**说明**：
- `up`：上传速度（bytes/s）
- `down`：下载速度（bytes/s）

**示例（JavaScript）**：
```javascript
const ws = new WebSocket('ws://127.0.0.1:9090/traffic');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`↑ ${data.up} B/s  ↓ ${data.down} B/s`);
};
```

**示例（Python）**：
```python
import websocket
import json

def on_message(ws, message):
    data = json.loads(message)
    print(f"↑ {data['up']} B/s  ↓ {data['down']} B/s")

ws = websocket.WebSocketApp(
    "ws://127.0.0.1:9090/traffic",
    on_message=on_message
)
ws.run_forever()
```

#### 内存占用（WebSocket）
```http
GET /memory
Upgrade: websocket
```

**响应格式**：
```json
{
  "inuse": 45678912,
  "oslimit": 0
}
```

---

### 4. 节点管理

#### 获取所有节点/分组
```http
GET /proxies
```

**响应示例**：
```json
{
  "proxies": {
    "GLOBAL": {
      "type": "Fallback",
      "name": "GLOBAL",
      "all": ["Proxy", "US", "HK"],
      "now": "Proxy"
    },
    "Proxy": {
      "type": "Selector",
      "name": "Proxy",
      "all": ["US-1", "US-2", "HK-1"],
      "now": "US-1"
    },
    "US-1": {
      "type": "VMess",
      "name": "US-1",
      "udp": true,
      "history": [
        {"time": "2025-11-21T19:00:00Z", "delay": 120}
      ]
    }
  }
}
```

#### 获取单个节点/分组信息
```http
GET /proxies/{name}
```

**响应示例**：
```json
{
  "type": "Selector",
  "name": "Proxy",
  "all": ["US-1", "US-2", "HK-1"],
  "now": "US-1",
  "udp": true,
  "history": []
}
```

#### 切换节点（仅 Selector 分组）
```http
PUT /proxies/{groupName}
Content-Type: application/json
```

**请求体**：
```json
{
  "name": "US-2"
}
```

**响应**：`204 No Content`

**示例**：
```bash
# 将 "Proxy" 组切换到 "US-2" 节点
curl -X PUT http://127.0.0.1:9090/proxies/Proxy \
     -H "Content-Type: application/json" \
     -d '{"name":"US-2"}'
```

#### 获取分组列表（Meta API）
```http
GET /group
```

**响应示例**：
```json
{
  "proxies": [
    {
      "type": "Selector",
      "name": "Proxy",
      "all": ["US-1", "US-2"],
      "now": "US-1"
    }
  ]
}
```

---

### 5. 实时日志

#### 订阅日志（WebSocket）
```http
GET /logs?level=info
Upgrade: websocket
```

**URL 参数**：
- `level`：日志级别（`trace`/`debug`/`info`/`warn`/`error`），默认 `info`

**响应格式**：
```json
{
  "type": "info",
  "payload": "[TCP] 192.168.1.100:54321 --> www.google.com:443"
}
```

**示例（JavaScript）**：
```javascript
const ws = new WebSocket('ws://127.0.0.1:9090/logs?level=info');
ws.onmessage = (event) => {
  const log = JSON.parse(event.data);
  console.log(`[${log.type}] ${log.payload}`);
};
```

---

### 6. 连接管理

#### 获取所有连接（快照）
```http
GET /connections
```

**响应示例**：
```json
{
  "downloadTotal": 1048576000,
  "uploadTotal": 524288000,
  "connections": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "metadata": {
        "network": "tcp",
        "type": "HTTP",
        "sourceIP": "192.168.1.100",
        "destinationIP": "142.250.185.46",
        "sourcePort": "54321",
        "destinationPort": "443",
        "host": "www.google.com",
        "dnsMode": "normal",
        "processPath": "/usr/bin/chrome"
      },
      "upload": 12800,
      "download": 102400,
      "start": "2025-11-21T19:00:00Z",
      "chains": ["Proxy", "US-1"],
      "rule": "DOMAIN-SUFFIX",
      "rulePayload": "google.com"
    }
  ]
}
```

#### 实时连接（WebSocket）
```http
GET /connections
Upgrade: websocket
Connection: Upgrade
```

**URL 参数**：
- `interval`：推送间隔（毫秒），默认 `1000`

**示例**：
```bash
wscat -c "ws://127.0.0.1:9090/connections?interval=500"
```

#### 断开单个连接
```http
DELETE /connections/{id}
```

**响应**：`204 No Content`

**示例**：
```bash
curl -X DELETE http://127.0.0.1:9090/connections/550e8400-e29b-41d4-a716-446655440000
```

#### 断开所有连接
```http
DELETE /connections
```

**响应**：`204 No Content`

**示例**：
```bash
curl -X DELETE http://127.0.0.1:9090/connections
```

---

### 7. 延迟测试

#### 测试单个节点延迟
```http
GET /proxies/{name}/delay?url={url}&timeout={timeout}
```

**URL 参数**：
- `url`：测试 URL（可选，默认使用配置的 URL）
- `timeout`：超时时间（毫秒），例如 `5000`

**响应示例**：
```json
{
  "delay": 120
}
```

**错误响应**：
- `504 Gateway Timeout`：测试超时
- `503 Service Unavailable`：节点不可用

**示例**：
```bash
curl "http://127.0.0.1:9090/proxies/US-1/delay?url=https://www.gstatic.com/generate_204&timeout=5000"
```

#### 测试分组所有节点延迟
```http
GET /group/{name}/delay?url={url}&timeout={timeout}
```

**响应示例**：
```json
{
  "US-1": 120,
  "US-2": 150,
  "HK-1": 80
}
```

**示例**：
```bash
curl "http://127.0.0.1:9090/group/Proxy/delay?url=https://www.gstatic.com/generate_204&timeout=5000"
```

---

## 完整示例

### Kotlin 客户端实现

#### 1. 依赖配置（build.gradle.kts）

```kotlin
dependencies {
    // OkHttp for HTTP requests
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    
    // Gson for JSON parsing
    implementation("com.google.code.gson:gson:2.10.1")
    
    // Coroutines (可选，用于异步)
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.7.3")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.7.3")
}
```

#### 2. 数据模型

```kotlin
data class VersionResponse(
    val version: String,
    val premium: Boolean,
    val meta: Boolean
)

data class ConfigResponse(
    val mode: String,
    @SerializedName("mode-list") val modeList: List<String>,
    @SerializedName("log-level") val logLevel: String
)

data class ProxyInfo(
    val type: String,
    val name: String,
    val udp: Boolean,
    val all: List<String>? = null,
    val now: String? = null,
    val history: List<URLTestHistory>? = null
)

data class URLTestHistory(
    val time: String,
    val delay: Int
)

data class ProxiesResponse(
    val proxies: Map<String, ProxyInfo>
)

data class ConnectionMetadata(
    val network: String,
    val type: String,
    val sourceIP: String,
    val destinationIP: String,
    val sourcePort: String,
    val destinationPort: String,
    val host: String? = null
)

data class Connection(
    val id: String,
    val metadata: ConnectionMetadata,
    val upload: Long,
    val download: Long,
    val start: String,
    val chains: List<String>,
    val rule: String,
    val rulePayload: String
)

data class ConnectionsResponse(
    val downloadTotal: Long,
    val uploadTotal: Long,
    val connections: List<Connection>
)

data class TrafficData(
    val up: Long,
    val down: Long
)

data class DelayResponse(
    val delay: Int
)
```

#### 3. API 客户端

```kotlin
import com.google.gson.Gson
import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException

class AnchorClashAPI(
    private val baseUrl: String = "http://127.0.0.1:9090",
    private val token: String? = null
) {
    private val client = OkHttpClient()
    private val gson = Gson()
    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    private fun buildRequest(url: String): Request.Builder {
        return Request.Builder().url(url).apply {
            token?.let { addHeader("Authorization", "Bearer $it") }
        }
    }

    // ========== 同步方法 ==========
    
    /**
     * 获取版本信息
     */
    fun getVersion(): VersionResponse? {
        val request = buildRequest("$baseUrl/version").build()
        return executeRequest(request)
    }

    /**
     * 获取配置信息
     */
    fun getConfig(): ConfigResponse? {
        val request = buildRequest("$baseUrl/configs").build()
        return executeRequest(request)
    }

    /**
     * 获取当前模式
     */
    fun getCurrentMode(): String? {
        return getConfig()?.mode
    }

    /**
     * 切换路由模式
     * @param mode "Rule" / "Global" / "Direct"
     */
    fun setMode(mode: String): Boolean {
        val json = gson.toJson(mapOf("mode" to mode))
        val body = json.toRequestBody(jsonMediaType)
        val request = buildRequest("$baseUrl/configs")
            .patch(body)
            .build()
        
        client.newCall(request).execute().use { response ->
            return response.isSuccessful
        }
    }

    /**
     * 获取所有节点/分组
     */
    fun getProxies(): Map<String, ProxyInfo>? {
        val request = buildRequest("$baseUrl/proxies").build()
        val response: ProxiesResponse? = executeRequest(request)
        return response?.proxies
    }

    /**
     * 获取单个节点信息
     */
    fun getProxy(name: String): ProxyInfo? {
        val request = buildRequest("$baseUrl/proxies/$name").build()
        return executeRequest(request)
    }

    /**
     * 选择节点（仅适用于 Selector 分组）
     * @param group 分组名称
     * @param node 节点名称
     */
    fun selectNode(group: String, node: String): Boolean {
        val json = gson.toJson(mapOf("name" to node))
        val body = json.toRequestBody(jsonMediaType)
        val request = buildRequest("$baseUrl/proxies/$group")
            .put(body)
            .build()
        
        client.newCall(request).execute().use { response ->
            return response.isSuccessful
        }
    }

    /**
     * 测试节点延迟
     * @param name 节点名称
     * @param timeout 超时时间（毫秒）
     */
    fun testDelay(name: String, timeout: Int = 5000): Int? {
        val url = "$baseUrl/proxies/$name/delay?timeout=$timeout"
        val request = buildRequest(url).build()
        val response: DelayResponse? = executeRequest(request)
        return response?.delay
    }

    /**
     * 获取所有连接
     */
    fun getConnections(): ConnectionsResponse? {
        val request = buildRequest("$baseUrl/connections").build()
        return executeRequest(request)
    }

    /**
     * 断开单个连接
     */
    fun closeConnection(id: String): Boolean {
        val request = buildRequest("$baseUrl/connections/$id")
            .delete()
            .build()
        
        client.newCall(request).execute().use { response ->
            return response.isSuccessful
        }
    }

    /**
     * 断开所有连接
     */
    fun closeAllConnections(): Boolean {
        val request = buildRequest("$baseUrl/connections")
            .delete()
            .build()
        
        client.newCall(request).execute().use { response ->
            return response.isSuccessful
        }
    }

    // ========== 通用请求执行方法 ==========
    
    private inline fun <reified T> executeRequest(request: Request): T? {
        return try {
            client.newCall(request).execute().use { response ->
                if (response.isSuccessful) {
                    response.body?.string()?.let {
                        gson.fromJson(it, T::class.java)
                    }
                } else {
                    null
                }
            }
        } catch (e: IOException) {
            e.printStackTrace()
            null
        }
    }

    // ========== WebSocket 实时监听 ==========
    
    /**
     * 监听实时流量
     */
    fun watchTraffic(callback: (TrafficData) -> Unit): WebSocket {
        val wsUrl = baseUrl.replace("http://", "ws://") + "/traffic"
        val request = Request.Builder().url(wsUrl).apply {
            token?.let { addHeader("Authorization", "Bearer $it") }
        }.build()

        return client.newWebSocket(request, object : WebSocketListener() {
            override fun onMessage(webSocket: WebSocket, text: String) {
                try {
                    val data = gson.fromJson(text, TrafficData::class.java)
                    callback(data)
                } catch (e: Exception) {
                    e.printStackTrace()
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                t.printStackTrace()
            }
        })
    }

    /**
     * 监听实时日志
     */
    fun watchLogs(level: String = "info", callback: (String, String) -> Unit): WebSocket {
        val wsUrl = baseUrl.replace("http://", "ws://") + "/logs?level=$level"
        val request = Request.Builder().url(wsUrl).apply {
            token?.let { addHeader("Authorization", "Bearer $it") }
        }.build()

        return client.newWebSocket(request, object : WebSocketListener() {
            override fun onMessage(webSocket: WebSocket, text: String) {
                try {
                    val json = gson.fromJson(text, Map::class.java)
                    val type = json["type"] as? String ?: ""
                    val payload = json["payload"] as? String ?: ""
                    callback(type, payload)
                } catch (e: Exception) {
                    e.printStackTrace()
                }
            }
        })
    }

    /**
     * 监听实时连接
     */
    fun watchConnections(
        interval: Int = 1000,
        callback: (ConnectionsResponse) -> Unit
    ): WebSocket {
        val wsUrl = baseUrl.replace("http://", "ws://") + 
                    "/connections?interval=$interval"
        val request = Request.Builder().url(wsUrl).apply {
            token?.let { addHeader("Authorization", "Bearer $it") }
        }.build()

        return client.newWebSocket(request, object : WebSocketListener() {
            override fun onMessage(webSocket: WebSocket, text: String) {
                try {
                    val data = gson.fromJson(text, ConnectionsResponse::class.java)
                    callback(data)
                } catch (e: Exception) {
                    e.printStackTrace()
                }
            }
        })
    }
}
```

#### 4. 使用示例

```kotlin
import kotlinx.coroutines.*

fun main() = runBlocking {
    val api = AnchorClashAPI(
        baseUrl = "http://127.0.0.1:9090",
        token = "your-secret-token"
    )

    // 1. 获取版本信息
    println("=== 版本信息 ===")
    api.getVersion()?.let { version ->
        println("版本: ${version.version}")
        println("Premium: ${version.premium}")
    }

    // 2. 获取当前模式
    println("\n=== 当前模式 ===")
    val currentMode = api.getCurrentMode()
    println("当前模式: $currentMode")

    // 3. 切换到全局代理模式
    println("\n=== 切换模式 ===")
    if (api.setMode("Global")) {
        println("✓ 已切换到全局代理模式")
    }

    // 4. 获取节点列表
    println("\n=== 节点列表 ===")
    api.getProxies()?.forEach { (name, info) ->
        println("节点: $name (${info.type})")
        info.now?.let { println("  当前选中: $it") }
    }

    // 5. 切换节点
    println("\n=== 切换节点 ===")
    if (api.selectNode("Proxy", "US-1")) {
        println("✓ 已切换到 US-1")
    }

    // 6. 测试延迟
    println("\n=== 测试延迟 ===")
    api.testDelay("US-1")?.let { delay ->
        println("US-1 延迟: ${delay}ms")
    }

    // 7. 获取连接列表
    println("\n=== 当前连接 ===")
    api.getConnections()?.let { data ->
        println("连接数: ${data.connections.size}")
        println("总上传: ${data.uploadTotal} bytes")
        println("总下载: ${data.downloadTotal} bytes")
    }

    // 8. 实时流量监控
    println("\n=== 开始监听实时流量 ===")
    val trafficWs = api.watchTraffic { traffic ->
        println("↑ ${traffic.up} B/s  ↓ ${traffic.down} B/s")
    }

    // 9. 实时日志监听
    println("\n=== 开始监听日志 ===")
    val logWs = api.watchLogs(level = "info") { type, payload ->
        println("[$type] $payload")
    }

    // 保持运行 10 秒后关闭
    delay(10000)
    trafficWs.close(1000, "Done")
    logWs.close(1000, "Done")
}
```

#### 5. Android ViewModel 集成示例

```kotlin
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.WebSocket

class VPNViewModel : ViewModel() {
    private val api = AnchorClashAPI(
        baseUrl = "http://127.0.0.1:9090",
        token = "your-secret-token"
    )

    private val _currentMode = MutableStateFlow("Rule")
    val currentMode: StateFlow<String> = _currentMode

    private val _trafficData = MutableStateFlow(TrafficData(0, 0))
    val trafficData: StateFlow<TrafficData> = _trafficData

    private val _connections = MutableStateFlow<List<Connection>>(emptyList())
    val connections: StateFlow<List<Connection>> = _connections

    private var trafficWebSocket: WebSocket? = null

    init {
        loadCurrentMode()
        startTrafficMonitoring()
    }

    private fun loadCurrentMode() {
        viewModelScope.launch {
            withContext(Dispatchers.IO) {
                api.getCurrentMode()?.let { mode ->
                    _currentMode.value = mode
                }
            }
        }
    }

    fun switchMode(mode: String) {
        viewModelScope.launch {
            withContext(Dispatchers.IO) {
                if (api.setMode(mode)) {
                    _currentMode.value = mode
                }
            }
        }
    }

    fun selectNode(group: String, node: String) {
        viewModelScope.launch {
            withContext(Dispatchers.IO) {
                api.selectNode(group, node)
            }
        }
    }

    fun closeConnection(id: String) {
        viewModelScope.launch {
            withContext(Dispatchers.IO) {
                api.closeConnection(id)
                refreshConnections()
            }
        }
    }

    fun closeAllConnections() {
        viewModelScope.launch {
            withContext(Dispatchers.IO) {
                api.closeAllConnections()
                _connections.value = emptyList()
            }
        }
    }

    private fun refreshConnections() {
        viewModelScope.launch {
            withContext(Dispatchers.IO) {
                api.getConnections()?.let { data ->
                    _connections.value = data.connections
                }
            }
        }
    }

    private fun startTrafficMonitoring() {
        trafficWebSocket = api.watchTraffic { traffic ->
            _trafficData.value = traffic
        }
    }

    override fun onCleared() {
        super.onCleared()
        trafficWebSocket?.close(1000, "ViewModel cleared")
    }
}
```

#### 6. Compose UI 示例

```kotlin
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

@Composable
fun VPNControlScreen(viewModel: VPNViewModel) {
    val currentMode by viewModel.currentMode.collectAsState()
    val trafficData by viewModel.trafficData.collectAsState()

    Column(modifier = Modifier.padding(16.dp)) {
        // 模式切换
        Text("路由模式", style = MaterialTheme.typography.titleLarge)
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            listOf("Rule", "Global", "Direct").forEach { mode ->
                FilterChip(
                    selected = currentMode == mode,
                    onClick = { viewModel.switchMode(mode) },
                    label = { Text(mode) }
                )
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        // 流量显示
        Text("实时流量", style = MaterialTheme.typography.titleLarge)
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("上传: ${formatBytes(trafficData.up)}/s")
                Text("下载: ${formatBytes(trafficData.down)}/s")
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        // 连接管理
        Button(
            onClick = { viewModel.closeAllConnections() },
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("断开所有连接")
        }
    }
}

fun formatBytes(bytes: Long): String {
    return when {
        bytes >= 1_000_000 -> "%.2f MB".format(bytes / 1_000_000.0)
        bytes >= 1_000 -> "%.2f KB".format(bytes / 1_000.0)
        else -> "$bytes B"
    }
}
```

---

## 错误处理

### HTTP 状态码

| 状态码 | 含义 |
|--------|------|
| `200` | 成功 |
| `204` | 成功（无返回内容） |
| `400` | 请求参数错误 |
| `401` | 未授权（Token 错误） |
| `404` | 资源不存在 |
| `500` | 服务器内部错误 |
| `503` | 服务不可用 |
| `504` | 网关超时 |

### 错误响应格式

```json
{
  "message": "错误描述信息"
}
```

**示例**：
```json
{
  "message": "Proxy not found"
}
```

---

## API 参考表

| 功能 | 方法 | 端点 | 说明 |
|------|------|------|------|
| **系统信息** | GET | `/version` | 获取版本 |
| **模式管理** | GET | `/configs` | 获取配置 |
| | PATCH | `/configs` | 切换模式 |
| **节点管理** | GET | `/proxies` | 获取所有节点 |
| | GET | `/proxies/{name}` | 获取单个节点 |
| | PUT | `/proxies/{name}` | 选择节点 |
| | GET | `/group` | 获取分组列表 |
| **延迟测试** | GET | `/proxies/{name}/delay` | 测试节点延迟 |
| | GET | `/group/{name}/delay` | 测试分组延迟 |
| **流量监控** | GET | `/traffic` (WS) | 实时流量 |
| | GET | `/memory` (WS) | 内存占用 |
| **日志** | GET | `/logs` (WS) | 实时日志 |
| **连接管理** | GET | `/connections` | 获取连接列表 |
| | GET | `/connections` (WS) | 实时连接 |
| | DELETE | `/connections/{id}` | 断开单个连接 |
| | DELETE | `/connections` | 断开所有连接 |

---

## 与 LibAnchor API 对比

| 功能 | Clash API (CLI) | LibAnchor (iOS/macOS) |
|------|-----------------|------------------------|
| **通信方式** | HTTP/WebSocket | 进程内调用 |
| **适用场景** | 远程控制、Web Dashboard | 原生 App 集成 |
| **性能** | 网络开销 | 直接调用，零开销 |
| **安全性** | Token 认证 | 进程隔离 |
| **实时推送** | WebSocket | 回调函数 |

---

**文档版本**: v1.0  
**更新日期**: 2025-11-21  
**适用版本**: sing-box 1.8.0+
