package main

import (
	"embed"
	"flag"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"

	"github.com/gin-gonic/gin"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			fmt.Println(string(debug.Stack()))
			fmt.Println("报错了！按回车键退出...")

			_, _ = fmt.Scanln()
		}
	}()

	flag.Parse()
	analysis()
	initWebUi()

	select {}
}

var (
	//go:embed static
	htmlFiles embed.FS
	//go:embed index.html
	htmlIndex []byte
)

func initWebUi() {
	gin.SetMode(gin.ReleaseMode)

	g := gin.New()
	g.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", htmlIndex)
	})
	g.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, run)
	})
	g.GET("/summary", func(c *gin.Context) {
		c.JSON(http.StatusOK, summary)
	})
	g.GET("/totalData", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"realTime": realTimeTotalData,
			"gameTime": gameTimeTotalData,
		})
	})
	g.GET("/reset", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"realTime": realTimeReset,
			"gameTime": gameTimeReset,
		})
	})
	g.GET("/breakdown", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"segments": runBreakdownSegments,
			"data":     runBreakdown,
		})
	})
	g.GET("/segment", func(c *gin.Context) {
		index := c.Query("index")

		i, err := strconv.Atoi(index)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid index"})
			return
		}

		result, err := getSegment(i)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		c.JSON(http.StatusOK, result)
	})
	g.StaticFS("/x/", http.FS(htmlFiles))

	go func() {
		if err := g.Run("127.0.0.1:12334"); err != nil {
			panic(err)
		}
	}()
}
