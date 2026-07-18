# SSPU Connect

> 本项目是 [Mythologyli/zju-connect](https://github.com/Mythologyli/zju-connect) 的 SSPU（上海第二工业大学）独立优化适配版。底层 EasyConnect/aTrust、用户态网络栈与多数通用功能来自上游；本仓库专注于 `vpn.sspu.edu.cn` 的兼容性、会话管理、Clash 共存和可诊断日志。感谢 zju-connect 及更早的 [EasierConnect](https://github.com/lyc8503/EasierConnect) 项目。

SSPU Connect 是非官方客户端，与上海第二工业大学、深信服及上游项目均无隶属关系。软件按原样提供，请遵守学校网络使用规定；因使用本软件产生的风险由使用者自行承担。

## SSPU 适配内容

- 默认使用 EasyConnect 协议连接 `vpn.sspu.edu.cn:443`。
- 接受 SSPU 不下发远端 DNS 的服务端配置，保留服务端下发的 IP、域名和静态 DNS 资源。
- 每 60 秒更新服务端会话，适配“约 1 小时无响应自动断连”的策略。
- 退出时主动注销当前会话；重新登录可按 SSPU 行为覆盖同一账号已有的活跃连接。
- 阻止 Clash Fake-IP（`198.18.0.0/15`）误入 VPN 隧道，降低错误分流造成的循环连接和频繁报错。
- 域名规则按 DNS 标签边界匹配，避免 `not-example.edu.cn` 错误命中 `example.edu.cn`。
- 路由、DNS、会话和代理错误使用结构化日志，同时隐藏 TWFID、隧道 token、会话 ID 等敏感信息。

SSPU 的账号规则是同一时间只保留一个 VPN 连接。启动新的连接可能挤掉同账号之前的连接，这是服务端的正常行为。

## 快速开始

### Windows 发行包

1. 从 [Releases](https://github.com/Qintsg/sspu-connect/releases) 下载 `sspu-connect-v0.1.0-windows-amd64.zip` 并解压。
2. 在解压目录复制 `vpn.env.example` 为 `vpn.env`，填写账号和密码。
3. 在 PowerShell 中执行：

```powershell
$config = @{}
Get-Content .\vpn.env | Where-Object { $_ -match '^\s*[^#].*=' } | ForEach-Object {
    $key, $value = $_ -split '=', 2
    $config[$key.Trim()] = $value.Trim()
}

.\sspu-connect.exe `
    -protocol $config.VPN_PROTOCOL `
    -server $config.VPN_SERVER `
    -port ([int]$config.VPN_PORT) `
    -username $config.VPN_USERNAME `
    -password $config.VPN_PASSWORD
```

启动成功后提供以下本地代理：

- SOCKS5：`127.0.0.1:1080`
- HTTP：`127.0.0.1:1081`

按 `Ctrl+C` 停止程序。程序会先请求服务端注销，再关闭本地代理。

也可以直接传入参数：

```powershell
.\sspu-connect.exe -username "学号" -password "密码"
```

直接传参可能将密码留在终端历史中，日常使用推荐 `vpn.env`。`vpn.env` 已被 Git 忽略，请勿提交、截图或上传该文件。

### 从源码运行

需要 Go 1.25 或更高版本：

```powershell
go mod download
go run . -username "学号" -password "密码"
```

构建 Windows x64 可执行文件：

```powershell
$env:CGO_ENABLED = '0'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
go build -trimpath -ldflags '-s -w -buildid=' -o sspu-connect.exe .
```

## 与 Clash 共存

SSPU Connect 支持在 Clash/FlClash TUN 已开启时运行，并保留 Clash 对普通直连流量的处理，不要求绑定物理网卡绕过 Clash。

推荐使用方式：

1. 保持 Clash 正常运行。
2. 启动 SSPU Connect，仅暴露本地 SOCKS5/HTTP 代理。
3. 对需要访问 SSPU VPN 资源的应用，显式使用 `127.0.0.1:1080` 或 `127.0.0.1:1081`。
4. 不要再把 SSPU Connect 的监听端口配置成回到自身的上游代理。

当 Clash Fake-IP 被误判为 VPN 目标时，程序会拒绝连接并记录：

```text
route host="..." destination="198.18.x.x:443" network=tcp action=REJECT reason=clash-fake-ip
```

这表示域名在进入 SSPU Connect 前已被 Clash 替换成 Fake-IP。应在 Clash 规则中修正该域名的解析或代理顺序，而不是把 Fake-IP 送进 SSPU 隧道。

如果确实需要让未命中 SSPU 规则的连接显式走 Clash HTTP 代理，可以增加：

```powershell
-dial-direct-proxy http://127.0.0.1:7890
```

通常不需要该参数；默认直连会继续遵循 Windows 当前系统路由，因此本机 Clash TUN 仍然生效。

## 路由与 DNS

SSPU 服务端当前不下发通用远端 DNS，这是预期行为。程序会：

- 使用服务端静态 DNS 表解析已下发的校内域名；
- 使用服务端域名/IP 资源决定是否进入 VPN；
- 其余域名通过本机/备用 DNS 解析，并按普通系统路由连接；
- 不修改系统固定 DNS，也不会伪造“服务端已下发 DNS”。

登录后可从摘要日志确认实际资源：

```text
resource summary ip_rules=... domain_rules=... static_dns=... remote_dns_available=false
```

`remote_dns_available=false` 对 SSPU 是正常状态，不代表登录失败。

## 会话与保活

程序包含两类保活：

- EasyConnect 会话更新：固定每 60 秒请求服务端，默认开启，不依赖服务端 DNS。
- 业务流量保活：上游的 `keep-alive-url` 机制；SSPU 无远端 DNS 时，未配置 URL 会自动停用。

一般无需设置 `keep-alive-url`。不要使用 `-disable-keep-alive` 关闭上游业务保活后误以为会话更新也被关闭；SSPU 的会话更新由 EasyConnect 客户端生命周期独立维护。

如果同一账号在另一台设备或另一个进程重新登录，旧连接可能被覆盖并断开。请确保本机只运行一个 SSPU Connect 实例。

## 常用参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-protocol` | `easyconnect` | SSPU 使用 EasyConnect |
| `-server` | `vpn.sspu.edu.cn` | VPN 服务端 |
| `-port` | `443` | VPN 服务端端口 |
| `-username` | 空 | 学号/账号 |
| `-password` | 空 | VPN 密码 |
| `-socks-bind` | `:1080` | SOCKS5 监听地址；仅本机使用可设为 `127.0.0.1:1080` |
| `-http-bind` | `:1081` | HTTP 代理监听地址；设为空字符串可禁用 |
| `-dial-direct-proxy` | 空 | 未命中 VPN 规则时使用的 HTTP 上游代理 |
| `-custom-proxy-domain` | 空 | 强制进入 VPN 的额外域名，多个域名用逗号分隔 |
| `-custom-dns` | 空 | 静态域名映射，格式为 `域名:IP,域名:IP` |
| `-proxy-all` | `false` | 强制代理全部 IPv4；不建议与 Clash TUN 同时使用 |
| `-config` | 空 | 使用 TOML 配置文件；启用后其他命令行配置不生效 |
| `-debug-dump` | `false` | 输出调试流量，仅排错时短期开启 |
| `-version` | — | 显示版本 |

完整上游参数仍然保留，可执行 `sspu-connect.exe -h` 查看。TUN、aTrust、端口转发、Shadowsocks、证书和验证码等高级能力沿用上游实现，但未作为 SSPU 默认使用路径进行保证。

## 日志与排错

正常启动应依次看到登录成功、资源摘要、代理监听和周期性会话保活日志。重点字段如下：

- `action=VPN`：命中 SSPU 域名/IP 资源，连接进入隧道。
- `action=DIRECT`：未命中 SSPU 资源，连接沿本机系统路由发出。
- `action=REJECT reason=clash-fake-ip`：拦截 Clash Fake-IP，避免代理循环。
- `action=REJECT reason=server-acl`：服务端 ACL 明确不允许该目标进入隧道。
- `session keepalive status=OK`：服务端会话更新成功。
- `remote_dns_available=false`：SSPU 未下发远端 DNS，属于正常状态。

常见问题：

1. **频繁出现连接失败或 SHUTDOWN**：先看目标是 `VPN`、`DIRECT` 还是 `REJECT`。错误网站被送入 VPN 可能触发服务端 ACL 或断连。
2. **目标地址是 `198.18.x.x`**：这是 Clash Fake-IP，不是实际网站 IP；检查 Clash DNS 与分流规则。
3. **刚连上就把另一台设备踢下线**：SSPU 只允许一个活跃连接，新登录覆盖旧登录。
4. **运行一段时间后断开**：检查是否持续出现 `session keepalive status=OK`，以及是否有网络切换、休眠或另一处重新登录。
5. **代理对局域网暴露**：默认 `:1080`/`:1081` 会监听所有接口。只在本机使用时显式设置 `-socks-bind 127.0.0.1:1080 -http-bind 127.0.0.1:1081`。

提交 Issue 前请删除账号、密码、TWFID、Cookie、token、会话 ID 和内网业务数据。即使本版本已减少敏感日志，也应人工检查后再公开日志。

## 配置文件

仓库提供 [config.toml.example](config.toml.example)。复制后修改并运行：

```powershell
Copy-Item .\config.toml.example .\config.toml
.\sspu-connect.exe -config .\config.toml
```

包含密码的 `config.toml` 不应提交到仓库。对于本仓库的本地联调，优先使用已忽略的 `vpn.env`。

## 上游同步策略

本仓库保留上游提交历史，并配置 `upstream` 指向 `Mythologyli/zju-connect`。SSPU 特有改动尽量集中在服务端兼容、路由边界、会话生命周期、日志和文档层，便于持续同步上游安全修复与协议更新。

上游通用问题和功能请优先在 [zju-connect](https://github.com/Mythologyli/zju-connect) 查找；仅 SSPU 环境可复现的问题请提交到本仓库。

## 许可证与致谢

本项目遵循仓库中的 [GPL-3.0 License](LICENSE)。

- [Mythologyli/zju-connect](https://github.com/Mythologyli/zju-connect)
- [lyc8503/EasierConnect](https://github.com/lyc8503/EasierConnect)
- zju-connect 的全部贡献者与依赖项目维护者
