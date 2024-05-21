package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"hq"
	"io/ioutil"
	"lib"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var Token = flag.String("token", "", "jvQuant平台的访问token")
var listen = flag.String("listen", ":31800", "http监听地址")
var initCodesFile = flag.String("initCodesFile", "./data/initCodes.json", "启动即订阅的code")
var saveHqFile = flag.String("saveHqFile", "./data/hq", "行情写入文件,自动加日期后缀。为空则不写入文件。")

var (
	hq            = hangqing.Hq{}
	date          = lib.GetDate()
	saveFile      = ""
	hqWriteChan   = make(chan []byte, 128)
	hqToParseChan = make(chan []byte, 128)
	hqSseChanMap  = sync.Map{}
	logSseChanMap = sync.Map{}
	opLogArr      = []string{}
	stop          = make(chan os.Signal, 1)
)

type initCodeJson struct {
	Lv1 []string `json:"lv1"`
	Lv2 []string `json:"lv2"`
}
type httpEngine struct {
}

//行情回调
func onHqRev(rb []byte) {
	unzip := lib.DeCompress(rb)
	//解析队列
	hqToParseChan <- unzip
	//接收存储
	if saveFile != "" {
		hqWriteChan <- unzip
	}
}

func hqParseService() {
	for rb := range hqToParseChan {
		hqSseChanMap.Range(func(key, value any) bool {
			Chan := key.(chan []byte)
			Chan <- rb
			return true
		})
	}
}

//提示信息回调
func onMsgRev(rb []byte) {
	str := fmt.Sprintf("%s %s %s", lib.GetTime(), "收到应答:", rb)

	//保留日志记录，有新连接发送所有历史log
	opLogArr = append(opLogArr, str)

	//发送至所有log sse
	logSseChanMap.Range(func(key, value any) bool {
		Chan := key.(chan []byte)
		Chan <- []byte(str)
		return true
	})
	log.Println("收到应答:", string(rb))
}

func writeHq() {
	for rb := range hqWriteChan {
		lib.WriteAppend(saveFile, append(rb, '\t'))
	}
}

func main() {
	flag.Parse()
	var err error
	httpHandle := new(httpEngine)
	server := &http.Server{
		Addr:    *listen,
		Handler: httpHandle,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("http服务开启异常：" + err.Error())
			os.Exit(-1)
		}
	}()
	log.Println("行情服务已启动，监听地址:", *listen)

	if *saveHqFile != "" {
		saveFile = *saveHqFile + "." + date + ".txt"
		go func() {
			writeHq()
		}()
	}
	initCodes := initCodeJson{}
	if *initCodesFile != "" {
		initCodes, err = readInitCodes(*initCodesFile)
		if err != nil {
			log.Println("读取预设行情配置异常:", err)
		}
	}
	go func() {
		hqParseService()
	}()

	//初始化行情服务,注册level1行情和level2行情处理方法
	hq.Construct(*Token, "", onHqRev, onMsgRev)
	if len(initCodes.Lv1) != 0 {
		hq.AddLv1(initCodes.Lv1)
	}
	if len(initCodes.Lv2) != 0 {
		hq.AddLv2(initCodes.Lv2)
	}

	go func() {
		hq.Wait()
		stop <- syscall.SIGTERM
	}()

	// 等待信号关闭服务器
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞直到接收到信号
	sig := <-stop
	log.Println("收到退出信号:", sig)
	// 创建一个5秒的超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 优雅关闭服务器，等待所有的活动连接关闭
	log.Println("正在关闭服务...")
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server Shutdown: %v", err)
	}
	log.Println("行情服务已退出")
}

func readInitCodes(path string) (initCode initCodeJson, err error) {
	cfh, err := os.Open(path)
	if err != nil {
		return
	}
	cfc, err := ioutil.ReadAll(cfh)
	if err != nil {
		return
	}
	err = json.Unmarshal(cfc, &initCode)
	if err != nil {
		return
	}
	return
}
func hqSse(w http.ResponseWriter, Chan chan []byte) {
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/event-stream;charset=utf-8")
	w.Header().Set("Connection", "keep-alive")

	//发送占位,避免浏览器空屏
	fmt.Fprintf(w, "\n")
	w.(http.Flusher).Flush()
	//不同于log,行情实时推送,旧数据不推送，以接入时间为起始接收数据。
	for {
		select {
		case <-time.Tick(time.Minute):
			Chan <- []byte{}
		case h, ok := <-Chan:
			if !ok {
				return
			}
			_, err := fmt.Fprintf(w, "%s\n", h)
			if err != nil {
				log.Println("hq sse err:", err)
				return
			}
			w.(http.Flusher).Flush()
		}
	}
}

