package main

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"click/game"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

//go:embed templates/* static/*
var webAssets embed.FS

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Non-browser WebSocket clients do not have an Origin.
		}
		u, err := url.Parse(origin)
		return err == nil && u.Host != "" && strings.EqualFold(u.Host, r.Host)
	},
}

func main() {
	assetVersion := webAssetVersion()

	store, err := game.NewStore("data/game.db")
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	engine, err := game.NewEngine(store)
	if err != nil {
		log.Fatal(err)
	}
	hub := game.NewHub(engine)
	engine.SetHub(hub)
	go hub.Run()
	engine.Start()

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' https://um.krjojo.com; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws: wss: https://um.krjojo.com; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		path := c.Request.URL.Path
		switch {
		case path == "/":
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		case strings.HasPrefix(path, "/static/"):
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	})
	tmpl := template.Must(template.ParseFS(webAssets, "templates/*"))
	r.SetHTMLTemplate(tmpl)
	staticFS, err := fs.Sub(webAssets, "static")
	if err != nil {
		log.Fatal(err)
	}
	r.StaticFS("/static", http.FS(staticFS))

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{"AssetVersion": assetVersion})
	})

	r.GET("/ws", func(c *gin.Context) {
		trustProxy := strings.EqualFold(os.Getenv("TRUST_PROXY"), "true")
		ip := game.ClientIP(c.Request.RemoteAddr, c.GetHeader("X-Forwarded-For"), c.GetHeader("X-Real-IP"), trustProxy)
		if !hub.Admit(ip) {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			hub.Release(ip)
			return
		}
		game.ServeWS(hub, conn, ip)
	})

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		engine.ForceSave()
		os.Exit(0)
	}()

	addr := ":3001"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Println("全民打工 running at http://localhost" + addr)
	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func webAssetVersion() string {
	h := sha256.New()
	for _, name := range []string{"static/css/style.css", "static/js/app.js", "static/favicon.avif"} {
		data, err := webAssets.ReadFile(name)
		if err != nil {
			panic(err)
		}
		_, _ = h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:8])
}
