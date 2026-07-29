package main

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

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
		return true
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
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		ip := game.ClientIP(c.Request.RemoteAddr, c.GetHeader("X-Forwarded-For"), c.GetHeader("X-Real-IP"))
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
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func webAssetVersion() string {
	h := sha256.New()
	for _, name := range []string{"static/css/style.css", "static/js/app.js"} {
		data, err := webAssets.ReadFile(name)
		if err != nil {
			panic(err)
		}
		_, _ = h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:8])
}
