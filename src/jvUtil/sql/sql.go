package sql

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

type ServerAddrRsp struct {
	Code   string `json:"code"`
	Server string `json:"server"`
}

type Sql struct {
	token  string //jvQuant Token
	server string
}

//初始化
func (sql *Sql) Construct(token, server string) {
	sql.token = token
	sql.server = server
	if server == "" {
		sql.server = sql.initServer()
	}
}

//获取数据库地址
func (sql *Sql) initServer() (server string) {
	params := url.Values{
		"market": []string{"ab"},
		"type":   []string{"sql"},
		"token":  []string{sql.token},
	}
	req := "http://jvquant.com/query/server?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 3000)
	if err != nil {
		log.Fatalln("获取SQL服务器地址失败:", req, err)
	}
	rspMap := ServerAddrRsp{}
	err = json.Unmarshal(rb, &rspMap)
	if err != nil {
		log.Fatalln("解析SQL服务器地址异常:", string(rb), err)
	}
	server = rspMap.Server
	if rspMap.Code != "0" || server == "" {
		log.Fatalln("获取SQL服务器地址异常:", string(rb))
	}
	log.Println("获取SQL服务器地址成功:", server)
	return
}

//语义泛查
func (sql *Sql) Query(query, page string) (rb []byte) {
	params := url.Values{
		"token": []string{sql.token},
		"mode":  []string{"sql"},
		"query": []string{query},
		"page":  []string{page},
	}
	req := sql.server + "/sql?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 1000)
	if err != nil {
		log.Println("SQL请求泛查异常:", req, err)
	}
	return
}

//获取K线
func (sql *Sql) Kline(code, cate, _type, fq, limit string) (rb []byte) {
	params := url.Values{
		"token": []string{sql.token},
		"mode":  []string{"kline"},
		"cate":  []string{cate},
		"code":  []string{code},
		"type":  []string{_type},
		"fq":    []string{fq},
		"limit": []string{limit},
	}
	req := sql.server + "/sql?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 1000)
	if err != nil {
		log.Println("SQL请求K线异常:", req, err)
	}
	return
}

//获取可转债信息
func (sql *Sql) Bond() (rb []byte) {
	params := url.Values{
		"token": []string{sql.token},
		"mode":  []string{"bond"},
	}
	req := sql.server + "/sql?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 1000)
	if err != nil {
		log.Println("SQL请求转债信息异常:", req, err)
	}
	return
}

//获取行业分类信息
func (sql *Sql) Industry() (rb []byte) {
	params := url.Values{
		"token": []string{sql.token},
		"mode":  []string{"industry"},
	}
	req := sql.server + "/sql?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 1000)
	if err != nil {
		log.Println("SQL请求行业分类异常:", req, err)
	}
	return
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
