// Package main provides a Gin-based Gantt chart web app with CSV/XLSX mapping.
package main

import (
	"bytes"
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type Task struct {
	TaskName      string `json:"taskName"`
	Project       string `json:"project"`
	ColorGroup    string `json:"colorGroup"`
	StartISO      string `json:"startISO"`
	EndISO        string `json:"endISO"`
	PlanStartISO  string `json:"planStartISO"`
	PlanEndISO    string `json:"planEndISO"`
	DurationDays  int    `json:"durationDays"`
	Description   string `json:"description"`
	MilestoneName string `json:"milestoneName"`
	MilestoneISO  string `json:"milestoneISO"`
	Owner         string `json:"owner"`
}

type Stats struct {
	TaskCount            int     `json:"taskCount"`
	AvgDurationDays      float64 `json:"avgDurationDays"`
	TotalDurationDay     int     `json:"totalDurationDay"`
	MaxDurationDay       int     `json:"maxDurationDay"`
	PlanTotalDurationDay int     `json:"planTotalDurationDay"`
	HasPlanTotalDuration bool    `json:"hasPlanTotalDuration"`
}

type ChartOptions struct {
	HierarchicalView bool   `json:"hierarchicalView"`
	ShowTaskDetails  bool   `json:"showTaskDetails"`
	ShowDuration     bool   `json:"showDuration"`
	DarkTheme        bool   `json:"darkTheme"`
	ChartTheme       string `json:"chartTheme"`
	TimeGranularity  string `json:"timeGranularity"`
}

type Dataset struct {
	ID      string
	Name    string
	Headers []string
	Rows    [][]string
}

type MappingDefaults struct {
	TaskCol          string
	StartCol         string
	EndCol           string
	ProjectCol       string
	ColorCol         string
	DescCol          string
	MilestoneCol     string
	MilestoneDateCol string
	PlanStartCol     string
	PlanEndCol       string
	OwnerCol         string
}

type MappingConfig struct {
	TaskCol          string
	StartCol         string
	EndCol           string
	ProjectCol       string
	ColorCol         string
	DescCol          string
	MilestoneCol     string
	MilestoneDateCol string
	PlanStartCol     string
	PlanEndCol       string
	OwnerCol         string
	SortByStart      bool
	ShowTaskNumber   bool
}

var datasetStore = struct {
	sync.RWMutex
	Data map[string]Dataset
}{
	Data: make(map[string]Dataset),
}

//go:embed static/* templates/*.tmpl
var embeddedAssets embed.FS

const demoCSV = `Project,Task,StartDate,EndDate,Description,Milestone,MilestoneDate,PlanStartDate,PlanEndDate,Owner
Website,Requirements,2026-04-01,2026-04-05,Define scope,Scope Locked,2026-04-05
Website,Design,2026-04-06,2026-04-12,UI and UX design,Design Review,2026-04-11,2026-04-06,2026-04-10,Alice
Website,Frontend,2026-04-13,2026-04-24,Build pages,,,2026-04-12,2026-04-22,Bob
Website,Backend,2026-04-14,2026-04-26,Build APIs,,,2026-04-13,2026-04-23,Chris
Website,Integration,2026-04-27,2026-05-02,Integrate FE and BE,Integration Complete,2026-05-02,2026-04-25,2026-05-01,Diana
Website,Launch,2026-05-03,2026-05-05,Release to production,Go Live,2026-05-05,2026-05-02,2026-05-04,Eric
`

func main() {
	r := gin.Default()

	staticFS, err := fs.Sub(embeddedAssets, "static")
	if err != nil {
		log.Fatal(err)
	}
	r.StaticFS("/static", http.FS(staticFS))
	r.SetHTMLTemplate(template.Must(template.ParseFS(embeddedAssets, "templates/*.tmpl")))

	r.GET("/", handleHome)
	r.GET("/demo", handleDemo)
	r.GET("/clear", handleClear)
	r.POST("/upload", handleUpload)
	r.POST("/chart", handleChart)

	log.Println("server started at http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func handleHome(c *gin.Context) {
	c.HTML(200, "index.tmpl", gin.H{
		"Title": "Gantt - Go + Gin + ECharts",
	})
}

func handleDemo(c *gin.Context) {
	dataset, err := parseCSVReader("demo.csv", strings.NewReader(demoCSV))
	if err != nil {
		c.HTML(200, "index.tmpl", gin.H{"Title": "Gantt - Go + Gin + ECharts", "Error": err.Error()})
		return
	}
	storeDataset(dataset)
	renderMapper(c, dataset, "")
}

func handleClear(c *gin.Context) {
	c.HTML(200, "index.tmpl", gin.H{
		"Title": "Gantt - Go + Gin + ECharts",
	})
}

func handleUpload(c *gin.Context) {
	fileHeader, err := c.FormFile("data_file")
	if err != nil {
		c.HTML(200, "index.tmpl", gin.H{"Title": "Gantt - Go + Gin + ECharts", "Error": "请选择要上传的 CSV 或 XLSX 文件。"})
		return
	}

	dataset, err := parseUploadedFile(fileHeader)
	if err != nil {
		c.HTML(200, "index.tmpl", gin.H{"Title": "Gantt - Go + Gin + ECharts", "Error": err.Error()})
		return
	}

	storeDataset(dataset)
	renderMapper(c, dataset, "")
}

func handleChart(c *gin.Context) {
	datasetID := c.PostForm("dataset_id")
	dataset, ok := loadDataset(datasetID)
	if !ok {
		c.HTML(200, "index.tmpl", gin.H{"Title": "Gantt - Go + Gin + ECharts", "Error": "数据已过期，请重新上传文件。"})
		return
	}

	mapping := MappingConfig{
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

	options := ChartOptions{
		HierarchicalView: true,
		ShowTaskDetails:  c.PostForm("show_task_details") == "on",
		ShowDuration:     c.PostForm("show_duration") == "on",
		DarkTheme:        c.PostForm("dark_theme") == "on",
		ChartTheme:       strings.TrimSpace(c.PostForm("chart_theme")),
		TimeGranularity:  strings.TrimSpace(c.PostForm("time_granularity")),
	}
	if options.ChartTheme == "" {
		options.ChartTheme = "default"
	}
	if options.TimeGranularity == "" {
		options.TimeGranularity = "month"
	}

	tasks, err := buildTasks(dataset, mapping)
	if err != nil {
		renderMapper(c, dataset, err.Error())
		return
	}

	stats := computeStats(tasks)
	renderWorkspace(c, dataset, mapping, options, tasks, stats, "")
}

func parseCSVReader(name string, reader io.Reader) (Dataset, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1

	records, err := csvReader.ReadAll()
	if err != nil {
		return Dataset{}, fmt.Errorf("CSV 读取失败: %w", err)
	}
	if len(records) < 2 {
		return Dataset{}, fmt.Errorf("CSV 至少需要一行表头和一行数据")
	}

	headers := make([]string, 0, len(records[0]))
	for i, col := range records[0] {
		v := strings.TrimSpace(col)
		if v == "" {
			v = fmt.Sprintf("Column_%d", i+1)
		}
		headers = append(headers, v)
	}

	rows := make([][]string, 0, len(records)-1)
	for _, row := range records[1:] {
		normalized := make([]string, len(headers))
		nonEmpty := false
		for i := range headers {
			if i < len(row) {
				normalized[i] = strings.TrimSpace(row[i])
				if normalized[i] != "" {
					nonEmpty = true
				}
			}
		}
		if nonEmpty {
			rows = append(rows, normalized)
		}
	}

	if len(rows) == 0 {
		return Dataset{}, fmt.Errorf("上传文件没有可用数据行")
	}

	return Dataset{ID: newDatasetID(), Name: name, Headers: headers, Rows: rows}, nil
}

func parseUploadedFile(fileHeader *multipart.FileHeader) (Dataset, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return Dataset{}, fmt.Errorf("无法打开上传文件")
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	switch ext {
	case ".csv":
		return parseCSVReader(fileHeader.Filename, file)
	case ".xlsx", ".xlsm", ".xltx", ".xltm":
		data, err := io.ReadAll(file)
		if err != nil {
			return Dataset{}, fmt.Errorf("读取 Excel 文件失败")
		}
		return parseXLSXReader(fileHeader.Filename, bytes.NewReader(data))
	default:
		return Dataset{}, fmt.Errorf("仅支持 CSV 或 XLSX 文件")
	}
}

func parseXLSXReader(name string, reader io.Reader) (Dataset, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return Dataset{}, fmt.Errorf("Excel 解析失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return Dataset{}, fmt.Errorf("Excel 中没有工作表")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil || len(rows) < 2 {
		return Dataset{}, fmt.Errorf("Excel 数据不足，至少需要一行表头和一行数据")
	}

	headers := make([]string, len(rows[0]))
	for i, col := range rows[0] {
		v := strings.TrimSpace(col)
		if v == "" {
			v = fmt.Sprintf("Column_%d", i+1)
		}
		headers[i] = v
	}

	dataRows := make([][]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		normalized := make([]string, len(headers))
		nonEmpty := false
		for i := range headers {
			if i < len(row) {
				normalized[i] = strings.TrimSpace(row[i])
				if normalized[i] != "" {
					nonEmpty = true
				}
			}
		}
		if nonEmpty {
			dataRows = append(dataRows, normalized)
		}
	}

	if len(dataRows) == 0 {
		return Dataset{}, fmt.Errorf("Excel 没有可用数据行")
	}

	return Dataset{ID: newDatasetID(), Name: name, Headers: headers, Rows: dataRows}, nil
}

func buildTasks(dataset Dataset, cfg MappingConfig) ([]Task, error) {
	if cfg.TaskCol == "" || cfg.StartCol == "" || cfg.EndCol == "" {
		return nil, fmt.Errorf("请至少选择任务列、开始日期列、结束日期列")
	}

	headerIndex := make(map[string]int, len(dataset.Headers))
	for i, h := range dataset.Headers {
		headerIndex[h] = i
	}

	getIdx := func(col string) int {
		if col == "" {
			return -1
		}
		idx, ok := headerIndex[col]
		if !ok {
			return -1
		}
		return idx
	}

	taskIdx := getIdx(cfg.TaskCol)
	startIdx := getIdx(cfg.StartCol)
	endIdx := getIdx(cfg.EndCol)
	projectIdx := getIdx(cfg.ProjectCol)
	colorIdx := getIdx(cfg.ColorCol)
	descIdx := getIdx(cfg.DescCol)
	mileIdx := getIdx(cfg.MilestoneCol)
	mileDateIdx := getIdx(cfg.MilestoneDateCol)
	planStartIdx := getIdx(cfg.PlanStartCol)
	planEndIdx := getIdx(cfg.PlanEndCol)
	ownerIdx := getIdx(cfg.OwnerCol)

	if taskIdx < 0 || startIdx < 0 || endIdx < 0 {
		return nil, fmt.Errorf("列映射无效，请重新选择必填列")
	}

	tasks := make([]Task, 0, len(dataset.Rows))
	for _, row := range dataset.Rows {
		taskName := cell(row, taskIdx)
		if taskName == "" {
			continue
		}

		startAt, err := parseDate(cell(row, startIdx))
		if err != nil {
			continue
		}
		endAt, err := parseDate(cell(row, endIdx))
		if err != nil {
			continue
		}
		if endAt.Before(startAt) {
			startAt, endAt = endAt, startAt
		}

		project := cell(row, projectIdx)
		if project == "" {
			project = "未分组"
		}

		colorGroup := cell(row, colorIdx)
		if colorGroup == "" {
			colorGroup = project
		}

		planStartISO := ""
		if t, err := parseDate(cell(row, planStartIdx)); err == nil {
			planStartISO = t.Format(time.RFC3339)
		}
		planEndISO := ""
		if t, err := parseDate(cell(row, planEndIdx)); err == nil {
			planEndISO = t.Format(time.RFC3339)
		}

		mileName := cell(row, mileIdx)
		mileISO := ""
		if t, err := parseDate(cell(row, mileDateIdx)); err == nil {
			mileISO = t.Format(time.RFC3339)
		} else if mileName != "" {
			mileISO = startAt.Format(time.RFC3339)
		}

		days := int(endAt.Sub(startAt).Hours()/24) + 1
		if days < 1 {
			days = 1
		}

		tasks = append(tasks, Task{
			TaskName:      taskName,
			Project:       project,
			ColorGroup:    colorGroup,
			StartISO:      startAt.Format(time.RFC3339),
			EndISO:        endAt.Format(time.RFC3339),
			PlanStartISO:  planStartISO,
			PlanEndISO:    planEndISO,
			DurationDays:  days,
			Description:   cell(row, descIdx),
			MilestoneName: mileName,
			MilestoneISO:  mileISO,
			Owner:         cell(row, ownerIdx),
		})
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("未解析出有效任务，请检查日期列格式")
	}

	if cfg.SortByStart {
		sort.SliceStable(tasks, func(i, j int) bool {
			if tasks[i].ColorGroup == tasks[j].ColorGroup {
				return tasks[i].StartISO < tasks[j].StartISO
			}
			return tasks[i].ColorGroup < tasks[j].ColorGroup
		})
	} else {
		sort.SliceStable(tasks, func(i, j int) bool {
			if tasks[i].ColorGroup == tasks[j].ColorGroup {
				return i < j
			}
			return tasks[i].ColorGroup < tasks[j].ColorGroup
		})
	}

	if cfg.ShowTaskNumber {
		for i := range tasks {
			tasks[i].TaskName = fmt.Sprintf("%02d  %s", i+1, tasks[i].TaskName)
		}
	}

	return tasks, nil
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func guessColumn(headers []string, keys ...string) string {
	lower := make([]string, len(headers))
	for i, h := range headers {
		lower[i] = strings.ToLower(h)
	}
	for _, key := range keys {
		for i := range headers {
			if strings.Contains(lower[i], strings.ToLower(key)) {
				return headers[i]
			}
		}
	}
	return ""
}

func inferDefaults(headers []string) MappingDefaults {
	return MappingDefaults{
		TaskCol:          guessColumn(headers, "task", "任务", "name"),
		StartCol:         guessColumn(headers, "start", "开始", "date"),
		EndCol:           guessColumn(headers, "end", "结束", "date"),
		ProjectCol:       guessColumn(headers, "project", "项目", "group"),
		ColorCol:         guessColumn(headers, "project", "分类", "group", "color"),
		DescCol:          guessColumn(headers, "desc", "description", "detail", "说明"),
		MilestoneCol:     guessColumn(headers, "milestone", "里程碑"),
		MilestoneDateCol: guessColumn(headers, "milestone date", "milestonedate", "里程碑日期"),
		PlanStartCol:     guessColumn(headers, "planstart", "计划开始", "baseline start"),
		PlanEndCol:       guessColumn(headers, "planend", "计划结束", "baseline end"),
		OwnerCol:         guessColumn(headers, "owner", "负责人", "assignee"),
	}
}

func mappingFromDefaults(def MappingDefaults) MappingConfig {
	return MappingConfig{
		TaskCol:          def.TaskCol,
		StartCol:         def.StartCol,
		EndCol:           def.EndCol,
		ProjectCol:       def.ProjectCol,
		ColorCol:         def.ColorCol,
		DescCol:          def.DescCol,
		MilestoneCol:     def.MilestoneCol,
		MilestoneDateCol: def.MilestoneDateCol,
		PlanStartCol:     def.PlanStartCol,
		PlanEndCol:       def.PlanEndCol,
		OwnerCol:         def.OwnerCol,
	}
}

func defaultChartOptions() ChartOptions {
	return ChartOptions{
		HierarchicalView: true,
		ShowTaskDetails:  true,
		ShowDuration:     true,
		DarkTheme:        false,
		ChartTheme:       "default",
		TimeGranularity:  "month",
	}
}

func renderWorkspace(
	c *gin.Context,
	dataset Dataset,
	mapping MappingConfig,
	options ChartOptions,
	tasks []Task,
	stats Stats,
	errMsg string,
) {
	preview := dataset.Rows
	if len(preview) > 8 {
		preview = preview[:8]
	}

	view := gin.H{
		"Title":     "Gantt - 列映射与图表",
		"DatasetID": dataset.ID,
		"FileName":  dataset.Name,
		"Headers":   dataset.Headers,
		"Preview":   preview,
		"Mapping":   mapping,
		"Options":   options,
		"Error":     errMsg,
		"HasChart":  len(tasks) > 0,
	}

	if len(tasks) > 0 {
		tasksJSON, _ := json.Marshal(tasks)
		statsJSON, _ := json.Marshal(stats)
		optionsJSON, _ := json.Marshal(options)
		view["TasksJSON"] = template.JS(string(tasksJSON))
		view["StatsJSON"] = template.JS(string(statsJSON))
		view["OptionsJSON"] = template.JS(string(optionsJSON))
	}

	c.HTML(200, "index.tmpl", view)
}

func renderMapper(c *gin.Context, dataset Dataset, errMsg string) {
	defs := inferDefaults(dataset.Headers)
	renderWorkspace(c, dataset, mappingFromDefaults(defs), defaultChartOptions(), nil, Stats{}, errMsg)
}

func storeDataset(dataset Dataset) {
	datasetStore.Lock()
	datasetStore.Data[dataset.ID] = dataset
	datasetStore.Unlock()
}

func loadDataset(id string) (Dataset, bool) {
	datasetStore.RLock()
	v, ok := datasetStore.Data[id]
	datasetStore.RUnlock()
	return v, ok
}

func newDatasetID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func parseDate(value string) (time.Time, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"01-02-06",
		"1-2-06",
		"01/02/06",
		"1/2/06",
		"01.02.06",
		"1.2.06",
		"01-02-2006",
		"1-2-2006",
		"01/02/2006",
		"1/2/2006",
		"01-02-06 15:04",
		"1-2-06 15:04",
		"01-02-06 15:04:05",
		"1-2-06 15:04:05",
		"2006-01-02",
		"2006-1-2",
		"2006/01/02",
		"2006/1/2",
		"2006.01.02",
		"2006.1.2",
		"2006年01月02日",
		"2006年1月2日",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-1-2 15:04:05",
		"2006-1-2 15:04",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/1/2 15:04:05",
		"2006/1/2 15:04",
		"1/2/2006",
		"1/2/2006 15:04",
		"1/2/2006 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, v, time.Local); err == nil {
			return t, nil
		}
	}

	// Excel serial date (e.g. 45291 or 45291.5) is common in xlsx raw values.
	if serial, err := strconv.ParseFloat(v, 64); err == nil {
		if t, err := excelize.ExcelDateToTime(serial, false); err == nil {
			return t, nil
		}
		if t, err := excelize.ExcelDateToTime(serial, true); err == nil {
			return t, nil
		}
	}

	if unixMS, err := strconv.ParseInt(v, 10, 64); err == nil && len(v) >= 12 {
		return time.UnixMilli(unixMS), nil
	}
	return time.Time{}, fmt.Errorf("unsupported date format: %s", value)
}

func computeStats(tasks []Task) Stats {
	if len(tasks) == 0 {
		return Stats{}
	}
	totalTaskDuration := 0
	maxDur := 0

	actualMinStart := time.Time{}
	actualMaxEnd := time.Time{}

	planMinStart := time.Time{}
	planMaxEnd := time.Time{}
	hasPlan := false

	for _, t := range tasks {
		totalTaskDuration += t.DurationDays
		if t.DurationDays > maxDur {
			maxDur = t.DurationDays
		}

		startAt, startErr := time.Parse(time.RFC3339, t.StartISO)
		endAt, endErr := time.Parse(time.RFC3339, t.EndISO)
		if startErr == nil && endErr == nil {
			if endAt.Before(startAt) {
				startAt, endAt = endAt, startAt
			}
			if actualMinStart.IsZero() || startAt.Before(actualMinStart) {
				actualMinStart = startAt
			}
			if actualMaxEnd.IsZero() || endAt.After(actualMaxEnd) {
				actualMaxEnd = endAt
			}
		}

		planStartAt, planStartErr := time.Parse(time.RFC3339, t.PlanStartISO)
		planEndAt, planEndErr := time.Parse(time.RFC3339, t.PlanEndISO)
		if planStartErr == nil && planEndErr == nil {
			if planEndAt.Before(planStartAt) {
				planStartAt, planEndAt = planEndAt, planStartAt
			}
			if !hasPlan {
				planMinStart = planStartAt
				planMaxEnd = planEndAt
				hasPlan = true
			} else {
				if planStartAt.Before(planMinStart) {
					planMinStart = planStartAt
				}
				if planEndAt.After(planMaxEnd) {
					planMaxEnd = planEndAt
				}
			}
		}
	}

	actualSpan := 0
	if !actualMinStart.IsZero() && !actualMaxEnd.IsZero() {
		actualSpan = int(actualMaxEnd.Sub(actualMinStart).Hours()/24) + 1
		if actualSpan < 1 {
			actualSpan = 1
		}
	}

	planSpan := 0
	if hasPlan {
		planSpan = int(planMaxEnd.Sub(planMinStart).Hours()/24) + 1
		if planSpan < 1 {
			planSpan = 1
		}
	}

	return Stats{
		TaskCount:            len(tasks),
		AvgDurationDays:      float64(totalTaskDuration) / float64(len(tasks)),
		TotalDurationDay:     actualSpan,
		MaxDurationDay:       maxDur,
		PlanTotalDurationDay: planSpan,
		HasPlanTotalDuration: hasPlan,
	}
}
