# 本地开发说明

## 1. 初始化配置

在根目录运行 `scripts/setup-local.ps1`。它从各模块的 `*.example` 文件生成本地配置，已有文件会跳过；后端 JWT 使用随机值。实际配置被 `.gitignore` 排除。

前端环境示例保留 Vite 使用的变量名；其中 `VITE_` 变量会进入浏览器构建产物，不能用于保管服务端密钥。示例已关闭模板 API 加密并清空相关键值及默认登录口令。

## 2. 本地依赖和后端

根目录 `docker-compose.yml` 只提供本机 MySQL、Redis 和 EMQX，监听端口均绑定 127.0.0.1。MySQL 使用命名数据卷，停止服务不会自动清除数据。不要把本地示例 Compose 当作生产部署方案。

后端在 `backend/` 中执行 `go run ./cmd/server`，默认读取 `config/config.yaml`。也可以用 `-config` 指定路径。启动过程包含数据库迁移和初始化，首次运行请只连接独立的本地数据库。

首次管理员口令可通过当前终端的 `SSM_ADMIN_INITIAL_PASSWORD` 环境变量设置；不设置时，按现有后端实现生成初始凭据文件。请在本机查看该文件并保管，不提交到仓库。数据库已有管理员时，修改初始化变量不等于重置账号。

微信和真实设备默认未配置。需要隔离的本地 Mock 联调时，在本地配置设置 `server.enable_mock: true`，或启动前设置 `$env:SSM_SERVER_ENABLE_MOCK = 'true'`。这只开启现有模拟接口，不会自动生成完整演示数据。退出联调后关闭 Mock。

## 3. 前端

两个 V3 前端分别安装依赖并运行。平台端端口 5173，商家端 5174；开发 API 默认指向 localhost:8080。`build/vite/` 是源码，不是可删除的构建结果。

首次安装依赖通常需要网络。优先使用各自 pnpm-lock.yaml 与 `pnpm install --frozen-lockfile`。不复用旧项目的 node_modules 作为发布内容。

本轮受网络权限限制，构建验证使用独立复制的本机已有依赖；依赖与产物均被 Git 排除。这不能替代在另一台电脑进行锁文件全新安装验证。两个后台已分别通过生产构建、桌面前端构建与完整类型检查，具体范围见 PREPARATION.md。

复制的依赖保留包管理器安装元数据，pnpm 的依赖状态校验可能尝试重新安装；当前环境会被网络权限阻止。因此本机离线检查优先使用根目录 `scripts/check-local.ps1`，直接调用已安装工具。正式开发或在新电脑使用前，仍应完成一次正常锁文件安装，而非把本机 node_modules 发布给他人。

自动生成的 `src/types/auto-imports.d.ts` 与 `auto-components.d.ts` 作为类型声明种子随源码保留，插件构建时可更新；不要将它们与依赖或构建产物一起排除。

## 4. 微信小程序

在微信开发者工具中分别导入两个小程序目录。公开 `project.config.json` 使用 `touristappid` 占位；正式登录、手机号授权等功能需要更换为自己的 AppID 并配置后端凭据。

开发 API 位于各模块 `utils/api.js`，默认 `http://localhost:8080/api/v1`。localhost 只适用于开发工具所在电脑；真机调试需配置可访问的 HTTPS 开发后端和合法域名。不要把真实 AppSecret 放入小程序前端。

## 5. Windows 桌面入口

从对应 desktop 目录执行 `./build.ps1`。打包脚本先构建相邻 V3 前端，再复制静态文件到 `web/`，最后编译 Windows exe。`web/`、`*.exe`、`*.syso` 及安装包均不提交源码仓库。

平台本地端口 8787，商家 8788；公开版反代默认连接 localhost:8080，需要先运行本地后端。商家安装脚本需要另行安装 Inno Setup，安装包构建尚未在本轮验证。

已有 icon.ico 可直接打包。重新生成图标时，显式传入自己的图片：`./make-icon.ps1 -Source C:/path/to/logo.png`；脚本不再依赖原作者电脑上的官网目录。

## 6. 检查

- 后端：`go test ./...`。
- 平台与商家前端：各自执行 `pnpm ts:check`、`pnpm lint`、`pnpm build`。
- 小程序：开发工具编译与两端流程联调。
- 设备：使用测试门店和测试硬件验证 MQTT、订单时效和自动关闭策略。

当前实际执行情况见 PREPARATION.md，不能用上述命令清单代替验证结果。
