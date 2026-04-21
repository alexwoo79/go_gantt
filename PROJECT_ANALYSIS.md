# go_echarts_tools 项目完整分析文档

## 一、项目概述

**项目名称**: go_echarts_tools (甘特图与可视化工具)  
**核心功能**: 
- CSV/XLSX 文件自动解析
- 列字段智能识别和映射
- 甘特图和多种 ECharts 可视化图表生成
- Web 界面实时预览和配置

**技术定位**: Go 后端 + Web 前端的轻量级数据可视化平台

---

## 二、文件结构详解

```
go_echarts_tools/
├── main.go                    # 【入口点】仅做资源嵌入和服务启动
├── go.mod                     # 依赖配置 (Gin、ECharts、Excelize)
├── build.sh                   # 多平台编译脚本
│
├── internal/                  # 核心业务逻辑
│   ├── model/
│   │   └── types.go           # 【数据模型层】Task/Stats/Dataset/MappingConfig
│   │
│   ├── data/
│   │   ├── parse.go           # 【数据解析】CSV/XLSX 解析器 + 日期格式转换
│   │   └── store.go           # 【内存存储】线程安全的 Dataset 缓存
│   │
│   ├── charts/
│   │   ├── chart.go           # 【核心接口】ChartBuilder 注册表 (支持多图表类型)
│   │   └── gantt/
│   │       └── gantt.go       # 【甘特图实现】Task 构建 + 数据统计
│   │
│   ├── server/
│   │   ├── server.go          # 【路由配置】Gin 服务器初始化
│   │   ├── handlers.go        # 【HTTP 处理】上传/映射/图表生成端点
│   │   └── render.go          # 【模板渲染】传递数据到前端
│   │
│   └── viz/
│       └── viz.go             # 【通用可视化】多种 ECharts 类型支持
│
├── static/                    # 前端资源
│   ├── chart.js               # ECharts 甘特图配置
│   ├── viz.js                 # 通用 ECharts 可视化逻辑
│   ├── chart_layout.js        # 页面布局管理
│   ├── style.css              # 样式表
│   └── echarts_sidebar_icons/ # 图标资源
│
├── templates/                 # HTML 模板 (Gin 模板)
│   ├── entry.tmpl             # 入口页面
│   ├── index.tmpl             # 甘特图编辑页面
│   ├── chart.tmpl             # 图表展示页面
│   └── viz.tmpl               # 通用可视化编辑页面
│
└── test_csv/
    └── radar.csv              # 测试数据
```

---

## 三、技术栈详解

### 后端
| 层级 | 技术 | 用途 |
|------|------|------|
| **Web 框架** | Gin v1.12.0 | 高性能 HTTP 路由和中间件 |
| **Excel 解析** | excelize v2.10.1 | XLSX 文件读取 |
| **标准库** | csv、time、sync | CSV 解析、日期处理、并发控制 |
| **部署** | Go 1.25.0 | 原生编译，无依赖运行 |

### 前端
- **图表库**: ECharts (嵌入式 JS 文件)
- **交互**: 原生 HTML/CSS/JavaScript
- **特点**: 零框架依赖、完全由后端嵌入资源服务

---

## 四、数据流概览

### 用户交互流程：

1. **上传阶段**
   - 用户上传 CSV/XLSX 文件 → handlers.ganttUpload
   - data.ParseUploadedFile 自动检测格式
   - 数据规范化、处理空值
   - 生成 Dataset 对象并存储

2. **映射阶段**
   - 前端显示列名勾选表单
   - inferDefaults 智能推荐列映射
   - 用户可手动调整 MappingConfig

3. **构建阶段**
   - 用户点击"生成图表" → POST /gantt/chart
   - ChartBuilder.Build() 调用
   - 转换列索引、解析日期、计算统计

4. **渲染阶段**
   - Task 和 Stats 对象转为 JSON
   - 传递给前端 chart.js
   - ECharts 初始化并展示甘特图

---

## 五、核心设计模式

### 1. ChartBuilder 工厂模式

```
charts.Register() 
  ↓ 
[ChartBuilder 接口]
  ↓ 
charts/gantt/ 自动注册 (init 方法)
```

**优势**: 新增图表类型只需：
1. 在 `internal/charts/<新类型>/builder.go` 实现 `ChartBuilder` 接口
2. 在 init() 中调用 `charts.Register()`
3. 在 server.go 中加导入 `_ "gantt/internal/charts/<新类型>"`

### 2. 线程安全存储

```go
var store = struct {
    sync.RWMutex
    m map[string]model.Dataset
}{}
```

**设计目的**: 支持并发上传和查询请求

### 3. 资源嵌入 (Embed)

```go
//go:embed static/* templates/*.tmpl
var embeddedAssets embed.FS
```

**好处**: 编译后二进制包含所有资源，单文件部署

---

## 六、HTTP API 路由

