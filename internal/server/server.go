// Package server wires together routing, templates, and HTTP handlers.
package server

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"gantt/internal/charts"
	_ "gantt/internal/charts/gantt"
)

// Server holds the Gin engine.
type Server struct {
	engine *gin.Engine
}

// New creates a configured Server using the provided embedded file system.
func New(assets fs.FS) *Server {
	r := gin.Default()

	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		log.Fatal("embedded FS missing 'static/' directory:", err)
	}
	r.StaticFS("/static", http.FS(staticFS))
	r.SetHTMLTemplate(template.Must(template.ParseFS(assets, "templates/*.tmpl")))

	h := &handlers{assets: assets}
	r.GET("/", h.entry)

	gantt := r.Group("/gantt")
	{
		gantt.GET("", h.ganttHome)
		gantt.GET("/demo", h.ganttDemo)
		gantt.GET("/clear", h.ganttClear)
		gantt.POST("/upload", h.ganttUpload)
		gantt.POST("/chart", h.ganttChart)
	}

	viz := r.Group("/viz")
	{
		viz.GET("", h.vizHome)
		viz.GET("/demo", h.vizDemo)
		viz.GET("/clear", h.vizClear)
		viz.POST("/upload", h.vizUpload)
		viz.POST("/chart", h.vizChart)
	}

	_ = charts.All()

	return &Server{engine: r}
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}
