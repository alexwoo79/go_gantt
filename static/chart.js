(function () {
  const payload = window.GANTT_DATA || { tasks: [], stats: {}, options: {} };
  const tasks = payload.tasks || [];
  const stats = payload.stats || {};
  const options = payload.options || {};
  const selectedTheme = String(options.chartTheme || "default").toLowerCase();

  const setText = (id, v) => {
    const node = document.getElementById(id);
    if (node) node.textContent = v;
  };

  setText("mTaskCount", stats.taskCount || 0);
  setText("mAvg", Number(stats.avgDurationDays || 0).toFixed(1));
  setText("mTotal", stats.totalDurationDay || 0);
  setText("mMax", stats.maxDurationDay || 0);
  setText("mPlanTotal", stats.planTotalDurationDay || 0);

  const planTotalWrap = document.getElementById("mPlanTotalWrap");
  if (planTotalWrap) {
    planTotalWrap.style.display = stats.hasPlanTotalDuration ? "block" : "none";
  }

  const paletteByTheme = {
    default: ["#006d77", "#ff8f00", "#2e7d32", "#ad1457", "#5d4037", "#1976d2", "#6a1b9a"],
    macarons: ["#2ec7c9", "#b6a2de", "#5ab1ef", "#ffb980", "#d87a80", "#8d98b3", "#e5cf0d"],
    infographic: ["#c1232b", "#27727b", "#fcce10", "#e87c25", "#b5c334", "#fe8463", "#9bca63"],
    shine: ["#c12e34", "#e6b600", "#0098d9", "#2b821d", "#005eaa", "#339ca8", "#cda819"],
    roma: ["#e01f54", "#001852", "#f5e8c8", "#b8d2c7", "#c6b38e", "#a4d8c2", "#f3d999"],
    vintage: ["#d87c7c", "#919e8b", "#d7ab82", "#6e7074", "#61a0a8", "#efa18d", "#787464"],
    dark: ["#4dd0e1", "#ffb74d", "#81c784", "#f06292", "#a1887f", "#64b5f6", "#ba68c8"]
  };
  const palette = paletteByTheme[selectedTheme] || paletteByTheme.default;
  const ganttPanel = document.getElementById("ganttPanel");
  const fullscreenBtn = document.getElementById("ganttFullscreenBtn");
  const groups = [...new Set(tasks.map((t) => t.colorGroup || t.project || "未分组"))];
  const colorMap = {};
  groups.forEach((g, i) => {
    colorMap[g] = palette[i % palette.length];
  });

  function isoWeekNumber(ts) {
    const d = new Date(ts);
    d.setHours(0, 0, 0, 0);
    d.setDate(d.getDate() + 3 - ((d.getDay() + 6) % 7));
    const week1 = new Date(d.getFullYear(), 0, 4);
    return 1 + Math.round(((d - week1) / DAY_MS - 3 + ((week1.getDay() + 6) % 7)) / 7);
  }

  function axisLabelFormatter(ts) {
    const d = new Date(ts);
    if (granularity === "day") {
      return echarts.format.formatTime("MM-dd", ts);
    }
    if (granularity === "week") {
      const week = String(isoWeekNumber(ts)).padStart(2, "0");
      return "W" + week + " " + echarts.format.formatTime("MM-dd", ts);
    }
    if (granularity === "month") {
      return echarts.format.formatTime("yyyy-MM", ts);
    }
    if (granularity === "quarter") {
      const quarter = Math.floor(d.getMonth() / 3) + 1;
      return d.getFullYear() + " Q" + quarter;
    }
    return echarts.format.formatTime("yyyy", ts);
  }

  const DAY_MS = 24 * 3600 * 1000;
  const granularity = options.timeGranularity || "month";
  const chartTheme = selectedTheme;
  const echartsTheme = chartTheme === "default" ? null : chartTheme;
  const dark = !!options.darkTheme || chartTheme === "dark";
  const surfaceTheme = {
    panelBg: dark ? "rgba(15,23,42,0.96)" : "rgba(255,255,255,0.98)",
    panelBorder: dark ? "rgba(148,163,184,0.30)" : "#d6deea",
    panelText: dark ? "#e2e8f0" : "#1e2d41",
    panelMuted: dark ? "#94a3b8" : "#64748b",
    panelShadow: dark ? "0 14px 28px rgba(2,8,23,0.34)" : "0 12px 24px rgba(15,23,42,0.16)",
    toolHintBg: dark ? "rgba(15,23,42,0.92)" : "rgba(255,255,255,0.94)",
    toolHintText: dark ? "#e2e8f0" : "#1e2d41"
  };

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/\"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function floorToGranularity(ts, g) {
    const d = new Date(ts);
    if (g === "day") {
      d.setHours(0, 0, 0, 0);
      return +d;
    }
    if (g === "week") {
      d.setHours(0, 0, 0, 0);
      const day = d.getDay();
      const delta = day === 0 ? -6 : 1 - day;
      d.setDate(d.getDate() + delta);
      return +d;
    }
    if (g === "month") {
      return +new Date(d.getFullYear(), d.getMonth(), 1);
    }
    if (g === "quarter") {
      const qMonth = Math.floor(d.getMonth() / 3) * 3;
      return +new Date(d.getFullYear(), qMonth, 1);
    }
    return +new Date(d.getFullYear(), 0, 1);
  }

  function ceilToGranularity(ts, g) {
    const d = new Date(ts);
    if (g === "day") {
      return +new Date(d.getFullYear(), d.getMonth(), d.getDate() + 1);
    }
    if (g === "week") {
      const f = floorToGranularity(ts, g);
      return f + 7 * DAY_MS;
    }
    if (g === "month") {
      return +new Date(d.getFullYear(), d.getMonth() + 1, 1);
    }
    if (g === "quarter") {
      const qMonth = Math.floor(d.getMonth() / 3) * 3;
      return +new Date(d.getFullYear(), qMonth + 3, 1);
    }
    return +new Date(d.getFullYear() + 1, 0, 1);
  }
  const timeGridByGranularity = {
    day: { splitNumber: 18, minInterval: DAY_MS, interval: DAY_MS, minorSplitNumber: 3, showMinor: true },
    week: { splitNumber: 14, minInterval: 7 * DAY_MS, interval: 7 * DAY_MS, minorSplitNumber: 1, showMinor: true },
    month: { splitNumber: 12, minInterval: 28 * DAY_MS, interval: null, minorSplitNumber: 0, showMinor: false },
    quarter: { splitNumber: 8, minInterval: 84 * DAY_MS, interval: null, minorSplitNumber: 0, showMinor: false },
    year: { splitNumber: 6, minInterval: 365 * DAY_MS, interval: null, minorSplitNumber: 0, showMinor: false }
  };
  const timeGridCfg = timeGridByGranularity[granularity] || timeGridByGranularity.month;

  const gantt = echarts.init(document.getElementById("gantt"), echartsTheme);
  const canvasBgByTheme = {
    default: dark ? "#10213b" : "#ffffff",
    macarons: dark ? "#1e2733" : "#fff7fb",
    infographic: dark ? "#18222b" : "#fffdf5",
    shine: dark ? "#122533" : "#f9fcff",
    roma: dark ? "#1d2026" : "#fffaf0",
    vintage: dark ? "#25211f" : "#f7f1e6",
    dark: "#10213b"
  };
  const milestoneColor = palette[Math.max(0, palette.length - 1)] || "#c62828";
  const nowTs = Date.now();

  function isFullscreenActive() {
    return document.fullscreenElement === ganttPanel
      || document.webkitFullscreenElement === ganttPanel
      || document.mozFullScreenElement === ganttPanel
      || document.msFullscreenElement === ganttPanel;
  }

  function updateFullscreenButtonText() {
    if (!fullscreenBtn) return;
    fullscreenBtn.textContent = isFullscreenActive() ? "退出全屏" : "全屏";
  }

  function toggleGanttFullscreen() {
    if (!ganttPanel) return;
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

    if (ganttPanel.requestFullscreen) {
      ganttPanel.requestFullscreen();
    } else if (ganttPanel.webkitRequestFullscreen) {
      ganttPanel.webkitRequestFullscreen();
    } else if (ganttPanel.mozRequestFullScreen) {
      ganttPanel.mozRequestFullScreen();
    } else if (ganttPanel.msRequestFullscreen) {
      ganttPanel.msRequestFullscreen();
    }
  }

  function barHeightForRow(rowType) {
    return rowType === "project" ? 36 : 24;
  }

  function visualBarHeight(laneHeight, preferredHeight) {
    const fallback = laneHeight * 0.7;
    return Math.min(preferredHeight || fallback, laneHeight * 0.92);
  }

  function barRadiusForRow(rowType) {
    return rowType === "project" ? 6 : 4;
  }

  function readableTextColor(bgHex, isDarkTheme) {
    if (typeof bgHex !== "string") {
      return isDarkTheme ? "#f8fafc" : "#0f172a";
    }
    const hex = bgHex.replace("#", "").trim();
    if (!(hex.length === 3 || hex.length === 6)) {
      return isDarkTheme ? "#f8fafc" : "#0f172a";
    }
    const full = hex.length === 3
      ? hex.split("").map((c) => c + c).join("")
      : hex;
    const r = parseInt(full.slice(0, 2), 16);
    const g = parseInt(full.slice(2, 4), 16);
    const b = parseInt(full.slice(4, 6), 16);
    if (Number.isNaN(r) || Number.isNaN(g) || Number.isNaN(b)) {
      return isDarkTheme ? "#f8fafc" : "#0f172a";
    }
    const luminance = 0.2126 * r + 0.7152 * g + 0.0722 * b;
    if (isDarkTheme) {
      return luminance < 145 ? "#f8fafc" : "#0f172a";
    }
    return luminance < 150 ? "#f8fafc" : "#0f172a";
  }

  function planStrokeTheme(isDarkTheme) {
    if (isDarkTheme) {
      return {
        outer: "rgba(248,250,252,0.25)",
        inner: "#dbe7f3"
      };
    }
    return {
      outer: "rgba(51,65,85,0.20)",
      inner: "#5b6f8a"
    };
  }

  function progressForRange(startTs, endTs) {
    const total = Math.max(1, endTs - startTs);
    const passed = Math.min(Math.max(nowTs - startTs, 0), total);
    return passed / total;
  }

  function buildHierRows() {
    const grouped = {};
    tasks.forEach((t) => {
      const p = t.project || "未分组";
      if (!grouped[p]) {
        grouped[p] = {
          items: [],
          minStart: Number.POSITIVE_INFINITY,
          maxEnd: Number.NEGATIVE_INFINITY
        };
      }
      grouped[p].items.push(t);
      const s = +new Date(t.startISO);
      const e = +new Date(t.endISO);
      grouped[p].minStart = Math.min(grouped[p].minStart, s);
      grouped[p].maxEnd = Math.max(grouped[p].maxEnd, e);
    });

    const projectOrder = Object.keys(grouped).sort((a, b) => {
      if (grouped[a].minStart !== grouped[b].minStart) {
        return grouped[a].minStart - grouped[b].minStart;
      }
      return a.localeCompare(b, "zh-CN");
    });

    const rows = [];
    projectOrder.forEach((project) => {
      const list = grouped[project].items.slice().sort((a, b) => {
        const sa = +new Date(a.startISO);
        const sb = +new Date(b.startISO);
        if (sa !== sb) {
          return sa - sb;
        }
        return (a.taskName || "").localeCompare(b.taskName || "", "zh-CN");
      });
      const minStart = grouped[project].minStart;
      const maxEnd = grouped[project].maxEnd;
      const owner = (list.find((x) => x.owner) || {}).owner || "";
      rows.push({
        rowLabel: project,
        rowType: "project",
        start: minStart,
        end: maxEnd,
        project,
        colorGroup: list[0].colorGroup || project,
        task: project,
        duration: Math.round((maxEnd - minStart) / 86400000) + 1,
        description: owner,
        progress: progressForRange(minStart, maxEnd)
      });

      list.forEach((t) => {
        rows.push({
          rowLabel: "  " + t.taskName,
          rowType: "task",
          start: +new Date(t.startISO),
          end: +new Date(t.endISO),
          project: t.project,
          colorGroup: t.colorGroup || t.project,
          task: t.taskName,
          duration: t.durationDays,
          description: t.description || "",
          progress: progressForRange(+new Date(t.startISO), +new Date(t.endISO)),
          planStart: t.planStartISO ? +new Date(t.planStartISO) : null,
          planEnd: t.planEndISO ? +new Date(t.planEndISO) : null,
          milestoneName: t.milestoneName || "",
          milestone: t.milestoneISO ? +new Date(t.milestoneISO) : null
        });
      });
    });

    return rows;
  }

  function barRender(params, api) {
    const y = api.value(0);
    const s = api.coord([api.value(1), y]);
    const e = api.coord([api.value(2), y]);
    const laneHeight = api.size([0, 1])[1];
    const preferred = api.value(6);
    const h = visualBarHeight(laneHeight, preferred);
    const rect = echarts.graphic.clipRectByRect({
      x: s[0],
      y: s[1] - h / 2,
      width: e[0] - s[0],
      height: h
    }, {
      x: params.coordSys.x,
      y: params.coordSys.y,
      width: params.coordSys.width,
      height: params.coordSys.height
    });
    if (!rect) {
      return null;
    }

    const progress = Math.max(0, Math.min(1, Number(api.value(9)) || 0));
    const rowType = api.value(10);
    const radius = barRadiusForRow(rowType);
    const detail = api.value(8) || api.value(5) || "";
    const progressWidth = Math.max(2, rect.width * progress);
    const barFill = api.value(3);
    const textColor = readableTextColor(barFill, dark);

    const children = [
      {
        type: "rect",
        shape: {
          x: rect.x,
          y: rect.y,
          width: rect.width,
          height: rect.height,
          r: radius
        },
        style: api.style({
          fill: barFill,
          opacity: rowType === "project" ? 0.95 : 0.86
        })
      },
      {
        type: "rect",
        shape: {
          x: rect.x,
          y: rect.y,
          width: progressWidth,
          height: rect.height,
          r: radius
        },
        style: {
          fill: dark ? "rgba(255,255,255,0.16)" : "rgba(255,255,255,0.32)"
        }
      }
    ];

    // Keep task detail text inside task bars to match PM-style gantt readability.
    if (rowType === "task" && options.showTaskDetails && detail && rect.width > 36) {
      children.push({
        type: "text",
        style: {
          x: rect.x + 6,
          y: rect.y + rect.height / 2,
          text: detail,
          width: Math.max(24, rect.width - 12),
          overflow: "truncate",
          fill: textColor,
          fontSize: 12,
          fontWeight: 600,
          textVerticalAlign: "middle",
          textAlign: "left"
        },
        silent: true
      });
    }

    return {
      type: "group",
      children
    };
  }

  const rows = buildHierRows();

  const minTime = Math.min.apply(null, rows.map((r) => r.start));
  const maxTime = Math.max.apply(null, rows.map((r) => r.end));
  const axisMin = floorToGranularity(minTime, granularity);
  const axisMax = ceilToGranularity(maxTime, granularity);
  const axisSpanDays = Math.max(1, Math.round((axisMax - axisMin) / DAY_MS));

  if (granularity === "day") {
    const intervalDays = axisSpanDays > 90 ? 10 : axisSpanDays > 45 ? 7 : axisSpanDays > 20 ? 2 : 1;
    timeGridCfg.interval = intervalDays * DAY_MS;
    timeGridCfg.splitNumber = null;
    timeGridCfg.showMinor = false;
  }

  if (granularity === "week") {
    const intervalWeeks = axisSpanDays > 200 ? 4 : axisSpanDays > 100 ? 2 : 1;
    timeGridCfg.interval = intervalWeeks * 7 * DAY_MS;
    timeGridCfg.splitNumber = null;
    timeGridCfg.showMinor = false;
  }

  const yAxisData = rows.map((r) => r.rowLabel);
  const barData = rows.map((r, idx) => ({
    value: [idx, r.start, r.end, colorMap[r.colorGroup] || "#1976d2", r.project, r.task, barHeightForRow(r.rowType), r.duration, r.description, r.progress || 0, r.rowType],
    rowType: r.rowType
  }));

  const planData = rows
    .map((r, idx) => ({ idx, start: r.planStart, end: r.planEnd, task: r.task, rowType: r.rowType }))
    .filter((r) => r.start && r.end)
    .map((r) => ({ value: [r.idx, r.start, r.end, barHeightForRow(r.rowType), r.rowType], task: r.task }));

  const milestoneData = rows
    .map((r, idx) => ({ idx, date: r.milestone, name: r.milestoneName, task: r.task }))
    .filter((m) => m.date && m.name)
    .map((m) => ({ name: m.name, value: [m.date, m.idx], task: m.task }));

  const chartOption = {
    animationDuration: 700,
    color: palette,
    backgroundColor: canvasBgByTheme[chartTheme] || (dark ? "#10213b" : "#ffffff"),
    grid: { left: 10, right: 20, top: 44, bottom: 44, containLabel: true },
    toolbox: {
      show: true,
      right: 14,
      top: 6,
      itemSize: 16,
      itemGap: 12,
      iconStyle: {
        color: "none",
        borderColor: dark ? "#94a3b8" : "#64748b",
        borderWidth: 1.5
      },
      emphasis: {
        iconStyle: {
          color: dark ? "rgba(255,255,255,0.10)" : "rgba(15,23,42,0.06)",
          borderColor: dark ? "#e2e8f0" : "#1e293b",
          borderWidth: 2,
          shadowBlur: 6,
          shadowColor: dark ? "rgba(255,255,255,0.18)" : "rgba(15,23,42,0.15)",
          textFill: surfaceTheme.toolHintText,
          textBackgroundColor: surfaceTheme.toolHintBg,
          textBorderRadius: 8,
          textPadding: [5, 8],
          textBorderColor: surfaceTheme.panelBorder,
          textBorderWidth: 1
        }
      },
      feature: {
        dataZoom: {
          yAxisIndex: "none",
          title: { zoom: "区域缩放", back: "缩放还原" },
          brushStyle: {
            color: dark ? "rgba(99,179,237,0.12)" : "rgba(59,130,246,0.10)",
            borderColor: dark ? "#63b3ed" : "#3b82f6",
            borderWidth: 1
          }
        },
        brush: {
          type: ["lineX", "clear"],
          title: { lineX: "横向刷选", clear: "清除刷选" }
        },
        restore: { title: "还原" },
        dataView: {
          title: "数据视图",
          lang: ["数据视图", "关闭", "刷新"],
          readOnly: true,
          backgroundColor: dark ? "#0f172a" : "#f8fbff",
          textareaColor: dark ? "#0f172a" : "#ffffff",
          textareaBorderColor: dark ? "rgba(148,163,184,0.24)" : "#d6deea",
          textColor: dark ? "#e2e8f0" : "#1e2d41",
          buttonColor: "#2563eb",
          buttonTextColor: "#ffffff",
          optionToContent: function () {
            var taskList = (window.GANTT_DATA && window.GANTT_DATA.tasks) || [];
            var cols = ["#", "任务名", "项目", "开始日期", "结束日期", "周期(天)", "进度", "负责人", "说明"];
            var html = '<div class="data-view-surface' + (dark ? ' data-view-dark' : '') + '">';
            html += '<div class="data-view-head">';
            html += '<h3 class="data-view-title">数据预览</h3>';
            html += '<div class="data-view-meta">共 ' + taskList.length + ' 条记录</div>';
            html += '</div>';
            html += '<div class="data-view-wrap">';
            html += '<table class="preview-table">';
            html += '<thead><tr>';
            cols.forEach(function (h) { html += '<th>' + escapeHtml(h) + '</th>'; });
            html += '</tr></thead><tbody>';
            taskList.forEach(function (t, i) {
              var pct = t.progress != null ? Math.round(t.progress * 100) + "%" : "—";
              var sd = t.startISO ? new Date(t.startISO).toLocaleDateString("zh-CN") : "—";
              var ed = t.endISO ? new Date(t.endISO).toLocaleDateString("zh-CN") : "—";
              var desc = escapeHtml(t.description || "—");
              html += '<tr>';
              html += '<td style="text-align:center;color:' + (dark ? '#94a3b8' : '#64748b') + ';">' + (i + 1) + '</td>';
              html += '<td style="font-weight:600;">' + escapeHtml(t.taskName || "") + '</td>';
              html += '<td style="color:' + (dark ? '#94a3b8' : '#64748b') + ';">' + escapeHtml(t.project || "—") + '</td>';
              html += '<td>' + escapeHtml(sd) + '</td>';
              html += '<td>' + escapeHtml(ed) + '</td>';
              html += '<td style="text-align:center;">' + escapeHtml(t.durationDays || "—") + '</td>';
              html += '<td style="text-align:center;">' + escapeHtml(pct) + '</td>';
              html += '<td style="color:' + (dark ? '#94a3b8' : '#64748b') + ';">' + escapeHtml(t.owner || "—") + '</td>';
              html += '<td title="' + desc + '">' + desc + '</td>';
              html += '</tr>';
            });
            html += '</tbody></table></div></div>';
            return html;
          }
        },
        saveAsImage: {
          title: "下载 PNG",
          name: "gantt",
          pixelRatio: 2,
          backgroundColor: canvasBgByTheme[chartTheme] || (dark ? "#10213b" : "#ffffff")
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
        myFitAll: {
          show: true,
          title: "适配全部",
          icon: "path://M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5",
          onclick: function () {
            gantt.dispatchAction({ type: "dataZoom", start: 0, end: 100 });
          }
        },
        myFullscreen: {
          show: true,
          title: "全屏显示",
          icon: "path://M3 3h7v2H5v5H3V3zm18 0v7h-2V5h-5V3h7zM3 21v-7h2v5h5v2H3zm18-7v7h-7v-2h5v-5h2z",
          onclick: function () { toggleGanttFullscreen(); }
        }
      }
    },
    xAxis: {
      type: "time",
      min: axisMin,
      max: axisMax,
      minInterval: timeGridCfg.minInterval,
      splitNumber: timeGridCfg.interval ? undefined : timeGridCfg.splitNumber,
      interval: timeGridCfg.interval || undefined,
      axisLabel: {
        formatter: axisLabelFormatter,
        color: dark ? "#d8e0ea" : "#334155",
        showMinLabel: true,
        showMaxLabel: true,
        alignMinLabel: "left",
        alignMaxLabel: "right",
        hideOverlap: true,
        rotate: granularity === "day" ? 28 : (granularity === "week" ? 20 : 0),
        margin: 14
      },
      axisTick: { show: true },
      axisLine: { show: true, lineStyle: { color: dark ? "rgba(255,255,255,0.22)" : "#c8d4e2" } },
      splitLine: {
        show: true,
        lineStyle: { color: dark ? "rgba(255,255,255,0.12)" : "#dbe6f3", width: 1 }
      },
      minorTick: {
        show: timeGridCfg.showMinor,
        splitNumber: timeGridCfg.minorSplitNumber
      },
      minorSplitLine: {
        show: timeGridCfg.showMinor,
        lineStyle: { color: dark ? "rgba(255,255,255,0.05)" : "#eef3f9" }
      }
    },
    yAxis: {
      type: "category",
      inverse: true,
      data: yAxisData,
      splitArea: {
        show: true,
        areaStyle: {
          color: dark ? ["rgba(255,255,255,0.02)", "rgba(255,255,255,0.045)"] : ["#fbfdff", "#f5f9fe"]
        }
      },
      axisLabel: {
        color: dark ? "#d8e0ea" : "#334155",
        width: 150,
        overflow: "truncate"
      },
      axisLine: { show: true, lineStyle: { color: dark ? "rgba(255,255,255,0.18)" : "#c8d4e2" } }
    },
    tooltip: {
      show: true,
      trigger: "axis",
      axisPointer: { type: "line", snap: false, label: { show: true } },
      backgroundColor: surfaceTheme.panelBg,
      borderColor: surfaceTheme.panelBorder,
      borderWidth: 1,
      padding: [12, 14],
      extraCssText: "box-shadow: " + surfaceTheme.panelShadow + "; border-radius: 12px;",
      textStyle: { color: surfaceTheme.panelText },
      formatter: function (params) {
        const p = Array.isArray(params) ? params.find((x) => x.seriesName !== "任务详情") : params;
        if (!p) {
          return "";
        }
        if (p.seriesName === "里程碑") {
          return p.name + "<br/>任务: " + p.data.task + "<br/>日期: " + new Date(p.value[0]).toLocaleDateString();
        }
        if (p.seriesName === "计划基准线") {
          return "任务: " + p.data.task + "<br/>计划开始: " + new Date(p.value[1]).toLocaleDateString() + "<br/>计划结束: " + new Date(p.value[2]).toLocaleDateString();
        }
        const v = p.value;
        const durationPart = options.showDuration ? ("<br/>周期(天): " + v[7]) : "";
        const detailPart = options.showTaskDetails ? ("<br/>" + (v[8] || "")) : "";
        return v[5] + "<br/>项目: " + v[4] + "<br/>开始: " + new Date(v[1]).toLocaleDateString() + "<br/>结束: " + new Date(v[2]).toLocaleDateString() + durationPart + detailPart;
      }
    },
    series: [
      {
        name: "任务",
        type: "custom",
        renderItem: barRender,
        encode: { x: [1, 2], y: 0 },
        data: barData,
        itemStyle: { opacity: 0.9 }
      },
      {
        name: "计划基准线",
        type: "custom",
        renderItem: function (params, api) {
          const y = api.value(0);
          const s = api.coord([api.value(1), y]);
          const e = api.coord([api.value(2), y]);
          const laneHeight = api.size([0, 1])[1];
          const rowType = api.value(4);
          const h = visualBarHeight(laneHeight, api.value(3));
          const radius = barRadiusForRow(rowType);
          const rect = echarts.graphic.clipRectByRect({
            x: s[0],
            y: s[1] - h / 2,
            width: e[0] - s[0],
            height: h
          }, {
            x: params.coordSys.x,
            y: params.coordSys.y,
            width: params.coordSys.width,
            height: params.coordSys.height
          });
          if (!rect) {
            return null;
          }
          const strokeTheme = planStrokeTheme(dark);
          const outerLineWidth = 3;
          const innerLineWidth = 1.5;
          const dashPattern = rowType === "project" ? [7, 4] : [4, 3];
          const outerOpacity = rowType === "project" ? 0.62 : 1;
          const innerOpacity = rowType === "project" ? 0.74 : 1;
          const inset = outerLineWidth / 2;
          const insetRect = {
            x: rect.x + inset,
            y: rect.y + inset,
            width: Math.max(1, rect.width - outerLineWidth),
            height: Math.max(1, rect.height - outerLineWidth)
          };
          return {
            type: "group",
            children: [
              {
                type: "rect",
                shape: {
                  x: insetRect.x,
                  y: insetRect.y,
                  width: insetRect.width,
                  height: insetRect.height,
                  r: radius
                },
                style: {
                  fill: "transparent",
                  stroke: strokeTheme.outer,
                  lineWidth: outerLineWidth,
                  opacity: outerOpacity
                },
                silent: true
              },
              {
                type: "rect",
                shape: {
                  x: insetRect.x,
                  y: insetRect.y,
                  width: insetRect.width,
                  height: insetRect.height,
                  r: radius
                },
                style: {
                  fill: "transparent",
                  stroke: strokeTheme.inner,
                  lineDash: dashPattern,
                  lineWidth: innerLineWidth,
                  opacity: innerOpacity
                },
                silent: true
              }
            ]
          };
        },
        encode: { x: [1, 2], y: 0 },
        data: planData,
        z: 8
      },
      {
        name: "里程碑",
        type: "scatter",
        data: milestoneData,
        symbol: "diamond",
        symbolSize: 11,
        itemStyle: { color: milestoneColor },
        z: 10
      },
      {
        name: "今天",
        type: "line",
        markLine: {
          symbol: ["none", "none"],
          label: {
            show: true,
            formatter: "Today",
            color: dark ? "#fecaca" : "#b91c1c",
            backgroundColor: dark ? "rgba(127,29,29,0.18)" : "rgba(254,202,202,0.7)",
            padding: [2, 6],
            borderRadius: 8
          },
          lineStyle: {
            color: dark ? "#ef4444" : "#dc2626",
            type: "dashed",
            width: 1.2
          },
          data: [{ xAxis: nowTs }]
        },
        data: []
      }
    ]
  };

  gantt.setOption(chartOption);

  if (fullscreenBtn) {
    fullscreenBtn.addEventListener("click", toggleGanttFullscreen);
  }
  ["fullscreenchange", "webkitfullscreenchange", "mozfullscreenchange", "MSFullscreenChange"].forEach(function (evt) {
    document.addEventListener(evt, function () {
      updateFullscreenButtonText();
      setTimeout(function () { gantt.resize(); }, 120);
    });
  });
  updateFullscreenButtonText();

  /* ── 导出 SVG ────────────────────────────────────────── */
  function exportSVG() {
    var host = document.getElementById("gantt");
    var w = host.offsetWidth || 1200;
    var h = host.offsetHeight || 600;

    var tmpDiv = document.createElement("div");
    // Must be in DOM and have explicit size for ECharts SVG renderer
    tmpDiv.style.cssText = "position:absolute;left:-9999px;top:0;width:" + w + "px;height:" + h + "px;";
    document.body.appendChild(tmpDiv);

    try {
      var svgChart = echarts.init(tmpDiv, echartsTheme, { renderer: "svg", width: w, height: h });
      // Pass original option (NOT JSON clone) — cloning strips all functions (renderItem, formatter…)
      svgChart.setOption(chartOption);

      var svgStr;
      if (typeof svgChart.renderToSVGString === "function") {
        // ECharts 5.3+ preferred API
        svgStr = svgChart.renderToSVGString();
      } else {
        var svgEl = tmpDiv.querySelector("svg");
        if (!svgEl) {
          alert("SVG 导出失败，当前 ECharts 版本不支持，请升级到 5.3+。");
          return;
        }
        svgEl.setAttribute("xmlns", "http://www.w3.org/2000/svg");
        svgStr = svgEl.outerHTML;
      }

      var blob = new Blob([svgStr], { type: "image/svg+xml;charset=utf-8" });
      var url = URL.createObjectURL(blob);
      var a = document.createElement("a");
      a.href = url;
      a.download = "gantt.svg";
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

  /* ── 导出独立 HTML ───────────────────────────────────── */
  function exportHTML() {
    var bg = chartOption.backgroundColor || "#ffffff";
    // Use expression split so "</script" never appears literally in this source file,
    // preventing the HTML parser from closing the <script> block prematurely when
    // this file is itself embedded inline in the exported HTML.
    var CS = "</" + "script>";

    Promise.all([
      fetch("/static/chart.js").then(function (r) { return r.text(); }),
      fetch("https://cdn.jsdelivr.net/npm/echarts@5/dist/echarts.min.js").then(function (r) { return r.text(); })
    ]).then(function (results) {
      // Sanitize: replace </script → <\/script so embedded sources cannot close
      // the outer <script> block in the generated HTML file.
      var chartSrc = results[0].replace(/<\/script/gi, "<\\/script");
      var echartsSrc = results[1].replace(/<\/script/gi, "<\\/script");
      var data = JSON.stringify(window.GANTT_DATA, null, 2);

      var html = [
        "<!DOCTYPE html>",
        "<html lang=\"zh-CN\">",
        "<head>",
        "  <meta charset=\"UTF-8\" />",
        "  <meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\" />",
        "  <title>\u7518\u7279\u56fe</title>",
        "  <style>html,body{margin:0;padding:0;background:" + bg + ";}#gantt{width:100vw;height:100vh;}<" + "/style>",
        "</head>",
        "<body>",
        "  <div id=\"gantt\"></div>",
        "  <script>" + echartsSrc + CS,
        "  <script>window.GANTT_DATA=" + data + ";" + CS,
        "  <script>" + chartSrc + CS,
        "</body>",
        "</html>"
      ].join("\n");

      var blob = new Blob([html], { type: "text/html;charset=utf-8" });
      var url = URL.createObjectURL(blob);
      var a = document.createElement("a");
      a.href = url;
      a.download = "gantt.html";
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }).catch(function () { alert("HTML \u5bfc\u51fa\u5931\u8d25\uff0c\u8bf7\u91cd\u8bd5\u3002"); });
  }

  if (typeof ResizeObserver !== "undefined") {
    const host = document.getElementById("gantt");
    const ro = new ResizeObserver(function () {
      gantt.resize();
    });
    ro.observe(host);
  }

  window.addEventListener("resize", function () {
    gantt.resize();
  });
})();
