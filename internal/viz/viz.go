// Package viz contains the generic ECharts visualization registry and builders.
package viz

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"gantt/internal/data"
	"gantt/internal/model"
)

// Config stores the generic visualization form selections.
type Config struct {
	ChartKind       string
	Title           string
	SubTitle        string
	Theme           string
	SeriesName      string
	Series2Name     string
	Series3Name     string
	YMetricCount    int
	XCol            string
	YCol            string
	Y2Col           string
	Y3Col           string
	YExtraCols      []string
	NameCol         string
	ValueCol        string
	Value2Col       string
	SizeCol         string
	SwapAxis        bool
	SmoothLine      bool
	SortMode        string
	AggregateByName bool
	GaugeMode       string
	SourceCol       string
	TargetCol       string
	LinkValueCol    string
	NodeIDCol       string
	ParentIDCol     string
	NodeValueCol    string
}

// HierarchyValidation summarizes hierarchy mapping/data checks for tree-like charts.
type HierarchyValidation struct {
	OK       bool           `json:"ok"`
	Errors   []string       `json:"errors"`
	Warnings []string       `json:"warnings"`
	Stats    map[string]int `json:"stats"`
}

// Definition describes a visual builder for the UI.
type Definition struct {
	Kind        string
	Name        string
	Family      string
	Description string
	Hint        string
}

// Builder creates a chart payload from a dataset and viz config.
type Builder interface {
	Definition() Definition
	Build(dataset model.Dataset, cfg Config) (map[string]any, error)
}

type builder struct {
	def   Definition
	build func(dataset model.Dataset, cfg Config) (map[string]any, error)
}

func (b builder) Definition() Definition { return b.def }
func (b builder) Build(dataset model.Dataset, cfg Config) (map[string]any, error) {
	return b.build(dataset, cfg)
}

var registry = map[string]Builder{}
var order []string

func register(def Definition, fn func(dataset model.Dataset, cfg Config) (map[string]any, error)) {
	registry[def.Kind] = builder{def: def, build: fn}
	order = append(order, def.Kind)
}

// Get returns a registered builder by chart kind.
func Get(kind string) (Builder, bool) {
	b, ok := registry[kind]
	return b, ok
}

// Definitions returns all builder metadata in registration order.
func Definitions() []Definition {
	out := make([]Definition, 0, len(order))
	for _, kind := range order {
		out = append(out, registry[kind].Definition())
	}
	return out
}

// Families groups definitions by family for the UI.
func Families() map[string][]Definition {
	out := map[string][]Definition{}
	for _, def := range Definitions() {
		out[def.Family] = append(out[def.Family], def)
	}
	return out
}

// Normalize applies defaults to config.
func Normalize(cfg Config) Config {
	if cfg.ChartKind == "" {
		cfg.ChartKind = "bar"
	}
	if cfg.Theme == "" {
		cfg.Theme = "default"
	}
	if cfg.SeriesName == "" {
		cfg.SeriesName = "Series 1"
	}
	if cfg.Series2Name == "" {
		cfg.Series2Name = "Series 2"
	}
	if cfg.Series3Name == "" {
		cfg.Series3Name = "Series 3"
	}
	if cfg.SortMode == "" {
		cfg.SortMode = "none"
	}
	if cfg.GaugeMode == "" {
		cfg.GaugeMode = "avg"
	}
	cfg.YExtraCols = normalizeCols(cfg.YExtraCols)
	selectedCount := len(selectedYCols(cfg))
	if cfg.YMetricCount < 1 {
		cfg.YMetricCount = selectedCount
	}
	if cfg.YMetricCount < 1 {
		cfg.YMetricCount = 1
	}
	if selectedCount > cfg.YMetricCount {
		cfg.YMetricCount = selectedCount
	}
	if cfg.YMetricCount > 10 {
		cfg.YMetricCount = 10
	}
	return cfg
}

