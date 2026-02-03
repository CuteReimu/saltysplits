# SaltySplits

这是一个用于分析 LiveSplits 的分段文件（.lss）的工具。它可以帮助用户更好地理解和管理他们的分段数据。

其灵感来自于：[jaspersiebring/saltysplits](https://github.com/jaspersiebring/saltysplits)

## 项目简介

SaltySplits 是一个专门为速通玩家设计的数据分析工具。它能够读取 LiveSplit 的 `.lss` 文件，解析其中的速通记录数据，并通过美观的 Chart.js 图表直观地展示统计信息。

### 主要功能

- **速通数据统计**：展示最佳纪录、SOB（Sum of Best）、可节约时间等关键指标
- **时间趋势分析**：使用折线图展示速通时间随尝试次数的变化趋势
- **重置统计**：通过饼图展示各分段的重置次数分布，帮助识别难点
- **速通分析**：对比展示多次最佳速通的分段用时情况
- **分段详细分析**：针对单个分段提供最快、最慢、平均、中位数及标准差等统计数据
- **自动打开浏览器**：程序启动后自动在浏览器中打开分析结果页面

### 适用场景

- 速通玩家优化练习策略，找出薄弱分段
- 分析长期练习趋势，评估进步情况
- 对比不同尝试的表现，学习成功经验

## 代码结构

```
saltysplits/
├── main.go           # 程序入口，Web服务器初始化和API路由
├── xml.go            # XML数据结构定义和自定义类型（Duration）
├── analysis.go       # 核心分析逻辑，数据统计和计算
├── index.html        # 前端页面，使用Vue.js和Chart.js展示数据
├── go.mod            # Go模块依赖管理
├── go.sum            # Go模块依赖校验
├── download_cdn.sh   # Linux/Mac下载前端依赖脚本
├── download_cdn.ps1  # Windows下载前端依赖脚本
└── static/           # 前端依赖库存放目录
```

### 核心模块说明

#### main.go
- 程序入口和异常处理
- 使用 Gin 框架初始化 Web 服务器（监听 `127.0.0.1:12334`）
- 提供以下 REST API 端点：
  - `GET /` - 返回前端页面
  - `GET /data` - 返回游戏和分类信息
  - `GET /summary` - 返回统计摘要数据
  - `GET /totalData` - 返回总时间数据（Real Time 和 Game Time）
  - `GET /reset` - 返回各分段重置统计
  - `GET /breakdown` - 返回速通分析数据
  - `GET /segment?index=N` - 返回指定分段的详细统计

#### xml.go
- 定义 LiveSplit `.lss` 文件的 XML 数据结构
- 实现自定义 `Duration` 类型，用于时间的解析和序列化
- 支持 XML 到 Go 结构的自动映射

#### analysis.go
- 实现所有数据分析逻辑：
  - `analysisInfo()` - 计算最佳时间、SOB、可节约时间等汇总信息
  - `analysisTotalData()` - 整理每次尝试的总时间数据
  - `analysisResetData()` - 统计各分段的重置次数
  - `analysisRun()` - 分析前5次最佳速通的分段数据
  - `getSegment()` - 计算指定分段的统计指标（平均值、中位数、标准差等）

#### index.html
- 单页面应用，使用 Vue 3 和 Element Plus 构建
- 使用 Chart.js 绘制多种图表（折线图、饼图等）
- 通过 Axios 异步获取后端数据并实时更新图表

## 编译与使用

### 前置要求

- Go 1.24 或更高版本
- Linux/Mac 需要有 `curl` 命令，Windows 可使用 PowerShell

### 步骤

#### 1. 下载前端依赖

首先需要下载前端所需的第三方JavaScript库：

**Linux/Mac：**
```bash
chmod +x download_cdn.sh
./download_cdn.sh
```

**Windows PowerShell：**
```powershell
.\download_cdn.ps1
```

该脚本会自动创建 `static/` 目录并下载以下库：
- Vue 3.5.22
- Element Plus 2.11.3
- Axios 1.12.2
- Chart.js 4.5.0

#### 2. 编译程序

**Linux/Mac：**
```bash
go build -o saltysplits
```

**Windows：**
```bash
go build -o saltysplits.exe
```

#### 3. 运行程序

**方式一：拖拽文件**
- 直接运行编译好的可执行文件
- 根据提示将 `.lss` 文件拖入终端窗口
- 按回车键开始分析

**方式二：命令行参数**
```bash
# Linux/Mac
./saltysplits -f /path/to/your/file.lss

# Windows
saltysplits.exe -f C:\path\to\your\file.lss
```

#### 4. 查看结果

程序会自动在默认浏览器中打开 `http://127.0.0.1:12334/` 展示分析结果。

如果自动打开失败，请手动访问该地址。

### 大文件处理

当 `.lss` 文件包含超过 200 次尝试时，程序会提示输入起始尝试ID，以便缩小分析范围，提高处理速度。

## 使用的第三方库

### 前端库（JavaScript）
- **Vue 3** - 渐进式JavaScript框架，用于构建用户界面
- **Element Plus** - 基于 Vue 3 的组件库，提供丰富的UI组件
- **Chart.js** - 灵活的JavaScript图表库，用于数据可视化
- **Axios** - 基于 Promise 的 HTTP 客户端，用于API请求

### 后端库（Go）
- **Gin** - 高性能的Go Web框架，用于构建REST API

## 开发相关

### 代码规范

项目使用 golangci-lint 进行代码质量检查，配置文件为 `.golangci.yml`。

运行代码检查：
```bash
golangci-lint run --timeout=5m
```

### 项目特点

- **无需外部数据库**：所有数据在内存中处理
- **跨平台支持**：支持 Windows、Linux、macOS
- **本地运行**：数据不会上传到外部服务器，保护隐私
- **自动资源管理**：使用 Go 的 embed 特性嵌入静态文件

## 许可证

请查看 LICENSE 文件了解详情。