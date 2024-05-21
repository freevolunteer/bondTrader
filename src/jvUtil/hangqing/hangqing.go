package hangqing

import (
	"bufio"
	"bytes"
	"compress/flate"
	"encoding/json"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type ServerAddrRsp struct {
	Code   string `json:"code"`
	Server string `json:"server"`
}

type Hq struct {
	token     string          //jvQuant Token
	server    string          //websocket服务器地址
	conn      *websocket.Conn //websocket连接
	cmdChan   chan string
	exitChan  chan int
	msgHandle func([]byte) //非行情接收处理方法
	hqHandle  func([]byte) //行情接收处理方法
	wg        *sync.WaitGroup
}

//实例初始化
func (hq *Hq) Construct(token, server string, hqHandle, msgHandle func([]byte)) {
	hq.token = token
	if server == "" {
		hq.server = hq.initServer()
	}
	hq.hqHandle = hqHandle
	hq.msgHandle = msgHandle
	hq.conn = hq.connect()
	hq.wg = &sync.WaitGroup{}
	hq.cmdChan = make(chan string, 128)
	hq.exitChan = make(chan int)

	//接收协程
	hq.wg.Add(2)
	go func() {
		hq.receive()
		hq.wg.Done()
	}()
	//发送协程
	go func() {
		hq.cmd()
		hq.wg.Done()
	}()
}

//获取行情服务器地址
func (hq *Hq) initServer() (server string) {
	params := url.Values{
		"market": []string{"ab"},
		"type":   []string{"websocket"},
		"token":  []string{hq.token},
	}
	req := "http://jvquant.com/query/server?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 3000)
	if err != nil {
		log.Fatalln("获取行情服务器地址失败:", req, err)
	}
	rspMap := ServerAddrRsp{}
	err = json.Unmarshal(rb, &rspMap)
	if err != nil {
		log.Fatalln("解析行情服务器地址失败:", string(rb), err)
	}
	server = rspMap.Server
	if rspMap.Code != "0" || server == "" {
		log.Fatalln("解析行情服务器地址失败:", string(rb))
	}
	log.Println("获取行情服务器地址成功:", server)
	return
}

//连接行情服务器
func (hq Hq) connect() (conn *websocket.Conn) {
	wsUrl := hq.server + "?token=" + hq.token
	conn, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)
	if err != nil {
		log.Fatalln("行情服务器连接错误：", err)
	}
	return
}

//增加level1行情订阅
func (hq Hq) AddLv1(codeArr []string) {
	cmd := "add="
	cmdArr := []string{}
	for _, code := range codeArr {
		cmdArr = append(cmdArr, "lv1_"+code)
	}
	cmd = cmd + strings.Join(cmdArr, ",")
	hq.SendRawCmd(cmd)
}

//取消level1行情订阅
func (hq Hq) DelLv1(codeArr []string) {
	cmd := "del="
	cmdArr := []string{}
	for _, code := range codeArr {
		cmdArr = append(cmdArr, "lv1_"+code)
	}
	cmd = cmd + strings.Join(cmdArr, ",")
	hq.SendRawCmd(cmd)
}

//增加level2行情订阅
func (hq Hq) AddLv2(codeArr []string) {
	cmd := "add="
	cmdArr := []string{}
	for _, code := range codeArr {
		cmdArr = append(cmdArr, "lv2_"+code)
	}
	cmd = cmd + strings.Join(cmdArr, ",")
	hq.SendRawCmd(cmd)
}

//取消level2行情订阅
func (hq Hq) DelLv2(codeArr []string) {
	cmd := "del="
	cmdArr := []string{}
	for _, code := range codeArr {
		cmdArr = append(cmdArr, "lv2_"+code)
	}
	cmd = cmd + strings.Join(cmdArr, ",")
	hq.SendRawCmd(cmd)
}

//指令入队列
func (hq Hq) SendRawCmd(cmd string) {
	hq.cmdChan <- cmd
	log.Println("发送指令:" + cmd)
}

//关闭行情连接
func (hq Hq) Close() {
	close(hq.cmdChan)
	hq.exitChan <- 1
	hq.conn.Close()
}

//线程阻塞等待
func (hq Hq) Wait() {
	hq.wg.Wait()
}

//websocket指令发送
func (hq Hq) cmd() {
	for {
		select {
		case <-hq.exitChan:
			log.Print("发送协程主动退出")
			return
		default:
			for cmd := range hq.cmdChan {
				err := hq.conn.WriteMessage(websocket.TextMessage, []byte(cmd))
				if err != nil {
					log.Println("指令发送错误退出：", err)
					return
				}
			}
		}
	}

}

//websocket行情接收处理
func (hq Hq) receive() {
	for {
		select {
		case <-hq.exitChan:
			log.Print("接收协程主动退出")
			return
		default:
			//阻塞接收
			messageType, rb, err := hq.conn.ReadMessage()
			if err != nil {
				log.Print("接收协程被动退出：", err)
				hq.Close()
				return
			}
			//文本消息
			if messageType == websocket.TextMessage {
				hq.msgHandle(rb)
			}
			//二进制消息
			if messageType == websocket.BinaryMessage {
				hq.hqHandle(rb)
			}
		}
	}
}

//二进制数据解压方法
func DeCompress(b []byte) []byte {
	var buffer bytes.Buffer
	buffer.Write([]byte(b))
	reader := flate.NewReader(&buffer)
	var result bytes.Buffer
	result.ReadFrom(reader)
	reader.Close()
	return result.Bytes()
}

//http请求封装
func HttpOnce(Url string, headers, postData map[string]string, msTimeOut int) (r []byte, err error) {
	client := &http.Client{
		Timeout: time.Duration(time.Duration(msTimeOut) * time.Millisecond),
	}
	method := http.MethodGet
	r = []byte{}
	err = nil
	if len(headers) == 0 {
		headers = map[string]string{}
	}
	if len(postData) != 0 {
		method = http.MethodPost
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}

	postParam := url.Values{}
	for k, v := range postData {
		postParam.Set(k, v)
	}
	postParamBuff := bytes.NewBufferString(postParam.Encode())
	req, err := http.NewRequest(method, Url, postParamBuff)

	if err != nil {
		return r, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	resp, er := client.Do(req)
	if er != nil {
		err = er
		return
	}
	defer resp.Body.Close()
	if err != nil {
		return r, err
	}
	br := bufio.NewReader(resp.Body)
	r, err = ioutil.ReadAll(br)
	return r, err
}