// Merge overlays non-zero values from override onto base.
func Merge(base, override Config) Config {
	out := base
	assign := func(dst *string, src string) {
		if src != "" {
			*dst = src
		}
	}
	assign(&out.ChartKind, override.ChartKind)
	assign(&out.Title, override.Title)
	assign(&out.SubTitle, override.SubTitle)
	assign(&out.Theme, override.Theme)
	assign(&out.SeriesName, override.SeriesName)
	assign(&out.Series2Name, override.Series2Name)
	assign(&out.Series3Name, override.Series3Name)
	if override.YMetricCount > 0 {
		out.YMetricCount = override.YMetricCount
	}
	assign(&out.XCol, override.XCol)
	assign(&out.YCol, override.YCol)
	assign(&out.Y2Col, override.Y2Col)
	assign(&out.Y3Col, override.Y3Col)
	if override.YExtraCols != nil {
		out.YExtraCols = normalizeCols(override.YExtraCols)
	}
	assign(&out.NameCol, override.NameCol)
	assign(&out.ValueCol, override.ValueCol)
	assign(&out.Value2Col, override.Value2Col)
	assign(&out.SizeCol, override.SizeCol)
	if override.SwapAxis {
		out.SwapAxis = true
	}
	assign(&out.SortMode, override.SortMode)
	assign(&out.GaugeMode, override.GaugeMode)
	assign(&out.SourceCol, override.SourceCol)
	assign(&out.TargetCol, override.TargetCol)
	assign(&out.LinkValueCol, override.LinkValueCol)
	assign(&out.NodeIDCol, override.NodeIDCol)
	assign(&out.ParentIDCol, override.ParentIDCol)
	assign(&out.NodeValueCol, override.NodeValueCol)
	if override.SmoothLine {
		out.SmoothLine = true
	}
	if override.AggregateByName {
		out.AggregateByName = true
	}
	return Normalize(out)
}

// InferDefaults creates a best-effort config from headers.
func InferDefaults(headers []string) Config {
	return Normalize(Config{
		ChartKind:    "bar",
		Theme:        "default",
		SeriesName:   "指标A",
		Series2Name:  "指标B",
		Series3Name:  "指标C",
		SortMode:     "none",
		GaugeMode:    "avg",
		XCol:         inferHeader(headers, "month", "date", "category", "name", "x"),
		YCol:         inferHeader(headers, "revenue", "value", "profit", "amount", "y"),
		Y2Col:        inferHeader(headers, "cost", "share", "value2", "y2"),
		Y3Col:        inferHeader(headers, "profit", "value3", "y3"),
		NameCol:      inferHeader(headers, "category", "name", "label"),
		ValueCol:     inferHeader(headers, "share", "revenue", "value", "amount"),
		Value2Col:    inferHeader(headers, "cost", "profit", "value2"),
		SizeCol:      inferHeader(headers, "scattersize", "size", "bubble"),
		SourceCol:    inferHeader(headers, "source", "from"),
		TargetCol:    inferHeader(headers, "target", "to"),
		LinkValueCol: inferHeader(headers, "linkvalue", "weight", "flow", "value"),
		NodeIDCol:    inferHeader(headers, "nodeid", "id"),
		ParentIDCol:  inferHeader(headers, "parentid", "parent"),
		NodeValueCol: inferHeader(headers, "nodevalue", "value", "amount"),
	})
}

// ToMap converts config to a template-friendly map.
func ToMap(cfg Config) map[string]any {
	cfg = Normalize(cfg)
	return map[string]any{
		"ChartKind":       cfg.ChartKind,
		"Title":           cfg.Title,
		"SubTitle":        cfg.SubTitle,
		"Theme":           cfg.Theme,
		"SeriesName":      cfg.SeriesName,
		"Series2Name":     cfg.Series2Name,
		"Series3Name":     cfg.Series3Name,
		"YMetricCount":    strconv.Itoa(cfg.YMetricCount),
		"XCol":            cfg.XCol,
		"YCol":            cfg.YCol,
		"Y2Col":           cfg.Y2Col,
		"Y3Col":           cfg.Y3Col,
		"YExtraCols":      cfg.YExtraCols,
		"NameCol":         cfg.NameCol,
		"ValueCol":        cfg.ValueCol,
		"Value2Col":       cfg.Value2Col,
		"SizeCol":         cfg.SizeCol,
		"SwapAxis":        cfg.SwapAxis,
		"SmoothLine":      cfg.SmoothLine,
		"SortMode":        cfg.SortMode,
		"AggregateByName": cfg.AggregateByName,
		"GaugeMode":       cfg.GaugeMode,
		"SourceCol":       cfg.SourceCol,
		"TargetCol":       cfg.TargetCol,
		"LinkValueCol":    cfg.LinkValueCol,
		"NodeIDCol":       cfg.NodeIDCol,
		"ParentIDCol":     cfg.ParentIDCol,
		"NodeValueCol":    cfg.NodeValueCol,
	}
}

func inferHeader(headers []string, keys ...string) string {
	lower := make([]string, len(headers))
	for i, h := range headers {
		lower[i] = strings.ToLower(h)
	}
	for _, key := range keys {
		needle := strings.ToLower(key)
		for i := range headers {
			if strings.Contains(lower[i], needle) {
				return headers[i]
			}
		}
	}
	return ""
}

