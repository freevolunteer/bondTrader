package lib

import (
	"bufio"
	"bytes"
	"compress/flate"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"
)

const DateTpl = "2006-01-02"
const TimeTpl = "15:04:05"

func SetupCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}
func GetTime() string {
	return time.Now().Format(TimeTpl)
}
func GetDate() string {
	return time.Now().Format(DateTpl)
}
func TimeToStamp(timeStr, layout, loc string) (stamp float64, err error) {
	if loc == "" {
		loc = "Asia/Shanghai"
	}
	if layout == "" {
		layout = DateTpl + " " + TimeTpl
	}
	Loc, _ := time.LoadLocation(loc)
	t1, err := time.ParseInLocation(layout, timeStr, Loc)
	return float64(t1.Unix()), err
}
func DeCompress(b []byte) []byte {
	var buffer bytes.Buffer
	buffer.Write([]byte(b))
	reader := flate.NewReader(&buffer)
	var result bytes.Buffer
	result.ReadFrom(reader)
	reader.Close()
	return result.Bytes()
}
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

func SseGet(url string, ch chan []byte) (err error) {
	var rb []byte
	var resp *http.Response
	resp, err = http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	br := bufio.NewReader(resp.Body)
	for {
		rb, err = br.ReadBytes('\n')
		if err != nil {
			return
		}
		ch <- rb
	}
	return
}

func WriteAppend(filePath string, data []byte) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("file open err:", filePath, err)
		return
	}
	file.Write(data)
	//及时关闭file句柄
	defer file.Close()
}
func WriteCover(filePath string, data []byte) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println("file open err:", filePath, err)
		return
	}
	file.Write(data)
	//及时关闭file句柄
	defer file.Close()
}

func IsStructEmpty(s interface{}) bool {
	return reflect.DeepEqual(s, reflect.Zero(reflect.TypeOf(s)).Interface())
}
