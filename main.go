package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	tmpl := template.Must(template.ParseFS(webAssets, "templates/*"))
	r.SetHTMLTemplate(tmpl)
	staticFS, err := fs.Sub(webAssets, "static")
	if err != nil {
		log.Fatal(err)
	}
	r.StaticFS("/static", http.FS(staticFS))

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
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