func headerIndex(headers []string) map[string]int {
	out := make(map[string]int, len(headers))
	for i, h := range headers {
		out[h] = i
	}
	return out
}

func idx(index map[string]int, col string) int {
	if col == "" {
		return -1
	}
	v, ok := index[col]
	if !ok {
		return -1
	}
	return v
}

func parseFloat(value string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}

func normalizeCols(cols []string) []string {
	if len(cols) == 0 {
		return nil
	}
	out := make([]string, 0, len(cols))
	seen := map[string]struct{}{}
	for _, c := range cols {
		v := strings.TrimSpace(c)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func selectedYCols(cfg Config) []string {
	cols := make([]string, 0, 3+len(cfg.YExtraCols))
	if strings.TrimSpace(cfg.YCol) != "" {
		cols = append(cols, strings.TrimSpace(cfg.YCol))
	}
	if strings.TrimSpace(cfg.Y2Col) != "" {
		cols = append(cols, strings.TrimSpace(cfg.Y2Col))
	}
	if strings.TrimSpace(cfg.Y3Col) != "" {
		cols = append(cols, strings.TrimSpace(cfg.Y3Col))
	}
	cols = append(cols, normalizeCols(cfg.YExtraCols)...)
	return normalizeCols(cols)
}

func buildCartesian(dataset model.Dataset, cfg Config) (map[string]any, error) {
	index := headerIndex(dataset.Headers)
	xIdx := idx(index, cfg.XCol)
	yCols := selectedYCols(cfg)
	if xIdx < 0 || len(yCols) == 0 {
		return nil, fmt.Errorf("请为该图形选择 X 轴列和 Y 轴列")
	}

	yIndices := make([]int, 0, len(yCols))
	resolvedYCols := make([]string, 0, len(yCols))
	for _, col := range yCols {
		colIdx := idx(index, col)
		if colIdx >= 0 {
			yIndices = append(yIndices, colIdx)
			resolvedYCols = append(resolvedYCols, col)
		}
	}
	if len(yIndices) == 0 {
		return nil, fmt.Errorf("未匹配到可用的 Y 轴列，请确认列名")
	}

	xAxis := make([]string, 0, len(dataset.Rows))
	seriesData := make([][]float64, len(yIndices))
	for i := range seriesData {
		seriesData[i] = make([]float64, 0, len(dataset.Rows))
	}

	for _, row := range dataset.Rows {
		x := data.Cell(row, xIdx)
		if x == "" {
			continue
		}
		values := make([]float64, len(yIndices))
		valid := true
		for i, colIdx := range yIndices {
			v, err := parseFloat(data.Cell(row, colIdx))
			if err != nil {
				valid = false
				break
			}
			values[i] = v
		}
		if !valid {
			continue
		}
		xAxis = append(xAxis, x)
		for i := range values {
			seriesData[i] = append(seriesData[i], values[i])
		}
	}
	if len(seriesData) == 0 || len(seriesData[0]) == 0 {
		return nil, fmt.Errorf("未解析到可用数值，请确认 Y 轴列为数值")
	}

	if cfg.SortMode == "asc" || cfg.SortMode == "desc" {
		order := make([]int, len(seriesData[0]))
		for i := range order {
			order[i] = i
		}
		multiplier := 1.0
		if cfg.SortMode == "desc" {
			multiplier = -1.0
		}
		sort.SliceStable(order, func(i, j int) bool {
			return multiplier*seriesData[0][order[i]] < multiplier*seriesData[0][order[j]]
		})
		sortedX := make([]string, 0, len(xAxis))
		sortedSeries := make([][]float64, len(seriesData))
		for i := range sortedSeries {
			sortedSeries[i] = make([]float64, 0, len(seriesData[i]))
		}
		for _, pos := range order {
			sortedX = append(sortedX, xAxis[pos])
			for i := range sortedSeries {
				sortedSeries[i] = append(sortedSeries[i], seriesData[i][pos])
			}
		}
		xAxis = sortedX
		seriesData = sortedSeries
	}

	seriesNames := make([]string, 0, len(resolvedYCols))
	for i, col := range resolvedYCols {
		switch i {
		case 0:
			seriesNames = append(seriesNames, cfg.SeriesName)
		case 1:
			seriesNames = append(seriesNames, cfg.Series2Name)
		case 2:
			seriesNames = append(seriesNames, cfg.Series3Name)
		default:
			seriesNames = append(seriesNames, col)
		}
	}

	seriesDefs := make([]map[string]any, 0, len(seriesData))
	for i := range seriesData {
		if len(seriesData[i]) != len(seriesData[0]) {
			continue
		}
		item := map[string]any{"name": seriesNames[i], "data": seriesData[i], "smooth": cfg.SmoothLine}
		seriesDefs = append(seriesDefs, item)
	}
	if len(seriesDefs) == 0 {
		return nil, fmt.Errorf("未解析到可用系列数据，请检查 Y 轴列")
	}

	return map[string]any{
		"kind":     cfg.ChartKind,
		"title":    map[string]any{"text": cfg.Title, "subtext": cfg.SubTitle},
		"xAxis":    xAxis,
		"series":   seriesDefs,
		"swapAxis": cfg.SwapAxis,
	}, nil
}

func buildScatter(dataset model.Dataset, cfg Config) (map[string]any, error) {
	index := headerIndex(dataset.Headers)
	xIdx := idx(index, cfg.XCol)
	yIdx := idx(index, cfg.YCol)
	sizeIdx := idx(index, cfg.SizeCol)
	if xIdx < 0 || yIdx < 0 {
		return nil, fmt.Errorf("请为散点图选择 X 轴列和 Y 轴列")
	}

	points := make([]map[string]any, 0, len(dataset.Rows))

	for _, row := range dataset.Rows {
		x, err := parseFloat(data.Cell(row, xIdx))
		if err != nil {
			continue
		}
		y, err := parseFloat(data.Cell(row, yIdx))
		if err != nil {
			continue
		}
		size := 12.0
		if sizeIdx >= 0 {
			if sv, err := parseFloat(data.Cell(row, sizeIdx)); err == nil {
				size = math.Max(6, math.Min(44, sv))
			}
		}

		points = append(points, map[string]any{"value": []float64{x, y, size}})
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("未解析到可用散点数据，请确认 X/Y 列为数值")
	}

	return map[string]any{
		"kind":       cfg.ChartKind,
		"title":      map[string]any{"text": cfg.Title, "subtext": cfg.SubTitle},
		"xName":      cfg.XCol,
		"yName":      cfg.YCol,
		"seriesName": cfg.SeriesName,
		"sizeName":   cfg.SizeCol,
		"points":     points,
	}, nil
}

func buildItems(dataset model.Dataset, cfg Config) (map[string]any, error) {
	index := headerIndex(dataset.Headers)
	nameIdx := idx(index, cfg.NameCol)
	valueIdx := idx(index, cfg.ValueCol)
	if nameIdx < 0 || valueIdx < 0 {
		return nil, fmt.Errorf("请为该图形选择名称列和值列")
	}
	agg := map[string]float64{}
	items := make([]map[string]any, 0, len(dataset.Rows))
	for _, row := range dataset.Rows {
		name := data.Cell(row, nameIdx)
		if name == "" {
			continue
		}
		v, err := parseFloat(data.Cell(row, valueIdx))
		if err != nil {
			continue
		}
		if cfg.AggregateByName {
			agg[name] += v
		} else {
			items = append(items, map[string]any{"name": name, "value": v})
		}
	}
	if cfg.AggregateByName {
		for name, value := range agg {
			items = append(items, map[string]any{"name": name, "value": value})
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("未解析到可用数值，请确认值列为数值")
	}
	return map[string]any{
		"kind":       cfg.ChartKind,
		"title":      map[string]any{"text": cfg.Title, "subtext": cfg.SubTitle},
		"seriesName": cfg.SeriesName,
		"items":      items,
	}, nil
}

func buildRadar(dataset model.Dataset, cfg Config) (map[string]any, error) {
	index := headerIndex(dataset.Headers)
	nameIdx := idx(index, cfg.NameCol)
	if nameIdx < 0 {
		return nil, fmt.Errorf("请为雷达图选择指标名称列（name_col）")
	}

	yCols := selectedYCols(cfg)
	if len(yCols) < 1 {
		return nil, fmt.Errorf("雷达图至少需要一个数值列")
	}

	yIndices := make([]int, 0, len(yCols))
	resolvedYCols := make([]string, 0, len(yCols))
	for _, col := range yCols {
		colIdx := idx(index, col)
		if colIdx >= 0 {
			yIndices = append(yIndices, colIdx)
			resolvedYCols = append(resolvedYCols, col)
		}
	}
	if len(yIndices) < 1 {
		return nil, fmt.Errorf("未找到可用的数值列")
	}

	// Build indicators from name_col and collect all values for each series
	indicators := []map[string]any{}
	seriesDataList := make([][]float64, len(yIndices)) // seriesDataList[i] holds values for Y column i
	for i := range seriesDataList {
		seriesDataList[i] = make([]float64, 0)
	}

	maxVals := make([]float64, len(yIndices)) // Track max per column for better scaling

	for _, row := range dataset.Rows {
		name := strings.TrimSpace(data.Cell(row, nameIdx))
		if name == "" {
			continue
		}

		rowValues := make([]float64, len(yIndices))
		valid := true
		for i, colIdx := range yIndices {
			v, err := parseFloat(data.Cell(row, colIdx))
			if err != nil {
				valid = false
				break
			}
			rowValues[i] = v
			if rowValues[i] > maxVals[i] {
				maxVals[i] = rowValues[i]
			}
		}
		if !valid {
			continue
		}

		// Add indicator
		indicators = append(indicators, map[string]any{"name": name})

		// Append values to each series
		for i, val := range rowValues {
			seriesDataList[i] = append(seriesDataList[i], val)
		}
	}

	if len(indicators) == 0 {
		return nil, fmt.Errorf("未解析到可用的指标行数据")
	}

	// Calculate max for each indicator and update indicators
	for i := range indicators {
		// Find the max value at this position across all series
		maxAtIdx := 0.0
		for seriesIdx := range seriesDataList {
			if i < len(seriesDataList[seriesIdx]) && seriesDataList[seriesIdx][i] > maxAtIdx {
				maxAtIdx = seriesDataList[seriesIdx][i]
			}
		}
		maxVal := math.Ceil(maxAtIdx*1.2 + 1)
		if maxVal < 10 {
			maxVal = 10
		}
		indicators[i]["max"] = maxVal
	}

	// Build series data: each Y column becomes a data series
	series := make([]map[string]any, len(yIndices))
	seriesNames := []string{cfg.SeriesName, cfg.Series2Name, cfg.Series3Name}
	for i, col := range resolvedYCols {
		seriesName := col
		if i < len(seriesNames) && strings.TrimSpace(seriesNames[i]) != "" {
			seriesName = seriesNames[i]
		}

		series[i] = map[string]any{
			"name": seriesName,
			"type": "radar",
			"data": []map[string]any{
				{
					"value": seriesDataList[i],
					"name":  seriesName,
				},
			},
		}
	}

	return map[string]any{
		"kind":       cfg.ChartKind,
		"title":      map[string]any{"text": cfg.Title, "subtext": cfg.SubTitle},
		"indicators": indicators,
		"series":     series,
	}, nil
}

func buildGauge(dataset model.Dataset, cfg Config) (map[string]any, error) {
	index := headerIndex(dataset.Headers)
	valueIdx := idx(index, cfg.ValueCol)
	if valueIdx < 0 {
		return nil, fmt.Errorf("请为仪表盘选择数值列")
	}
	vals := []float64{}
	for _, row := range dataset.Rows {
		if v, err := parseFloat(data.Cell(row, valueIdx)); err == nil {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return nil, fmt.Errorf("未解析到可用数值，请确认数值列为数字")
	}
	calc := vals[0]
	sum := 0.0
	minV, maxV := vals[0], vals[0]
	for _, v := range vals {
		sum += v
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	switch cfg.GaugeMode {
	case "min":
		calc = minV
	case "max":
		calc = maxV
	case "first":
		calc = vals[0]
	default:
		calc = sum / float64(len(vals))
	}
	gaugeMax := math.Ceil(maxV*1.2 + 1)
	if gaugeMax < 100 {
		gaugeMax = 100
	}
	return map[string]any{
		"kind":       cfg.ChartKind,
		"title":      map[string]any{"text": cfg.Title, "subtext": cfg.SubTitle},
		"seriesName": cfg.SeriesName,
		"value":      calc,
		"max":        gaugeMax,
	}, nil
}

func buildRelation(dataset model.Dataset, cfg Config) (map[string]any, error) {
	index := headerIndex(dataset.Headers)
	sourceIdx := idx(index, cfg.SourceCol)
	targetIdx := idx(index, cfg.TargetCol)
	valueIdx := idx(index, cfg.LinkValueCol)
	if sourceIdx < 0 || targetIdx < 0 {
		return nil, fmt.Errorf("请为该图形选择来源列和目标列")
	}
	nodeSet := map[string]struct{}{}
	degree := map[string]float64{}
	nodes := make([]map[string]any, 0)
	links := make([]map[string]any, 0)
	for _, row := range dataset.Rows {
		source := strings.TrimSpace(data.Cell(row, sourceIdx))
		target := strings.TrimSpace(data.Cell(row, targetIdx))
		if source == "" || target == "" {
			continue
		}
		value := 1.0
		if valueIdx >= 0 {
			if v, err := parseFloat(data.Cell(row, valueIdx)); err == nil {
				value = v
			}
		}
		if _, ok := nodeSet[source]; !ok {
			nodeSet[source] = struct{}{}
			nodes = append(nodes, map[string]any{"name": source})
		}
		if _, ok := nodeSet[target]; !ok {
			nodeSet[target] = struct{}{}
			nodes = append(nodes, map[string]any{"name": target})
		}
		degree[source] += value
		degree[target] += value
		links = append(links, map[string]any{"source": source, "target": target, "value": value})
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("未解析到关系数据，请确认来源/目标列")
	}
	if cfg.ChartKind == "graph" || cfg.ChartKind == "chord" {
		for i := range nodes {
			name := fmt.Sprintf("%v", nodes[i]["name"])
			size := 10 + math.Sqrt(math.Max(1, degree[name]))*2.6
			if size > 58 {
				size = 58
			}
			nodes[i]["value"] = degree[name]
			nodes[i]["symbolSize"] = size
		}
	}
	relationSubTitle := cfg.SubTitle
	if relationSubTitle == "" && cfg.ChartKind == "chord" {
		relationSubTitle = "和弦风格关系布局"
	}
	return map[string]any{
		"kind":       cfg.ChartKind,
		"title":      map[string]any{"text": cfg.Title, "subtext": relationSubTitle},
		"seriesName": cfg.SeriesName,
		"nodes":      nodes,
		"links":      links,
	}, nil
}

func buildHierarchy(dataset model.Dataset, cfg Config) (map[string]any, error) {
	validation := ValidateHierarchy(dataset, cfg)
	if len(validation.Errors) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(validation.Errors, "；"))
	}

	index := headerIndex(dataset.Headers)
	nodeIDIdx := idx(index, cfg.NodeIDCol)
	parentIDIdx := idx(index, cfg.ParentIDCol)
	nodeValueIdx := idx(index, cfg.NodeValueCol)
	nameIdx := idx(index, cfg.NameCol)
	if nodeIDIdx < 0 {
		nodeIDIdx = nameIdx
	}
	if nodeIDIdx < 0 || parentIDIdx < 0 {
		return nil, fmt.Errorf("请为该图形选择节点ID列与父节点列")
	}
	if nodeIDIdx == parentIDIdx {
		return nil, fmt.Errorf("节点ID列与父节点列不能相同，否则会形成自循环")
	}
	type treeNode struct {
		ID       string
		Name     string
		Value    float64
		Children []*treeNode
	}
	nodesByID := map[string]*treeNode{}
	getNode := func(id string) *treeNode {
		if n, ok := nodesByID[id]; ok {
			return n
		}
		n := &treeNode{ID: id, Name: id}
		nodesByID[id] = n
		return n
	}
	roots := map[string]*treeNode{}
	for i, row := range dataset.Rows {
		id := strings.TrimSpace(data.Cell(row, nodeIDIdx))
		if id == "" {
			id = fmt.Sprintf("node-%d", i+1)
		}
		parent := strings.TrimSpace(data.Cell(row, parentIDIdx))
		if parent == id {
			return nil, fmt.Errorf("第 %d 行节点ID与父节点相同（%s），请修正映射或数据", i+1, id)
		}
		name := ""
		if nameIdx >= 0 {
			name = strings.TrimSpace(data.Cell(row, nameIdx))
		}
		if name == "" {
			name = id
		}
		node := getNode(id)
		node.Name = name
		if nodeValueIdx >= 0 {
			if v, err := parseFloat(data.Cell(row, nodeValueIdx)); err == nil {
				node.Value = v
			}
		}
		if parent == "" {
			roots[id] = node
			continue
		}
		p := getNode(parent)
		exists := false
		for _, ch := range p.Children {
			if ch.ID == node.ID {
				exists = true
				break
			}
		}
		if !exists {
			p.Children = append(p.Children, node)
		}
		delete(roots, id)
	}
	var toPayload func(n *treeNode, stack map[string]bool) (map[string]any, error)
	toPayload = func(n *treeNode, stack map[string]bool) (map[string]any, error) {
		if stack[n.ID] {
			return nil, fmt.Errorf("检测到循环父子关系，节点ID: %s", n.ID)
		}
		stack[n.ID] = true
		defer delete(stack, n.ID)

		out := map[string]any{"name": n.Name}
		if n.Value != 0 {
			out["value"] = n.Value
		}
		if len(n.Children) > 0 {
			children := make([]map[string]any, 0, len(n.Children))
			for _, ch := range n.Children {
				child, err := toPayload(ch, stack)
				if err != nil {
					return nil, err
				}
				children = append(children, child)
			}
			out["children"] = children
		}
		return out, nil
	}
	rootNodes := make([]map[string]any, 0)
	for _, n := range roots {
		payloadNode, err := toPayload(n, map[string]bool{})
		if err != nil {
			return nil, err
		}
		rootNodes = append(rootNodes, payloadNode)
	}
	if len(rootNodes) == 0 {
		for _, n := range nodesByID {
			payloadNode, err := toPayload(n, map[string]bool{})
			if err != nil {
				return nil, err
			}
			rootNodes = append(rootNodes, payloadNode)
		}
	}
	if len(rootNodes) == 0 {
		return nil, fmt.Errorf("未解析到树结构数据")
	}
	root := map[string]any{"name": cfg.SeriesName, "children": rootNodes}
	if cfg.ChartKind == "tree" && len(rootNodes) == 1 {
		root = rootNodes[0]
	}
	return map[string]any{
		"kind":       cfg.ChartKind,
		"title":      map[string]any{"text": cfg.Title, "subtext": cfg.SubTitle},
		"seriesName": cfg.SeriesName,
		"tree":       root,
	}, nil
}

// ValidateHierarchy checks tree/treemap mapping consistency and basic topology issues.
func ValidateHierarchy(dataset model.Dataset, cfg Config) HierarchyValidation {
	res := HierarchyValidation{
		OK:       true,
		Errors:   []string{},
		Warnings: []string{},
		Stats:    map[string]int{},
	}

	index := headerIndex(dataset.Headers)
	nodeIDIdx := idx(index, cfg.NodeIDCol)
	parentIDIdx := idx(index, cfg.ParentIDCol)
	nameIdx := idx(index, cfg.NameCol)
	if nodeIDIdx < 0 {
		nodeIDIdx = nameIdx
	}
	if nodeIDIdx < 0 || parentIDIdx < 0 {
		res.OK = false
		res.Errors = append(res.Errors, "请为该图形选择节点ID列与父节点列")
		return res
	}
	if nodeIDIdx == parentIDIdx {
		res.OK = false
		res.Errors = append(res.Errors, "节点ID列与父节点列不能相同，否则会形成自循环")
		return res
	}

	type edgeRow struct {
		line   int
		nodeID string
		parent string
	}

	idRows := map[string][]int{}
	parentRefs := map[string][]int{}
	edges := make([]edgeRow, 0, len(dataset.Rows))
	parentByChild := map[string]string{}

	for i, row := range dataset.Rows {
		lineNo := i + 2
		id := strings.TrimSpace(data.Cell(row, nodeIDIdx))
		if id == "" {
			id = fmt.Sprintf("node-%d", i+1)
			res.Warnings = append(res.Warnings, fmt.Sprintf("第 %d 行节点ID为空，已使用自动ID: %s", lineNo, id))
		}
		parent := strings.TrimSpace(data.Cell(row, parentIDIdx))
		if parent == id && parent != "" {
			res.Errors = append(res.Errors, fmt.Sprintf("第 %d 行节点ID与父节点相同（%s）", lineNo, id))
		}

		idRows[id] = append(idRows[id], lineNo)
		if parent != "" {
			parentRefs[parent] = append(parentRefs[parent], lineNo)
		}
		edges = append(edges, edgeRow{line: lineNo, nodeID: id, parent: parent})

		if parent != "" {
			if existed, ok := parentByChild[id]; ok && existed != parent {
				res.Errors = append(res.Errors, fmt.Sprintf("第 %d 行节点 %s 的父节点与前序记录冲突（%s vs %s）", lineNo, id, existed, parent))
			} else {
				parentByChild[id] = parent
			}
		}
	}

	if len(edges) == 0 {
		res.Errors = append(res.Errors, "未解析到树结构数据")
	}

	for id, lines := range idRows {
		if len(lines) > 1 {
			res.Warnings = append(res.Warnings, fmt.Sprintf("节点ID %s 在多行重复出现（行: %v），会被合并为同一节点", id, lines))
		}
	}

	orphanCount := 0
	for parentID, lines := range parentRefs {
		if _, ok := idRows[parentID]; ok {
			continue
		}
		orphanCount++
		res.Warnings = append(res.Warnings, fmt.Sprintf("父节点 %s 未作为节点出现（引用行: %v），将被视为隐式父节点", parentID, lines))
	}

	// Detect cycle path by following each node's parent chain.
	seenCycles := map[string]bool{}
	for child := range parentByChild {
		pathIndex := map[string]int{}
		path := make([]string, 0, 8)
		cur := child
		for cur != "" {
			if start, ok := pathIndex[cur]; ok {
				cycle := append(append([]string{}, path[start:]...), cur)
				key := strings.Join(cycle, "->")
				if !seenCycles[key] {
					seenCycles[key] = true
					res.Errors = append(res.Errors, fmt.Sprintf("检测到循环父子关系: %s", strings.Join(cycle, " -> ")))
				}
				break
			}
			pathIndex[cur] = len(path)
			path = append(path, cur)
			next, ok := parentByChild[cur]
			if !ok {
				break
			}
			cur = next
		}
	}

	roots := 0
	for id := range idRows {
		parent, ok := parentByChild[id]
		if !ok || parent == "" {
			roots++
		}
	}

	res.Stats["rows"] = len(edges)
	res.Stats["nodes"] = len(idRows)
	res.Stats["roots"] = roots
	res.Stats["orphans"] = orphanCount
	res.OK = len(res.Errors) == 0
	return res
}

// Build creates a payload with the builder selected by chart kind.
func Build(dataset model.Dataset, cfg Config) (map[string]any, error) {
	cfg = Normalize(cfg)
	b, ok := Get(cfg.ChartKind)
	if !ok {
		return nil, fmt.Errorf("不支持的图形类型: %s", cfg.ChartKind)
	}
	payload, err := b.Build(dataset, cfg)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func init() {
	register(Definition{Kind: "bar", Name: "柱状图", Family: "基础分析", Description: "分类对比最直接", Hint: "适合按类别比较单值或多值。"}, buildCartesian)
	register(Definition{Kind: "line", Name: "折线图", Family: "基础分析", Description: "趋势变化", Hint: "适合按时间或顺序观察波动。"}, buildCartesian)
	register(Definition{Kind: "area", Name: "面积图", Family: "基础分析", Description: "趋势加体量感", Hint: "适合展示累计规模与趋势。"}, buildCartesian)
	register(Definition{Kind: "stack_bar", Name: "堆叠柱状图", Family: "基础分析", Description: "总量与构成并看", Hint: "适合对比总量和子项组成。"}, buildCartesian)
	register(Definition{Kind: "stack_area", Name: "堆叠面积图", Family: "基础分析", Description: "时间维度的构成变化", Hint: "适合看多个序列的累计走势。"}, buildCartesian)
	register(Definition{Kind: "scatter", Name: "散点图", Family: "基础分析", Description: "看分布与相关性", Hint: "支持气泡大小列。"}, buildScatter)
	register(Definition{Kind: "pie", Name: "饼图", Family: "构成分析", Description: "构成占比", Hint: "适合少量分类的比例展示。"}, buildItems)
	register(Definition{Kind: "donut", Name: "环形图", Family: "构成分析", Description: "环形占比", Hint: "和饼图类似，但更适合中间留白做摘要。"}, buildItems)
	register(Definition{Kind: "funnel", Name: "漏斗图", Family: "构成分析", Description: "阶段转化", Hint: "适合展示流程漏损。"}, buildItems)
	register(Definition{Kind: "radar", Name: "雷达图", Family: "构成分析", Description: "多指标能力画像", Hint: "适合少量核心维度的平均或代表值。"}, buildRadar)
	register(Definition{Kind: "gauge", Name: "仪表盘", Family: "构成分析", Description: "单指标聚合", Hint: "适合单个 KPI。"}, buildGauge)
	register(Definition{Kind: "sankey", Name: "桑基图", Family: "关系流向", Description: "节点流向关系", Hint: "需要来源、目标和可选权重。"}, buildRelation)
	register(Definition{Kind: "graph", Name: "关系图", Family: "关系流向", Description: "网络关系", Hint: "适合节点关系网络。"}, buildRelation)
	register(Definition{Kind: "chord", Name: "和弦图", Family: "关系流向", Description: "环形关系强度", Hint: "ECharts 6 原生类型，适合互联关系。"}, buildRelation)
	register(Definition{Kind: "tree", Name: "树图", Family: "层级结构", Description: "层级展开", Hint: "需要节点、父节点。"}, buildHierarchy)
	register(Definition{Kind: "treemap", Name: "矩形树图", Family: "层级结构", Description: "层级占比", Hint: "适合有父子层级和数值权重的数据。"}, buildHierarchy)
}
