package trade

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

const (
	O_BUY_TYPE  = "证券买入"
	O_SALE_TYPE = "证券卖出"

	O_STATUS_YET         = "未报"
	O_STATUS_NEW         = "已报"
	O_STATUS_DONE        = "已成"
	O_STATUS_CANCEL      = "已撤"
	O_STATUS_PART_DONE   = "部成"
	O_STATUS_PART_CANCEL = "部撤"
)

type Trade struct {
	token  string
	server string
	ticket string
}

//实例初始化
func (trade *Trade) Construct(token, acc, pwd, server, ticket string) {
	trade.token = token

	if server == "" {
		server = trade.initServer()
	}
	trade.server = server

	if ticket == "" && acc != "" && pwd != "" {
		ticket = trade.Login(trade.token, acc, pwd)
	}
	trade.ticket = ticket

	if ticket == "" {
		log.Fatalln("登录失败或未指定ticket,且未配置交易账户")
	}
}

//获取交易服务器地址
func (trade *Trade) initServer() (server string) {
	params := url.Values{
		"market": []string{"ab"},
		"type":   []string{"trade"},
		"token":  []string{trade.token},
	}
	req := "http://jvquant.com/query/server?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 3000)
	if err != nil {
		log.Fatalln("获取交易服务器地址失败:", req, err)
	}
	rspMap := ServerAddrRsp{}
	err = json.Unmarshal(rb, &rspMap)
	if err != nil {
		log.Fatalln("解析交易服务器地址异常:", string(rb), err)
	}
	server = rspMap.Server
	if rspMap.Code != "0" || server == "" {
		log.Fatalln("获取交易服务器地址异常:", string(rb))
	}
	log.Println("获取交易服务器地址成功:", server)
	return
}

func (trade *Trade) GetTicket() string {
	return trade.ticket
}
func (trade *Trade) GetServer() string {
	return trade.server
}

//获取交易凭证
func (trade *Trade) Login(token, acc, pwd string) (ticket string) {
	params := url.Values{
		"token": []string{token},
		"acc":   []string{acc},
		"pass":  []string{pwd},
	}
	log.Println("正在登录柜台...")
	req := trade.server + "/login?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 1000*45)
	if err != nil {
		log.Println("登录柜台失败:", req, err)
	}
	rspMap := LoginRsp{}
	err = json.Unmarshal(rb, &rspMap)
	if err != nil {
		log.Println("解析登录返回失败:", string(rb), err)
	}
	ticket = rspMap.Ticket
	if rspMap.Code != "0" || ticket == "" {
		log.Println("获取交易凭证失败:", string(rb))
	}
	log.Println("获取交易凭证成功:", ticket)
	return
}

//查询持仓列表
func (trade *Trade) CheckHold() (hold HoldRsp) {
	params := url.Values{
		"token":  []string{trade.token},
		"ticket": []string{trade.ticket},
	}
	req := trade.server + "/check_hold?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 1000)
	if err != nil {
		log.Println("获取持仓信息失败:", req, err)
	}
	rspMap := HoldRsp{}
	err = json.Unmarshal(rb, &rspMap)
	if err != nil {
		log.Println("解析持仓信息异常:", string(rb), err)
	}
	hold = rspMap
	return
}

//查询交易列表
func (trade *Trade) CheckOrder() (order OrderRsp) {
	params := url.Values{
		"token":  []string{trade.token},
		"ticket": []string{trade.ticket},
	}
	req := trade.server + "/check_order?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 1000)
	if err != nil {
		log.Println("获取交易信息失败:", req, err)
	}
	rspMap := OrderRsp{}
	err = json.Unmarshal(rb, &rspMap)
	if err != nil {
		log.Println("解析交易信息异常:", string(rb), err)
	}
	order = rspMap
	return
}

//委托买入
func (trade *Trade) Buy(code, name, price, vol string) (rsp TradeRsp) {
	params := url.Values{
		"token":  []string{trade.token},
		"ticket": []string{trade.ticket},
		"code":   []string{code},
		"name":   []string{name},
		"price":  []string{price},
		"volume": []string{vol},
	}
	req := trade.server + "/buy?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 1000)
	if err != nil {
		log.Println("买单委托失败:", req, err)
	}
	rspMap := TradeRsp{}
	err = json.Unmarshal(rb, &rspMap)
	if err != nil {
		log.Println("解析买单返回异常:", string(rb), err)
	}
	rsp = rspMap
	return
}

//委托卖出
func (trade *Trade) Sale(code, name, price, vol string) (rsp TradeRsp) {
	params := url.Values{
		"token":  []string{trade.token},
		"ticket": []string{trade.ticket},
		"code":   []string{code},
		"name":   []string{name},
		"price":  []string{price},
		"volume": []string{vol},
	}
	req := trade.server + "/sale?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 1000)
	if err != nil {
		log.Println("卖单委托失败:", req, err)
	}
	rspMap := TradeRsp{}
	err = json.Unmarshal(rb, &rspMap)
	if err != nil {
		log.Println("解析卖单返回异常:", string(rb), err)
	}
	rsp = rspMap
	return
}

//撤单
func (trade *Trade) Cancel(orderId string) (rsp CancelRsp) {
	params := url.Values{
		"token":    []string{trade.token},
		"ticket":   []string{trade.ticket},
		"order_id": []string{orderId},
	}
	req := trade.server + "/cancel?" + params.Encode()
	rb, err := HttpOnce(req, nil, nil, 1000)
	if err != nil {
		log.Println("撤单失败:", req, err)
	}
	rspMap := CancelRsp{}
	err = json.Unmarshal(rb, &rspMap)
	if err != nil {
		log.Println("解析撤单返回异常:", string(rb), err)
	}
	rsp = rspMap
	return
}

type ServerAddrRsp struct {
	Code   string `json:"code"`
	Server string `json:"server"`
}

type LoginRsp struct {
	Code   string  `json:"code"`
	Msg    string  `json:"msg"`
	Ticket string  `json:"ticket"`
	Expire float64 `json:"expire,string"`
}

type HoldRsp struct {
	Code     string     `json:"code"`
	Message  string     `json:"message"`
	Total    float64    `json:"total,string"`
	Usable   float64    `json:"usable,string"`
	DayEarn  float64    `json:"day_earn,string"`
	HoldEarn float64    `json:"hold_earn,string"`
	HoldList []HoldItem `json:"hold_list"`
}
type HoldItem struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	HoldVol   float64 `json:"hold_vol,string"`
	UsableVol float64 `json:"usable_vol,string"`
	HoldEarn  float64 `json:"hold_earn,string"`
	DayEarn   float64 `json:"day_earn,string"`
}

type OrderRsp struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	List    []OrderItem `json:"list"`
}

type OrderItem struct {
	OrderID     string  `json:"order_id"`
	Day         string  `json:"day"`
	Time        string  `json:"time"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	OrderPrice  float64 `json:"order_price,string"`
	OrderVolume float64 `json:"order_volume,string"`
	DealPrice   float64 `json:"deal_price,string"`
	DealVolume  float64 `json:"deal_volume,string"`
}

type CancelRsp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TradeRsp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	OrderId string `json:"order_id"`
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
