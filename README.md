# 文件结构
main.go                          ← 纯入口，仅 embed + 启动 server
internal/
  model/types.go                 ← 所有共享数据类型 (Task, Stats, Dataset, MappingConfig…)
  data/
    parse.go                     ← CSV/XLSX 解析、日期解析、Cell 工具函数
    store.go                     ← 数据集内存存储（线程安全）
  charts/
    chart.go                     ← ChartBuilder 接口 + 注册表 (Register/Get/All)
    gantt/
      gantt.go                   ← 甘特图实现，init() 自动注册
  server/
    server.go                    ← gin 路由、静态文件、模板装配
    handlers.go                  ← HTTP 处理器（home/demo/upload/chart）
    render.go                    ← renderWorkspace / renderMapper 渲染辅助


# 后续扩展
## 后续扩展新图表类型只需：

在 internal/charts/<新类型>/ 下实现 ChartBuilder 接口
在该包的 init() 中调用 charts.Register(New())
在 server.go 中加一行 _ "gantt/internal/charts/<新类型>" 侧效导入

## 无需修改任何核心代码。后续扩展新图表类型只需：

在 internal/charts/<新类型>/ 下实现 ChartBuilder 接口
在该包的 init() 中调用 charts.Register(New())
在 server.go 中加一行 _ "gantt/internal/charts/<新类型>" 侧效导入
无需修改任何核心代码。