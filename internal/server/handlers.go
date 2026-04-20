// Package server contains HTTP request handlers.
package server

import (
	"io/fs"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"gantt/internal/charts"
	chartgantt "gantt/internal/charts/gantt"
	"gantt/internal/data"
	"gantt/internal/model"
	"gantt/internal/viz"
)

const demoCSV = `Project,Task,StartDate,EndDate,Description,Milestone,MilestoneDate,PlanStartDate,PlanEndDate,Owner
Website,Requirements,2026-04-01,2026-04-05,Define scope,Scope Locked,2026-04-05
Website,Design,2026-04-06,2026-04-12,UI and UX design,Design Review,2026-04-11,2026-04-06,2026-04-10,Alice
Website,Frontend,2026-04-13,2026-04-24,Build pages,,,2026-04-12,2026-04-22,Bob
Website,Backend,2026-04-14,2026-04-26,Build APIs,,,2026-04-13,2026-04-23,Chris
Website,Integration,2026-04-27,2026-05-02,Integrate FE and BE,Integration Complete,2026-05-02,2026-04-25,2026-05-01,Diana
Website,Launch,2026-05-03,2026-05-05,Release to production,Go Live,2026-05-05,2026-05-02,2026-05-04,Eric
`

const vizDemoCSV = `Category,Month,Revenue,Cost,Profit,Share,Source,Target,LinkValue,NodeID,ParentID,NodeValue,ScatterX,ScatterY,ScatterSize
Cloud,2026-01,120,72,48,28,,,,Cloud,,,12,12.5,28.4,18
Cloud,2026-02,132,79,53,30,,,,Cloud,,,13,13.1,29.1,20
Cloud,2026-03,145,84,61,31,,,,Cloud,,,15,14.2,31.6,22
Security,2026-01,98,58,40,22,,,,Security,,,11,10.5,21.4,14
Security,2026-02,103,61,42,23,,,,Security,,,11.5,11.2,22.3,16
Security,2026-03,111,66,45,24,,,,Security,,,12.2,11.8,23.1,18
AI,2026-01,86,49,37,18,,,,AI,,,10.2,9.8,18.4,12
AI,2026-02,95,53,42,20,,,,AI,,,11.0,10.4,19.7,14
AI,2026-03,108,60,48,22,,,,AI,,,11.8,11.3,21.8,16
Data,2026-01,73,44,29,14,,,,Data,,,8.5,8.1,15.7,10
Data,2026-02,80,47,33,15,,,,Data,,,9.1,8.8,16.9,12
Data,2026-03,89,52,37,16,,,,Data,,,9.7,9.5,17.8,14
,,,,,,Cloud,Security,36,,,,,,
,,,,,,Cloud,AI,28,,,,,,
,,,,,,Cloud,Data,18,,,,,,
,,,,,,Security,AI,22,,,,,,
,,,,,,Security,Data,19,,,,,,
,,,,,,AI,Data,15,,,,,,
,,,,,,,,,Platform,,100,,,
,,,,,,,,,Cloud,Platform,40,,,
,,,,,,,,,Security,Platform,25,,,
,,,,,,,,,AI,Platform,20,,,
,,,,,,,,,Data,Platform,15,,,
`

type handlers struct {
	assets fs.FS
}

func (h *handlers) entry(c *gin.Context) {
	c.HTML(200, "entry.tmpl", gin.H{"Title": "图形工具入口"})
}

func (h *handlers) ganttHome(c *gin.Context) {
	c.HTML(200, "index.tmpl", gin.H{"Title": "Gantt - Go + Gin + ECharts"})
}

func (h *handlers) ganttDemo(c *gin.Context) {
	dataset, err := data.ParseCSV("demo.csv", strings.NewReader(demoCSV))
	if err != nil {
		c.HTML(200, "index.tmpl", gin.H{"Title": "Gantt - Go + Gin + ECharts", "Error": err.Error()})
		return
	}
	data.Store(dataset)
	renderMapper(c, dataset, "")
}

func (h *handlers) ganttClear(c *gin.Context) {
	c.HTML(200, "index.tmpl", gin.H{"Title": "Gantt - Go + Gin + ECharts"})
}

