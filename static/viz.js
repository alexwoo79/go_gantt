(function () {
    const data = window.VIZ_DATA || {};
    const payload = data.payload || {};
    const selectedTheme = String(data.theme || "default").toLowerCase();
    const node = document.getElementById("vizChart");
    const vizPanel = document.getElementById("vizPanel");
    if (!node || !window.echarts) return;

    const chart = echarts.init(node, selectedTheme === "default" ? null : selectedTheme);
    const kind = String(payload.kind || "bar").toLowerCase();
    const dark = selectedTheme === "dark";
    const titleData = payload.title || { text: "", subtext: "" };
    const seriesName = String(payload.seriesName || "").trim();

    // Enhance subtitle with series name if available
    if (seriesName && !String(titleData.subtext || "").includes(seriesName)) {
        titleData.subtext = String(titleData.subtext || "").trim()
            ? (titleData.subtext + " · " + seriesName)
            : seriesName;
    }

    const hasTitleText = !!String(titleData.text || "").trim();
    const hasSubTitleText = !!String(titleData.subtext || "").trim();
    // Keep legend below toolbox to avoid overlap with action icons.
    const legendTop = 44;
    const gridTop = 102;

    const toolboxTheme = {
        border: dark ? "#94a3b8" : "#64748b",
        borderHover: dark ? "#e2e8f0" : "#1e293b",
        fillHover: dark ? "rgba(255,255,255,0.10)" : "rgba(15,23,42,0.06)",
        shadowHover: dark ? "rgba(255,255,255,0.18)" : "rgba(15,23,42,0.15)",
        text: dark ? "#e2e8f0" : "#1e2d41",
        textBg: dark ? "rgba(15,23,42,0.92)" : "rgba(255,255,255,0.94)",
        textBorder: dark ? "rgba(148,163,184,0.30)" : "#d6deea"
    };

    function isFullscreenActive() {
        return document.fullscreenElement === vizPanel
            || document.webkitFullscreenElement === vizPanel
            || document.mozFullScreenElement === vizPanel
            || document.msFullscreenElement === vizPanel;
    }

    function toggleVizFullscreen() {
        if (!vizPanel) return;
        if (isFullscreenActive()) {
            if (document.exitFullscreen) {
                document.exitFullscreen();
            } else if (document.webkitExitFullscreen) {
                document.webkitExitFullscreen();
            } else if (document.mozCancelFullScreen) {
                document.mozCancelFullScreen();
            } else if (document.msExitFullscreen) {
                document.msExitFullscreen();
            }
            return;
        }
        if (vizPanel.requestFullscreen) {
            vizPanel.requestFullscreen();
        } else if (vizPanel.webkitRequestFullscreen) {
            vizPanel.webkitRequestFullscreen();
        } else if (vizPanel.mozRequestFullScreen) {
            vizPanel.mozRequestFullScreen();
        } else if (vizPanel.msRequestFullscreen) {
            vizPanel.msRequestFullscreen();
        }
    }

    function exportSVG() {
        const host = document.getElementById("vizChart");
        const w = host.offsetWidth || 1200;
        const h = host.offsetHeight || 600;
        const tmpDiv = document.createElement("div");
        tmpDiv.style.cssText = "position:absolute;left:-9999px;top:0;width:" + w + "px;height:" + h + "px;";
        document.body.appendChild(tmpDiv);
        try {
            const svgChart = echarts.init(tmpDiv, selectedTheme === "default" ? null : selectedTheme, { renderer: "svg", width: w, height: h });
            svgChart.setOption(baseOption);

            let svgStr;
            if (typeof svgChart.renderToSVGString === "function") {
                svgStr = svgChart.renderToSVGString();
            } else {
                const svgEl = tmpDiv.querySelector("svg");
                if (!svgEl) {
                    alert("SVG 导出失败，当前 ECharts 版本不支持，请升级后重试。");
                    return;
                }
                svgEl.setAttribute("xmlns", "http://www.w3.org/2000/svg");
                svgStr = svgEl.outerHTML;
            }

            const blob = new Blob([svgStr], { type: "image/svg+xml;charset=utf-8" });
            const url = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = "viz.svg";
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
            svgChart.dispose();
        } catch (e) {
            alert("SVG 导出出错: " + e.message);
        } finally {
            if (document.body.contains(tmpDiv)) document.body.removeChild(tmpDiv);
        }
    }

    function exportHTML() {
        const bg = baseOption.backgroundColor || "#ffffff";
        const CS = "</" + "script>";
        Promise.all([
            fetch("/static/viz.js").then(function (r) { return r.text(); }),
            fetch("https://cdn.jsdelivr.net/npm/echarts@6/dist/echarts.min.js").then(function (r) { return r.text(); })
        ]).then(function (results) {
            const vizSrc = results[0].replace(/<\/script/gi, "<\\/script");
            const echartsSrc = results[1].replace(/<\/script/gi, "<\\/script");
            const vizData = JSON.stringify(window.VIZ_DATA || {}, null, 2);

            const html = [
                "<!DOCTYPE html>",
                "<html lang=\"zh-CN\">",
                "<head>",
                "  <meta charset=\"UTF-8\" />",
                "  <meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\" />",
                "  <title>通用图形</title>",
                "  <style>html,body{margin:0;padding:0;background:" + bg + ";}#vizChart{width:100vw;height:100vh;}<" + "/style>",
                "</head>",
                "<body>",
                "  <div id=\"vizChart\"></div>",
                "  <script>" + echartsSrc + CS,
                "  <script>window.VIZ_DATA=" + vizData + ";" + CS,
                "  <script>" + vizSrc + CS,
                "</body>",
                "</html>"
            ].join("\n");

            const blob = new Blob([html], { type: "text/html;charset=utf-8" });
            const url = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = "viz.html";
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        }).catch(function () {
            alert("HTML 导出失败，请重试。");
        });
    }

    const baseOption = {
        backgroundColor: "transparent",
        title: Object.assign({
            left: 10,
            top: 10,
            textStyle: { fontSize: 16, fontWeight: 700 },
            subtextStyle: { fontSize: 11, color: "#64748b" }
        }, titleData),
        animationDuration: 500,
        animationDurationUpdate: 700,
        toolbox: {
            show: true,
            right: 10,
            top: 8,
            itemSize: 16,
            itemGap: 12,
            iconStyle: {
                color: "none",
                borderColor: toolboxTheme.border,
                borderWidth: 1.5
            },
            emphasis: {
                iconStyle: {
                    color: toolboxTheme.fillHover,
                    borderColor: toolboxTheme.borderHover,
                    borderWidth: 2,
                    shadowBlur: 6,
                    shadowColor: toolboxTheme.shadowHover,
                    textFill: toolboxTheme.text,
                    textBackgroundColor: toolboxTheme.textBg,
                    textBorderRadius: 8,
                    textPadding: [5, 8],
                    textBorderColor: toolboxTheme.textBorder,
                    textBorderWidth: 1
                }
            },
            feature: (function () {
                var features = {
                    restore: { title: "还原" },
                    dataView: {
                        title: "数据视图",
                        readOnly: true,
                        lang: ["数据视图", "关闭", "刷新"]
                    },
                    saveAsImage: {
                        title: "下载 PNG",
                        name: "viz",
                        pixelRatio: 2,
                        backgroundColor: "#ffffff"
                    },
                    mySaveSVG: {
                        show: true,
                        title: "下载 SVG",
                        icon: "path://M14 2H6a2 2 0 0 0-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm-1 2l5 5h-5V4zM9 19v-4H7l4-4 4 4h-2v4H9z",
                        onclick: function () { exportSVG(); }
                    },
                    mySaveHTML: {
                        show: true,
                        title: "下载 HTML",
                        icon: "path://M9.4 16.6L4.8 12l4.6-4.6L8 6l-6 6 6 6 1.4-1.4zm5.2 0l4.6-4.6-4.6-4.6L16 6l6 6-6 6-1.4-1.4z",
                        onclick: function () { exportHTML(); }
                    },
                    myFullscreen: {
                        show: true,
                        title: "全屏显示",
                        icon: "path://M3 3h7v2H5v5H3V3zm18 0v7h-2V5h-5V3h7zM3 21v-7h2v5h5v2H3zm18-7v7h-7v-2h5v-5h2z",
                        onclick: function () { toggleVizFullscreen(); }
                    }
                };
                features.dataZoom = {
                    yAxisIndex: "none",
                    title: { zoom: "区域缩放", back: "缩放还原" },
                    brushStyle: {
                        color: dark ? "rgba(99,179,237,0.12)" : "rgba(59,130,246,0.10)",
                        borderColor: dark ? "#63b3ed" : "#3b82f6",
                        borderWidth: 1
                    }
                };
                features.myFitAll = {
                    show: true,
                    title: "适配全部",
                    icon: "path://M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5",
                    onclick: function () {
                        chart.dispatchAction({ type: "dataZoom", start: 0, end: 100 });
                    }
                };
                features.brush = {
                    type: ["lineX", "clear"],
                    title: { lineX: "横向刷选", clear: "清除刷选" }
                };
                return features;
            })()
        },
        grid: { left: 54, right: 30, top: gridTop, bottom: 56 },
        tooltip: { trigger: "axis" },
        legend: { top: legendTop, left: 10, right: 260, type: "scroll" },
        axisPointer: { link: [{ xAxisIndex: "all" }] },
        dataZoom: [],
        xAxis: undefined,
        yAxis: undefined,
        radar: undefined,
        series: []
    };

    if (kind === "pie" || kind === "donut") {
        baseOption.tooltip = { trigger: "item" };
        baseOption.legend = { top: legendTop, left: 10, right: 260, type: "scroll" };
        baseOption.series = [
            {
                name: payload.seriesName || "Series 1",
                type: "pie",
                radius: kind === "donut" ? ["38%", "68%"] : ["0%", "68%"],
                center: ["50%", "58%"],
                data: payload.items || [],
                label: { formatter: "{b}: {d}%" },
                emphasis: {
                    itemStyle: {
                        shadowBlur: 20,
                        shadowOffsetX: 0,
                        shadowColor: "rgba(0,0,0,0.2)"
                    }
                }
            }
        ];
    } else if (kind === "funnel") {
        baseOption.tooltip = { trigger: "item" };
        baseOption.series = [
            {
                name: payload.seriesName || "Series 1",
                type: "funnel",
                left: "10%",
                top: 70,
                bottom: 24,
                width: "80%",
                sort: "descending",
                gap: 3,
                label: { show: true, position: "inside" },
                data: payload.items || []
            }
        ];
    } else if (kind === "scatter") {
        baseOption.xAxis = { type: "value", name: payload.xName || "X" };
        baseOption.yAxis = { type: "value", name: payload.yName || "Y" };
        baseOption.tooltip = {
            trigger: "item",
            formatter: function (params) {
                var point = params && params.data ? params.data : {};
                var value = Array.isArray(point.value) ? point.value : [];
                var xVal = value[0] != null ? value[0] : "-";
                var yVal = value[1] != null ? value[1] : "-";
                var sizeVal = value[2] != null ? value[2] : "-";
                var lines = [
                    (params.seriesName || payload.seriesName || "Series 1"),
                    (payload.xName || "X") + ": " + xVal,
                    (payload.yName || "Y") + ": " + yVal
                ];
                if (payload.sizeName) lines.push(payload.sizeName + ": " + sizeVal);
                return lines.join("<br/>");
            }
        };
        baseOption.series = [
            {
                name: payload.seriesName || "Series 1",
                type: "scatter",
                data: payload.points || [],
                symbolSize: function (point) {
                    var raw = Array.isArray(point)
                        ? point[2]
                        : (point && point.value && point.value[2]);
                    return raw || 10;
                }
            }
        ];
    } else if (kind === "radar") {
        baseOption.tooltip = { trigger: "item" };
        baseOption.legend = { show: false };
        baseOption.radar = {
            radius: "58%",
            center: ["50%", "61%"],
            indicator: payload.indicators || []
        };
        baseOption.series = [
            {
                name: payload.seriesName || "Series 1",
                type: "radar",
                data: [
                    {
                        name: payload.seriesName || "Series 1",
                        value: payload.values || []
                    }
                ],
                areaStyle: { opacity: 0.16 }
            }
        ];
    } else if (kind === "gauge") {
        baseOption.tooltip = { trigger: "item", formatter: "{a}<br/>{b}: {c}" };
        baseOption.series = [
            {
                name: payload.seriesName || "Series 1",
                type: "gauge",
                min: 0,
                max: payload.max || 100,
                detail: { formatter: "{value}" },
                data: [
                    {
                        value: payload.value || 0,
                        name: payload.seriesName || "Series 1"
                    }
                ]
            }
        ];
    } else if (kind === "sankey") {
        baseOption.tooltip = { trigger: "item" };
        baseOption.series = [
            {
                name: payload.seriesName || "Flow",
                type: "sankey",
                emphasis: { focus: "adjacency" },
                nodeAlign: "justify",
                lineStyle: { color: "gradient", curveness: 0.45 },
                data: payload.nodes || [],
                links: payload.links || []
            }
        ];
    } else if (kind === "chord") {
        const chordPalette = [
            "#4F6FD4",
            "#B2CF2F",
            "#52567C",
            "#F39245",
            "#73C0DE",
            "#9A60B4",
            "#EA7CCC"
        ];
        const sourceNodes = Array.isArray(payload.nodes) ? payload.nodes : [];
        const chordNodes = sourceNodes.map(function (n, idx) {
            const item = Object.assign({}, n);
            if (!item.itemStyle) {
                item.itemStyle = { color: chordPalette[idx % chordPalette.length] };
            }
            return item;
        });
        const labelFontSize = chordNodes.length <= 6 ? 16 : (chordNodes.length <= 12 ? 13 : 11);

        baseOption.backgroundColor = "transparent";
        baseOption.title = Object.assign({}, payload.title || { text: "", subtext: "" }, {
            left: 10,
            top: 10,
            textStyle: { fontSize: 15, fontWeight: 700 },
            subtextStyle: { fontSize: 11, color: "#6b7280" }
        });
        baseOption.tooltip = { trigger: "item" };
        baseOption.legend = {
            top: legendTop,
            left: 10,
            right: 260,
            type: "scroll",
            itemWidth: 28,
            itemHeight: 18,
            icon: "roundRect",
            textStyle: { fontSize: 13 },
            data: chordNodes.map(function (n) { return n.name; })
        };
        baseOption.series = [
            {
                name: payload.seriesName || "Chord",
                type: "chord",
                clockwise: false,
                radius: ["67.5%", "73%"],
                center: ["50%", "62%"],
                startAngle: 90,
                padAngle: 3,
                sort: "descending",
                sortSub: "descending",
                itemStyle: {
                    borderWidth: 1.2,
                    borderColor: "rgba(31, 41, 55, 0.85)"
                },
                label: {
                    show: true,
                    distance: 6,
                    color: "inherit",
                    fontSize: labelFontSize,
                    fontWeight: 500
                },
                lineStyle: {
                    color: "target",
                    opacity: 0.42,
                    width: 1.1,
                    curveness: 0.32
                },
                emphasis: {
                    focus: "adjacency",
                    lineStyle: { opacity: 0.72, width: 2.2 }
                },
                data: chordNodes,
                links: payload.links || []
            }
        ];
    } else if (kind === "graph") {
        baseOption.tooltip = { trigger: "item" };
        baseOption.series = [
            {
                name: payload.seriesName || "Graph",
                type: "graph",
                layout: "force",
                roam: true,
                data: payload.nodes || [],
                links: payload.links || [],
                lineStyle: {
                    color: "source",
                    curveness: 0.12,
                    width: 1.2,
                    opacity: 0.75
                },
                emphasis: { focus: "adjacency", lineStyle: { width: 4 } },
                label: { show: true, position: "right" },
                force: { repulsion: 160, edgeLength: [36, 120], gravity: 0.08 }
            }
        ];
    } else if (kind === "tree") {
        baseOption.tooltip = { trigger: "item", triggerOn: "mousemove" };
        baseOption.series = [
            {
                name: payload.seriesName || "Tree",
                type: "tree",
                data: [payload.tree || { name: "root" }],
                top: "10%",
                left: "8%",
                bottom: "8%",
                right: "28%",
                symbolSize: 10,
                orient: "LR",
                expandAndCollapse: true,
                initialTreeDepth: 4,
                animationDurationUpdate: 500,
                label: {
                    position: "left",
                    verticalAlign: "middle",
                    align: "right",
                    fontSize: 12
                },
                leaves: {
                    label: {
                        position: "right",
                        verticalAlign: "middle",
                        align: "left"
                    }
                }
            }
        ];
    } else if (kind === "treemap") {
        baseOption.tooltip = { trigger: "item" };
        baseOption.series = [
            {
                name: payload.seriesName || "Treemap",
                type: "treemap",
                roam: false,
                nodeClick: "zoomToNode",
                breadcrumb: { show: true },
                label: { show: true, formatter: "{b}" },
                data: (payload.tree && payload.tree.children) ? payload.tree.children : []
            }
        ];
    } else {
        const seriesDefs = Array.isArray(payload.series) ? payload.series : [];
        const type = kind === "bar" || kind === "stack_bar" ? "bar" : "line";
        baseOption.xAxis = {
            type: "category",
            data: payload.xAxis || [],
            axisLabel: { interval: 0, rotate: 0 }
        };
        baseOption.yAxis = { type: "value" };
        baseOption.tooltip = { trigger: "axis" };
        if ((payload.xAxis || []).length > 8) {
            baseOption.dataZoom = [
                { type: "inside", xAxisIndex: 0 },
                { type: "slider", xAxisIndex: 0, height: 18, bottom: 14 }
            ];
        }
        baseOption.series = seriesDefs.map(function (s) {
            var rawData = Array.isArray(s.data) ? s.data : [];
            const item = {
                name: s.name || "Series",
                type: type,
                data: rawData,
                smooth: !!s.smooth,
                showSymbol: type === "line",
                showBackground: type === "bar",
                barMaxWidth: 42
            };
            if (kind === "area" || kind === "stack_area") {
                item.areaStyle = { opacity: 0.18 };
            }
            if (kind === "stack_bar" || kind === "stack_area") {
                item.stack = "total";
            }
            return item;
        });
    }

    chart.setOption(baseOption);
    ["fullscreenchange", "webkitfullscreenchange", "mozfullscreenchange", "MSFullscreenChange"].forEach(function (evt) {
        document.addEventListener(evt, function () {
            setTimeout(function () { chart.resize(); }, 120);
        });
    });
    window.addEventListener("resize", function () {
        chart.resize();
    });
})();