### Gantt 路由
```
GET  /gantt              ← 甘特图首页（空白）
GET  /gantt/demo         ← 加载演示数据
POST /gantt/upload       ← 上传 CSV/XLSX → 显示列映射表单
POST /gantt/chart        ← 提交映射配置 → 生成甘特图
GET  /gantt/clear        ← 清空当前数据
```

### Viz 路由（通用可视化）
```
GET  /viz                ← 多图表类型选择页
GET  /viz/demo           ← 演示数据
POST /viz/upload         ← 上传文件
POST /viz/chart          ← 生成图表
POST /viz/validate-hierarchy  ← 验证树形结构
```

---

## 七、数据模型详览

### Task（任务对象）
```go
type Task struct {
    TaskName      string    // 任务名称
    Project       string    // 所属项目（用于分组）
    ColorGroup    string    // 颜色分组
    StartISO      string    // ISO 8601 格式开始日期
    EndISO        string    // ISO 8601 格式结束日期
    PlanStartISO  string    // 计划开始（可选）
    PlanEndISO    string    // 计划结束（可选）
    DurationDays  int       // 实际时长（天）
    Description   string    // 任务描述
    MilestoneName string    // 里程碑名称
    MilestoneISO  string    // 里程碑日期
    Owner         string    // 任务责任人
}
```

### Dataset（原始数据集）
```go
type Dataset struct {
    ID      string          // 唯一标识（纳秒时间戳）
    Name    string          // 原始文件名
    Headers []string        // 列名
    Rows    [][]string      // 数据行
}
```

### MappingConfig（列映射配置）
```go
type MappingConfig struct {
    TaskCol          string  // 任务列名
    StartCol         string  // 开始日期列
    EndCol           string  // 结束日期列
    ProjectCol       string  // 项目列
    PlanStartCol     string  // 计划开始列
    PlanEndCol       string  // 计划结束列
    DescCol          string  // 描述列
    // ... 等 11 个字段
    SortByStart      bool    // 是否按开始日期排序
    ShowTaskNumber   bool    // 是否显示任务编号
}
```

---

## 八、📈 技术拓展方向

### 优先级 1️⃣（高价值）

#### 1. 数据库持久化
- 替换 map 内存存储为 SQLite/PostgreSQL
- 支持上传历史查询和版本管理
- 建议：使用 gorm 库

#### 2. 认证与权限
- JWT 令牌认证
- 基于用户的数据隔离
- 分享链接（临时访问权限）

#### 3. 导出功能增强
- PDF 导出（ECharts 服务）
- PNG/SVG 图表快照
- Excel 报告（含统计和图表）
- 可用库：go-echarts/go-echarts-service

#### 4. 更多图表类型
- 📊 柱状图/折线图/散点图
- 🍪 饼图/环形图
- 📅 日历热力图
- 🌳 树形图/流向图
- 🗺️ 地图可视化

### 优先级 2️⃣（功能增强）

#### 5. 数据处理能力
- 数据清洗（去重、补全、转换）
- 高级过滤和分组聚合
- 时间序列分析
- 数据验证规则引擎

#### 6. 协作与分享
- 实时多人编辑（WebSocket）
- 评论和标注
- 模板库管理
- 权限分享设置

#### 7. 性能优化
- 大数据集分页加载
- 虚拟滚动（前端）
- 数据压缩传输
- 缓存策略（Redis）

#### 8. 数据源集成
- 直接连接 SQL 数据库
- API 数据源（Webhook）
- 定时执行和更新
- 区块链数据源（可选）

### 优先级 3️⃣（高级功能）

#### 9. 智能化
- AI 列映射建议（模式学习）
- 异常值检测
- 预测和趋势分析
- 自动报告生成

#### 10. 移动端支持
- 响应式设计优化
- 移动端专用交互
- PWA 离线支持

#### 11. 国际化
- 多语言支持 (i18n)
- 区域化时间和数字格式
- RTL 语言支持

---

## 九、🚀 快速开始指南

### 开发环境配置

```bash
# 1. 克隆或进入项目
cd /Users/crccredc/Documents/github/go_echarts_tools

# 2. 安装依赖
go mod download

# 3. 本地运行
go run main.go
# 输出: server started at http://localhost:8080

# 4. 前往浏览器
open http://localhost:8080
```

### 目录导航快速索引

| 目标 | 文件位置 | 说明 |
|------|---------|------|
| 修改 HTTP 路由 | internal/server/server.go | 加新的 GET/POST 端点 |
| 新增图表类型 | internal/charts/<新类型>/ | 实现 ChartBuilder 接口 |
| 修改前端逻辑 | static/chart.js 或 static/viz.js | ECharts 配置和交互 |
| 修改 UI 样式 | static/style.css | HTML 布局样式 |
| 调整模板 | templates/ | Gin 模板变量和结构 |
| 添加测试数据 | test_csv/ | CSV 或 XLSX 示例 |