func (h *handlers) ganttUpload(c *gin.Context) {
	fileHeader, err := c.FormFile("data_file")
	if err != nil {
		c.HTML(200, "index.tmpl", gin.H{"Title": "Gantt - Go + Gin + ECharts", "Error": "请选择要上传的 CSV 或 XLSX 文件。"})
		return
	}
	dataset, err := data.ParseUploadedFile(fileHeader)
	if err != nil {
		c.HTML(200, "index.tmpl", gin.H{"Title": "Gantt - Go + Gin + ECharts", "Error": err.Error()})
		return
	}
	data.Store(dataset)
	renderMapper(c, dataset, "")
}

func (h *handlers) ganttChart(c *gin.Context) {
	datasetID := c.PostForm("dataset_id")
	dataset, ok := data.Load(datasetID)
	if !ok {
		c.HTML(200, "index.tmpl", gin.H{"Title": "Gantt - Go + Gin + ECharts", "Error": "数据已过期，请重新上传文件。"})
		return
	}

	cfg := model.MappingConfig{
		TaskCol:          c.PostForm("task_col"),
		StartCol:         c.PostForm("start_col"),
		EndCol:           c.PostForm("end_col"),
		ProjectCol:       c.PostForm("project_col"),
		ColorCol:         c.PostForm("color_col"),
		DescCol:          c.PostForm("desc_col"),
		MilestoneCol:     c.PostForm("milestone_col"),
		MilestoneDateCol: c.PostForm("milestone_date_col"),
		PlanStartCol:     c.PostForm("plan_start_col"),
		PlanEndCol:       c.PostForm("plan_end_col"),
		OwnerCol:         c.PostForm("owner_col"),
		SortByStart:      c.PostForm("sort_by_start") == "on",
		ShowTaskNumber:   c.PostForm("show_task_number") == "on",
	}

	opts := model.ChartOptions{
		HierarchicalView: true,
		ShowTaskDetails:  c.PostForm("show_task_details") == "on",
		ShowDuration:     c.PostForm("show_duration") == "on",
		DarkTheme:        c.PostForm("dark_theme") == "on",
		ChartTheme:       strings.TrimSpace(c.PostForm("chart_theme")),
		TimeGranularity:  strings.TrimSpace(c.PostForm("time_granularity")),
	}
	if opts.ChartTheme == "" {
		opts.ChartTheme = "default"
	}
	if opts.TimeGranularity == "" {
		opts.TimeGranularity = "month"
	}

	// chart_type defaults to "gantt"; extensible via the registry.
	chartType := c.PostForm("chart_type")
	if chartType == "" {
		chartType = "gantt"
	}
	builder, ok := charts.Get(chartType)
	if !ok {
		renderMapper(c, dataset, "未知图表类型: "+chartType)
		return
	}

	result, err := builder.Build(dataset, cfg, opts)
	if err != nil {
		renderMapper(c, dataset, err.Error())
		return
	}

	ganttResult, ok := result.(chartgantt.Result)
	if !ok {
		renderMapper(c, dataset, "图表数据类型错误")
		return
	}

	renderWorkspace(c, dataset, cfg, opts, ganttResult.Tasks, ganttResult.Stats, "")
}

func (h *handlers) vizHome(c *gin.Context) {
	c.HTML(200, "viz.tmpl", gin.H{
		"Title":     "通用图形 - Go + Gin + ECharts",
		"VizDefs":   viz.Definitions(),
		"VizConfig": viz.ToMap(viz.Normalize(viz.Config{})),
	})
}

func (h *handlers) vizDemo(c *gin.Context) {
	dataset, err := data.ParseCSV("viz-demo.csv", strings.NewReader(vizDemoCSV))
	if err != nil {
		c.HTML(200, "viz.tmpl", gin.H{"Title": "通用图形 - Go + Gin + ECharts", "Error": err.Error()})
		return
	}
	data.Store(dataset)
	renderVizMapper(c, dataset, viz.Config{}, "")
}

func (h *handlers) vizClear(c *gin.Context) {
	c.HTML(200, "viz.tmpl", gin.H{
		"Title":     "通用图形 - Go + Gin + ECharts",
		"VizDefs":   viz.Definitions(),
		"VizConfig": viz.ToMap(viz.Normalize(viz.Config{})),
	})
}