func logSse(w http.ResponseWriter, Chan chan []byte) {
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/event-stream;charset=utf-8")
	w.Header().Set("Connection", "keep-alive")
	//发送历史log
	history := strings.Join(opLogArr, "\n")
	fmt.Fprintf(w, "%s\n", history)
	w.(http.Flusher).Flush()

	for {
		select {
		case <-time.Tick(time.Minute):
			Chan <- []byte{}
		case h := <-Chan:
			_, err := fmt.Fprintf(w, "%s\n", h)
			if err != nil {
				log.Println("log sse err:", err)
				return
			}
			w.(http.Flusher).Flush()
		}
	}
}

func (engine *httpEngine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lib.SetupCORS(&w)
	rb := []byte{}
	rstMap := map[string]string{}
	path := strings.ToLower(r.URL.Path)
	path = strings.TrimLeft(path, "/")
	pathEx := strings.Split(path, "/")
	ipInfo := strings.Split(r.RemoteAddr, ":")
	clientIp := ipInfo[0]
	_ = clientIp

	r.ParseForm()

	file := r.URL.Query().Get("file") //回放文件
	cmd := r.URL.Query().Get("cmd")   //接收指令
	op := r.URL.Query().Get("op")     //接收控制信息
	rstMap["code"] = "0"
	rstMap["message"] = ""

	if pathEx[0] == "hq" {
		//创建该协程专属管道
		Chan := make(chan []byte, 128)
		//实时行情模式
		if file == "" {
			//注册进全局分发map
			hqSseChanMap.Store(Chan, 1)
		} else { //回放模式不接受实时行情
			//开协程读取文件，注入发送队列
			go func() {
				replayHqFile(file, Chan)
				close(Chan)
			}()
		}
		hqSse(w, Chan)
		//sse服务结束,清除管道
		hqSseChanMap.Delete(Chan)
		if file == "" {
			close(Chan)
		}
		return
	}
	if pathEx[0] == "log" {
		//创建该协程专属管道
		Chan := make(chan []byte, 128)
		logSseChanMap.Store(Chan, 1)
		logSse(w, Chan)
		logSseChanMap.Delete(Chan)
		//sse服务结束,清除管道
		close(Chan)
		return
	}

	if pathEx[0] == "cmd" {
		str := fmt.Sprintf("%s %s %s", lib.GetTime(), "发出指令:", cmd)
		opLogArr = append(opLogArr, str)
		//发送至所有log sse
		logSseChanMap.Range(func(key, value any) bool {
			Chan := key.(chan []byte)
			Chan <- []byte(str)
			return true
		})
		hq.SendRawCmd(cmd)
		rstMap["message"] = cmd
		rb, _ = json.Marshal(rstMap)
		fmt.Fprintf(w, "%s", rb)
		return
	}

	if pathEx[0] == "ctl" { //接收控制
		rstMap["code"] = "0"
		rstMap["message"] = "行情未定义控制:" + op
		if op == "exit" {
			stop <- os.Signal(syscall.SIGTERM)
			rstMap["message"] = "行情服务控制退出:" + op
		}
		rb, _ = json.Marshal(rstMap)
		fmt.Fprintf(w, "%s", rb)
		return
	}

	rstMap["code"] = "-1"
	rstMap["message"] = "操作未定义"
	rb, _ = json.Marshal(rstMap)
	fmt.Fprintf(w, "%s", rb)
}

func replayHqFile(path string, Chan chan []byte) {
	file, err := os.Open(path) // 打开文件
	if err != nil {
		Chan <- []byte(err.Error())
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	br := bufio.NewReader(file)
	for {
		rb, err := br.ReadBytes('\t')
		if err != nil {
			return
		}
		Chan <- rb
	}
}