---

## 十、🔍 代码阅读路线图

**推荐阅读顺序**：

```
1. main.go  
   └─> 了解如何启动服务和嵌入资源

2. internal/model/types.go
   └─> 理解核心数据结构

3. internal/data/parse.go + store.go
   └─> 学习数据解析和存储

4. internal/charts/chart.go
   └─> 理解工厂模式和扩展机制

5. internal/charts/gantt/gantt.go
   └─> 看具体的图表构建实现

6. internal/server/server.go + handlers.go
   └─> 理解路由和 HTTP 处理

7. templates/ + static/
   └─> 理解前端如何渲染和交互
```

---

## 十一、📝 常见任务操作

### 任务 1: 新增一个柱状图表类型

```
1. 创建目录: internal/charts/bar/
2. 新建文件: internal/charts/bar/builder.go
3. 实现 ChartBuilder 接口:
   - ID() 返回 "bar"
   - Name() 返回 "柱状图"
   - InferDefaults() 推荐列映射
   - DefaultOptions() 返回默认选项
   - Build() 构建图表数据
4. 在 builder.go init() 中调用 charts.Register(New())
5. 在 internal/server/server.go 加入: _ "gantt/internal/charts/bar"
6. 重启应用，前端自动显示新图表类型
```

### 任务 2: 修改日期解析格式

**位置**: internal/data/parse.go ParseDate() 函数  
**操作**: 添加新格式到 layouts 切片即可

### 任务 3: 修改甘特图配色

**位置**: static/chart.js 中的色彩配置  
**操作**: 修改 colors 数组或 theme 配置

### 任务 4: 部署到生产环境

```bash
# 编译所有平台
bash build.sh

# 输出在 dist/ 目录，包含 Windows/macOS/Linux 版本
# 每个版本都是独立的可执行文件，包含所有资源
./dist/gantt-darwin-arm64/gantt &
```

---

## 十二、🧪 测试和调试

### 使用演示数据
```
GET /gantt/demo
GET /viz/demo
```

### 查看服务状态
```bash
curl http://localhost:8080/gantt
```

### 上传 CSV 测试
```bash
curl -F "data_file=@test_csv/radar.csv" http://localhost:8080/gantt/upload
```

### 调试 JavaScript
在浏览器开发者工具中：
- Console: 查看 JavaScript 错误
- Network: 查看 HTTP 请求和响应
- Sources: 设置断点调试

---

## 十三、⚠️ 已知限制和改进空间

| 项目 | 现状 | 建议改进 |
|------|------|---------|
| **数据持久化** | 内存存储，重启丢失 | 集成数据库，支持版本控制 |
| **并发限制** | RWMutex 线程安全，但单进程 | 分布式部署、负载均衡 |
| **上传文件大小** | 无限制（依赖 Gin 配置） | 添加文件大小限制、分块上传 |
| **日期格式** | 支持 20+ 格式，但有边界 | 用户自定义格式模板 |
| **柱体隔离** | 基于日期，不支持自定义分组 | 支持多维分组、钻取分析 |
| **字符编码** | 支持 UTF-8 | 添加自动编码检测 |
| **API 文档** | 无 | 集成 Swagger/OpenAPI |
| **单元测试** | 无 | 为核心函数补充测试 |

---

## 十四、💡 最佳实践建议

1. **列命名规范**: 在您的 CSV 中使用清晰的英文列名（Project、Task、StartDate 等），便于自动识别

2. **日期格式统一**: 建议使用 ISO 8601 格式 (YYYY-MM-DD)，提高兼容性

3. **模块化扩展**: 遵循 ChartBuilder 接口，不修改核心代码添加新功能

4. **定期清理内存**: 如果需要长期运行，建议集成数据库和定期数据清理

5. **安全考虑**: 
   - 上传文件验证（文件类型、大小）
   - 用户认证和授权
   - SQL 注入防护（如果接入数据库）

---

## 附录：关键算法详解

### 日期解析（parse.go）
支持格式包括：
- RFC3339 标准格式
- Excel 日期序列号
- 常见中文格式 (2026-01-06、2026/1/6 等)
- 20种+ 其他格式

### 智能列映射推荐
基于列名关键词匹配：
- 优先级匹配 "task"、"任务" → TaskCol
- 优先级匹配 "start"、"开始" → StartCol
- 优先级匹配 "end"、"结束" → EndCol
- 等等...

### 统计计算（gantt.go）
- **TaskCount**: 任务总数
- **AvgDurationDays**: 平均时长
- **MaxDurationDay**: 最长任务时长
- **TotalDurationDay**: 所有任务跨度（最后日期 - 最早日期）

---

**文档生成时间**: 2026年4月21日  
**版本**: 1.0  
**适用范围**: go_echarts_tools 项目完整分析和扩展指南
