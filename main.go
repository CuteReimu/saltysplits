package main

import (
	"bufio"
	"embed"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// serverAddr 是服务器监听的地址.
	serverAddr = "127.0.0.1:12334"
	// serverURL 是要在浏览器中打开的完整 URL.
	serverURL = "http://127.0.0.1:12334/"
)

var (
	//go:embed static
	htmlFiles embed.FS
	//go:embed index.html
	htmlIndex []byte
	//go:embed app.js
	appJs []byte
	//go:embed app.css
	appCss []byte

	// fileName 是要分析的 .lss 文件的路径.
	fileName = flag.String("i", "", "指定要分析的 .lss 文件路径")
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

	fn := getFileName()

	buf, err := os.ReadFile(fn)
	if err != nil {
		panic(err)
	}

	var run xmlRun
	if err := xml.Unmarshal(buf, &run); err != nil {
		panic(err)
	}

	startAttemptId := 0
	if len(run.Attempt) > LargeFileThreshold {
		fmt.Printf("该文件包含 %d 次尝试，你可以指定一个起始尝试ID以缩小分析范围: \n", len(run.Attempt))

		_, _ = fmt.Scanln(&startAttemptId)
		fmt.Printf("仅分析ID大于或等于 %d 的尝试...\n", startAttemptId)
	}

	analyzer := NewAnalyzer(&run, startAttemptId)
	if err := analyzer.Analyze(); err != nil {
		panic(err)
	}

	fmt.Println("请打开浏览器访问 " + serverURL + " 查看分析结果")
	openBrowser(serverURL)

	initWebUi(analyzer)

	select {}
}

func getFileName() string {
	if *fileName != "" {
		return *fileName
	}

	fmt.Println("将你的 .lss 文件拖进来，然后按回车键开始分析...")

	reader := bufio.NewReader(os.Stdin)

	line, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	name := strings.TrimSpace(line)
	// Handle quotes added by some shells when dragging files
	if strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"") {
		name = name[1 : len(name)-1]
	}
	// Handle single quotes as well
	if strings.HasPrefix(name, "'") && strings.HasSuffix(name, "'") {
		name = name[1 : len(name)-1]
	}

	return name
}

func openBrowser(url string) {
	var err error
	switch strings.ToLower(runtime.GOOS) {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = errors.New("不支持的平台")
	}

	if err != nil {
		fmt.Printf("无法自动打开浏览器: %v\n", err)
	}
}

func initWebUi(analyzer *Analyzer) {
	gin.SetMode(gin.ReleaseMode)

	g := gin.New()
	g.Use(gin.Recovery()) // Add recovery middleware

	g.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", htmlIndex)
	})
	g.GET("/app.js", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/javascript; charset=utf-8", appJs)
	})
	g.GET("/app.css", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/css; charset=utf-8", appCss)
	})
	g.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, analyzer.Run)
	})
	g.GET("/summary", func(c *gin.Context) {
		c.JSON(http.StatusOK, analyzer.Summary)
	})
	g.GET("/totalData", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"realTime": analyzer.RealTimeTotalData,
			"gameTime": analyzer.GameTimeTotalData,
		})
	})
	g.GET("/reset", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"realTime":    analyzer.RealTimeReset,
			"gameTime":    analyzer.GameTimeReset,
			"realTimeBig": analyzer.RealTimeResetBig,
			"gameTimeBig": analyzer.GameTimeResetBig,
			"disable":     analyzer.DisableShowBigSegment,
		})
	})
	g.GET("/breakdown", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"segments": analyzer.RunBreakdownSegments,
			"data":     analyzer.RunBreakdown,
		})
	})
	g.GET("/segment", func(c *gin.Context) {
		segmentHandler(c, analyzer)
	})
	g.StaticFS("/x/", http.FS(htmlFiles))

	go func() {
		if err := g.Run(serverAddr); err != nil {
			panic(err)
		}
	}()
}

func segmentHandler(c *gin.Context, analyzer *Analyzer) {
	indexStr := c.Query("index")

	i, err := strconv.Atoi(indexStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的索引"})
		return
	}

	result, err := analyzer.GetSegment(i)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