func (h *handlers) vizUpload(c *gin.Context) {
	fileHeader, err := c.FormFile("data_file")
	if err != nil {
		c.HTML(200, "viz.tmpl", gin.H{"Title": "通用图形 - Go + Gin + ECharts", "Error": "请选择要上传的 CSV 或 XLSX 文件。"})
		return
	}
	dataset, err := data.ParseUploadedFile(fileHeader)
	if err != nil {
		c.HTML(200, "viz.tmpl", gin.H{"Title": "通用图形 - Go + Gin + ECharts", "Error": err.Error()})
		return
	}
	data.Store(dataset)
	renderVizMapper(c, dataset, viz.Config{}, "")
}

func (h *handlers) vizValidateHierarchy(c *gin.Context) {
	datasetID := strings.TrimSpace(c.PostForm("dataset_id"))
	dataset, ok := data.Load(datasetID)
	if !ok {
		c.JSON(200, gin.H{
			"ok":       false,
			"errors":   []string{"数据已过期，请重新上传文件。"},
			"warnings": []string{},
			"stats":    gin.H{},
		})
		return
	}

	vizCfg := viz.Normalize(viz.Config{
		ChartKind:    strings.TrimSpace(c.PostForm("chart_kind")),
		NodeIDCol:    strings.TrimSpace(c.PostForm("node_id_col")),
		ParentIDCol:  strings.TrimSpace(c.PostForm("parent_id_col")),
		NameCol:      strings.TrimSpace(c.PostForm("name_col")),
		NodeValueCol: strings.TrimSpace(c.PostForm("node_value_col")),
	})

	result := viz.ValidateHierarchy(dataset, vizCfg)
	c.JSON(200, result)
}

func (h *handlers) vizChart(c *gin.Context) {
	datasetID := c.PostForm("dataset_id")
	dataset, ok := data.Load(datasetID)
	if !ok {
		c.HTML(200, "viz.tmpl", gin.H{"Title": "通用图形 - Go + Gin + ECharts", "Error": "数据已过期，请重新上传文件。", "VizDefs": viz.Definitions(), "VizConfig": viz.ToMap(viz.Normalize(viz.Config{}))})
		return
	}

	vizCfg := viz.Normalize(viz.Config{
		ChartKind:   strings.TrimSpace(c.PostForm("chart_kind")),
		Title:       strings.TrimSpace(c.PostForm("title_text")),
		SubTitle:    strings.TrimSpace(c.PostForm("subtitle_text")),
		Theme:       strings.TrimSpace(c.PostForm("chart_theme")),
		SeriesName:  strings.TrimSpace(c.PostForm("series_name")),
		Series2Name: strings.TrimSpace(c.PostForm("series2_name")),
		Series3Name: strings.TrimSpace(c.PostForm("series3_name")),
		YMetricCount: func() int {
			v := strings.TrimSpace(c.PostForm("y_metric_count"))
			n, err := strconv.Atoi(v)
			if err != nil {
				return 0
			}
			return n
		}(),
		XCol:            c.PostForm("x_col"),
		YCol:            c.PostForm("y_col"),
		Y2Col:           c.PostForm("y2_col"),
		Y3Col:           c.PostForm("y3_col"),
		YExtraCols:      c.PostFormArray("y_extra_cols"),
		NameCol:         c.PostForm("name_col"),
		ValueCol:        c.PostForm("value_col"),
		Value2Col:       c.PostForm("value2_col"),
		SizeCol:         c.PostForm("size_col"),
		SwapAxis:        c.PostForm("axis_swap") == "on",
		SmoothLine:      c.PostForm("smooth_line") == "on",
		SortMode:        strings.TrimSpace(c.PostForm("sort_mode")),
		AggregateByName: c.PostForm("aggregate_by_name") == "on",
		GaugeMode:       strings.TrimSpace(c.PostForm("gauge_mode")),
		SourceCol:       c.PostForm("source_col"),
		TargetCol:       c.PostForm("target_col"),
		LinkValueCol:    c.PostForm("link_value_col"),
		NodeIDCol:       c.PostForm("node_id_col"),
		ParentIDCol:     c.PostForm("parent_id_col"),
		NodeValueCol:    c.PostForm("node_value_col"),
	})

	payload, err := viz.Build(dataset, vizCfg)
	if err != nil {
		renderVizMapper(c, dataset, vizCfg, err.Error())
		return
	}

	renderVizChart(c, dataset, vizCfg, payload, "")
}
