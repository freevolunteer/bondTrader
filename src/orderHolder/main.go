package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"lib"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"trade"
)

/**
买单分类:
指定价，超时内能成则成，否则撤单
已报:待成
部成:已成部分，剩余待成
部撤:已成部分,剩余已撤
已撤:未成已撤

卖单分类:
强卖，每次挂单超时未成，则撤单再卖
已报:待成
部成:已成部分，剩余待成
部撤:已成部分,剩余已撤,需要重新挂卖单
已撤:未成已撤,需要重新挂卖单
*/

var (
	listen        = flag.String("listen", ":31888", "http服务器监听地址,如127.0.0.1:8080")
	orderInterval = flag.Int("orderInterval", 2, "委托单查询间隔")
	holdInterval  = flag.Int("holdInterval", 5, "手动清仓检测间隔")
	Token         = flag.String("token", "", "jvToken")
	tdAcc         = flag.String("acc", "", "资金账户")
	tdPwd         = flag.String("pwd", "", "资金密码")
	ticketFile    = flag.String("ticketFile", "./data/ticket.tmp", "交易凭证临时存储文件")
	orderLog      = flag.String("orderLog", "./data/orders.json", "服务退出时保留的交易历史文件,自带时间日期后缀")
)

var (
	td            = trade.Trade{}
	toCheckOidMap = sync.Map{}
	keyItemMap    = sync.Map{}
	stop          = make(chan os.Signal, 1)
)

type oidCheckMap struct {
	Key     string `json:"key"`
	Type    string `json:"type"`
	InTime  string `json:"inTime"`  //记录进入队列的时间
	InStamp int64  `json:"inStamp"` //记录进入队列的时间
	Oid     string `json:"oid"`
	Timeout int64  `json:"timeout"` //超时则撤，如果未超时期间接收到卖出请求，则触发撤单；待撤单后触发回调
	Cb      string `json:"cb"`      //委托退出机制触发，已成已撤
}

type keyStatusItem struct {
	Key      string  `json:"key"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	BStatus  string  `json:"b_status"`
	BOPrice  float64 `json:"b_oPrice"`
	BOVolume float64 `json:"b_oVolume"`
	BDPrice  float64 `json:"b_dPrice"`
	BDVolume float64 `json:"b_dVolume"`
	BOTime   string  `json:"b_oTime"`
	BOStamp  int64   `json:"b_oStamp"`
	BDStamp  int64   `json:"b_dStamp"`
	SStatus  string  `json:"s_status"`
	SOPrice  float64 `json:"s_oPrice"`
	SOVolume float64 `json:"s_oVolume"`
	SDPrice  float64 `json:"s_dPrice"`
	SDVolume float64 `json:"s_dVolume"`
	SOStamp  int64   `json:"s_oStamp"`
	SDStamp  int64   `json:"s_dStamp"`
	SDTime   string  `json:"s_dTime"`
	Earn     float64 `json:"earn"`
	BOid     string  `json:"boid"`
	FsOid    string  `json:"fsoid"` //首次卖单id,用来判断后续卖单是否为撤单再卖
	SOid     string  `json:"soid"`
}

type httpEngine struct {
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
	var (
		key     = r.URL.Query().Get("key")
		code    = r.URL.Query().Get("code")
		name    = r.URL.Query().Get("name")
		price   = r.URL.Query().Get("price")
		vol     = r.URL.Query().Get("vol")
		all     = r.URL.Query().Get("all")
		timeOut = r.URL.Query().Get("timeout")
		cb      = r.URL.Query().Get("cb")
		oid     = r.URL.Query().Get("order_id")
		op      = r.URL.Query().Get("op") //接收控制信息
	)

	rstMap["code"] = "0"
	rstMap["message"] = ""
	if pathEx[0] == "buy" {
		rsp := buy(key, code, name, price, vol, cb, timeOut)
		rb, _ = json.Marshal(rsp)
		fmt.Fprintf(w, "%s", rb)
		return
	}

	if pathEx[0] == "sale" {
		rsp := sale(key, code, name, price, vol, cb, timeOut)
		rb, _ = json.Marshal(rsp)
		fmt.Fprintf(w, "%s", rb)
		return
	}

	if pathEx[0] == "cancel" {
		if oid == "" {
			rstMap["message"] = "请填入需撤单号"
			rb, _ = json.Marshal(rstMap)
			fmt.Fprintf(w, "%s", rb)
			return
		}
		rsp := td.Cancel(oid)
		rb, _ = json.Marshal(rsp)
		fmt.Fprintf(w, "%s", rb)
		if rsp.Code != "0" {
			log.Println("撤单异常:", oid, rsp.Message)
		}
		return
	}

	if pathEx[0] == "get" {
		list := getItem(all)
		rb, _ = json.Marshal(list)
		fmt.Fprintf(w, "%s", rb)
		return
	}

	if pathEx[0] == "check" {
		list := getCheck()
		rb, _ = json.Marshal(list)
		fmt.Fprintf(w, "%s", rb)
		return
	}

	if pathEx[0] == "ctl" { //接收控制
		rstMap["code"] = "0"
		rstMap["message"] = "交易未定义控制:" + op
		if op == "exit" {
			stop <- os.Signal(syscall.SIGTERM)
			rstMap["message"] = "交易服务控制退出:" + op
		}
		rb, _ = json.Marshal(rstMap)
		fmt.Fprintf(w, "%s", rb)
		return
	}

	rstMap["code"] = "-1"
	rstMap["message"] = "操作未定义"
	rb, _ = json.Marshal(rstMap)
	fmt.Fprintf(w, "%s", rb)
	return
}

func main() {
	startTime := lib.GetDate() + "_" + lib.GetTime()
	//运行参数获取
	flag.Parse()

	//http服务器初始化
	httpHandle := new(httpEngine)
	server := &http.Server{
		Addr:    *listen,
		Handler: httpHandle,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalln("http服务开启异常：" + err.Error())
		}
	}()
	log.Println("http服务已启动，监听地址:", *listen)

	go func() {
		tradeTicketService()
	}()

	go func() {
		orderWatchService()
	}()

	go func() {
		holdWatchService()
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

	exitTime := lib.GetDate() + "_" + lib.GetTime()
	if *orderLog != "" {
		saveFile := fmt.Sprintf("%s.%s~%s", *orderLog, startTime, exitTime)
		rst := getItem("1")
		if len(rst) != 0 {
			jb, _ := json.Marshal(rst)
			lib.WriteCover(saveFile, jb)
			log.Println("order信息已保存:", saveFile)
		}
	}
	log.Println("交易服务已退出")

}

func tradeTicketService() {
	for {
		nowStamp := time.Now().Unix()
		expire := int64(9000)
		rsp, _ := readTicket(*ticketFile)
		ticket := ""
		server := ""
		lastTicketStamp := int64(0)
		if len(rsp) == 3 {
			ticket = rsp[0]
			lastTicketStampInt, _ := strconv.Atoi(rsp[1])
			lastTicketStamp = int64(lastTicketStampInt)
			server = rsp[2]
		}
		update := false
		if nowStamp-lastTicketStamp >= expire-60 { //提前60s刷新
			ticket = ""
			server = ""
			update = true
			log.Println("即将动态刷新交易凭证...")
		}

		td.Construct(*Token, *tdAcc, *tdPwd, server, ticket)

		if update { //有更新
			rsp = []string{}
			rsp = append(rsp, td.GetTicket())
			rsp = append(rsp, strconv.Itoa(int(nowStamp)))
			rsp = append(rsp, td.GetServer())
			js, _ := json.Marshal(rsp)
			lib.WriteCover(*ticketFile, js)
		}
		time.Sleep(time.Second * 30)
	}
}

func readTicket(path string) (rsp []string, err error) {
	cfh, err := os.Open(path)
	if err != nil {
		return
	}
	cfc, err := ioutil.ReadAll(cfh)
	if err != nil {
		return
	}
	err = json.Unmarshal(cfc, &rsp)
	return
}

//买单只支持一次性下单,超时未成的部分撤掉
func buy(key, code, name, price, vol, cb, timeout string) (rsp trade.TradeRsp) {
	rsp.Code = "-1"
	if _, ok := keyItemMap.Load(key); ok {
		rsp.Message = fmt.Sprintf("该key已存在,请更换key后再试:%s %s %s", code, name, key)
		return
	}
	nowInt64 := time.Now().Unix()
	rsp = td.Buy(code, name, price, vol)
	timeoutInt, err := strconv.Atoi(timeout)
	if err != nil || timeoutInt < 3 { //默认3秒
		timeoutInt = 3
	}
	if rsp.Code == "0" && rsp.OrderId != "" {
		keyItem := &keyStatusItem{}
		keyItem.Key = key
		keyItem.Code = code
		keyItem.Name = name
		keyItem.BOid = rsp.OrderId
		keyItem.BStatus = trade.O_STATUS_YET
		keyItem.BOStamp = nowInt64
		keyItem.BOTime = lib.GetTime()
		keyItemMap.Store(key, keyItem) //买入撤单后重挂需更换key
		toCheckOidMap.Store(rsp.OrderId, oidCheckMap{Key: key, Type: trade.O_BUY_TYPE, InStamp: nowInt64, InTime: lib.GetTime(), Oid: rsp.OrderId, Timeout: int64(timeoutInt), Cb: cb})
		log.Printf("买单请求 %s,code:%s,price:%s,vol:%s,timeout:%d,单号:%s", name, code, price, vol, timeoutInt, rsp.OrderId)
	} else {
		log.Printf("买单异常 %s,code:%s,price:%s,vol:%s,timeout:%d,异常:%s", name, code, price, vol, timeoutInt, rsp.Message)
		return
	}
	return
}

//卖单支持撤单后再卖,累计卖出量为买入量即为全卖
func sale(key, code, name, price, vol, cb, timeout string) (rsp trade.TradeRsp) {
	rsp.Code = "-1"
	vT, ok := keyItemMap.Load(key)
	if !ok {
		rsp.Message = fmt.Sprintf("不存在该key,请检查后再试:%s %s %s", code, name, key)
		return
	}
	keyItem := vT.(*keyStatusItem)

	//买卖必须为同一code
	if keyItem.Code != code {
		rsp.Message = fmt.Sprintf("code(%s)与该key持仓code(%s)不匹配,请检查后再试:%s", code, keyItem.Code, key)
		return
	}

	if keyItem.SStatus == trade.O_STATUS_NEW || keyItem.SStatus == trade.O_STATUS_PART_DONE || keyItem.SStatus == trade.O_STATUS_YET {
		rsp.Message = fmt.Sprintf("该key已有卖单在处理,请撤单完成后再试:%s %s %s", code, name, key) //需显式撤单
		return
	}

	nowInt64 := time.Now().Unix()
	rsp = td.Sale(code, name, price, vol)
	timeoutInt, err := strconv.Atoi(timeout)
	if err != nil || timeoutInt < 3 { //默认3秒
		timeoutInt = 3
	}

	if rsp.Code == "0" && rsp.OrderId != "" {
		if keyItem.FsOid == "" { //记录首次触发卖出单号
			keyItem.FsOid = rsp.OrderId
		}
		keyItem.SOStamp = nowInt64
		keyItem.SStatus = trade.O_STATUS_YET
		keyItem.SOid = rsp.OrderId //记录最后一次卖单id,下次触发卖出时自动撤单用
		keyItemMap.Store(key, keyItem)
		toCheckOidMap.Store(rsp.OrderId, oidCheckMap{Key: key, Type: trade.O_SALE_TYPE, InStamp: nowInt64, InTime: lib.GetTime(), Oid: rsp.OrderId, Timeout: int64(timeoutInt), Cb: cb})
		log.Printf("卖单请求 %s,code:%s,price:%s,vol:%s,timeout:%d,单号:%s", name, code, price, vol, timeoutInt, rsp.OrderId)
	} else {
		log.Printf("卖单异常 %s,code:%s,price:%s,vol:%s,timeout:%d,异常:%s", name, code, price, vol, timeoutInt, rsp.Message)
		return
	}
	return
}

//for http debug
func getItem(all string) (rst []keyStatusItem) {
	rst = []keyStatusItem{}
	keyItemMap.Range(func(key, value any) bool {
		item := value.(*keyStatusItem)
		if all == "" && item.BStatus == trade.O_STATUS_CANCEL {
			return true
		}
		rst = append(rst, *item)
		return true
	})

	//按买单时间倒序
	rsort := true
	sort.Slice(rst, func(i, j int) bool {
		if rsort {
			return rst[i].BOStamp > rst[j].BOStamp
		} else {
			return rst[i].BOStamp < rst[j].BOStamp
		}
	})
	return
}
func getCheck() (rst []oidCheckMap) {
	rst = []oidCheckMap{}
	toCheckOidMap.Range(func(key, value any) bool {
		item := value.(oidCheckMap)
		rst = append(rst, item)
		return true
	})
	return
}

func orderWatchService() {
	for {
		//map快照,oid存在就必查,及时获取委托成交状态
		oidMap := map[string]oidCheckMap{}
		toCheckOidMap.Range(func(key, value any) bool {
			oid := key.(string)
			oMap := value.(oidCheckMap)
			oidMap[oid] = oMap
			return true
		})
		timeStr := lib.GetTime()
		if len(oidMap) == 0 || !inTradeTime(timeStr) { //时间排除
			time.Sleep(time.Second)
			continue
		}

		orderInfo := td.CheckOrder()
		nowInt64 := time.Now().Unix()
		if orderInfo.Code != "0" {
			log.Println("查询交易异常:", orderInfo.Message)
			time.Sleep(time.Duration(*orderInterval) * time.Second)
			continue
		}

		//转换oid索引
		orderIdMap := map[string]trade.OrderItem{}
		for _, item := range orderInfo.List {
			orderIdMap[item.OrderID] = item
		}

		for oid, oMap := range oidMap {
			oInfo, ok := orderIdMap[oid]
			if !ok {
				log.Println("未查询到order信息:", oid)
				continue
			}
			vT, ok := keyItemMap.Load(oMap.Key)
			if !ok {
				log.Println("未查询到keyItem信息:", oMap.Key, oid)
				continue
			}
			keyItem := vT.(*keyStatusItem)

			//无论成交状态,更新keyItem
			if oInfo.Type == trade.O_BUY_TYPE { //买入
				keyItem.Name = oInfo.Name
				keyItem.BStatus = oInfo.Status
				keyItem.BOPrice = oInfo.OrderPrice
				keyItem.BOVolume = oInfo.OrderVolume
				keyItem.BDPrice = oInfo.DealPrice
				keyItem.BDVolume = oInfo.DealVolume
			}

			if oInfo.Type == trade.O_SALE_TYPE { //卖出
				keyItem.SStatus = oInfo.Status
				keyItem.SOPrice = oInfo.OrderPrice
				keyItem.SOVolume = oInfo.OrderVolume
				keyItem.SDPrice = oInfo.DealPrice
			}

			//全成全撤或部撤,委托完结,发起回调,清除map,下次无需再查。卖出未成再卖的逻辑需要回调处处理,再次调用卖出
			if oInfo.Status == trade.O_STATUS_DONE || oInfo.Status == trade.O_STATUS_CANCEL || oInfo.Status == trade.O_STATUS_PART_CANCEL {
				//先写后调,回调失败不再重复
				if oInfo.Type == trade.O_BUY_TYPE {
					keyItem.BDStamp = nowInt64
				}
				if oInfo.Type == trade.O_SALE_TYPE {
					keyItem.SDStamp = nowInt64
					keyItem.SDTime = lib.GetTime()
					if keyItem.FsOid != keyItem.SOid { //撤单再卖的情况，累加已卖成量
						keyItem.SDVolume += oInfo.DealVolume
					} else { //首次卖出
						keyItem.SDVolume = oInfo.DealVolume
					}
					//按本次卖出数量累加收益
					sigEarn := (keyItem.SDPrice - keyItem.BDPrice) * oInfo.DealVolume
					keyItem.Earn = math.Round((keyItem.Earn+sigEarn)*100) / 100
				}

				js, _ := json.Marshal(keyItem)
				//key相关信息Post回传
				postData := map[string]string{"data": string(js)}
				//oMap.Cb = oMap.Cb + "?&data=" + string(js)
				rb, err := lib.HttpOnce(oMap.Cb, nil, postData, 1000)
				if err != nil { //可以增加响应内容判断回调是否成功
					log.Printf("%s %s %s,回调失败:%s,回调地址:%s\n", oInfo.Type, oInfo.Type, oInfo.Status, err, oMap.Cb)
					//回调失败会在下次检查继续尝试回调，直至成功
					//continue
				} else {
					log.Printf("%s %s %s,回调完成:%s,回调地址:%s\n", oInfo.Type, oInfo.Name, oInfo.Status, rb, oMap.Cb)
				}

				//从map删除,下次不再检查
				toCheckOidMap.Delete(oid)
			}

			//超时,未全成,发起撤单
			if nowInt64-oMap.InStamp >= oMap.Timeout && (oInfo.Status == trade.O_STATUS_YET || oInfo.Status == trade.O_STATUS_NEW || oInfo.Status == trade.O_STATUS_PART_DONE) {
				log.Printf("%s %s,单号:%s,状态:%s,%d秒后未成自动撤单\n", oInfo.Type, oInfo.Name, oid, oInfo.Status, oMap.Timeout)
				r := td.Cancel(oid)
				if r.Code != "0" {
					log.Printf("%s %s撤单异常:%s,%s\n", oInfo.Type, oInfo.Name, oid, r.Message)
				}
			}
			keyItemMap.Store(oMap.Key, keyItem)
		}

		//周期执行
		time.Sleep(time.Duration(*orderInterval) * time.Second)
	}

}

//检测自动买入的单是否通过其他终端清仓
func holdWatchService() {
	for {
		nowInt64 := time.Now().Unix()
		needCheckKeys := []string{}
		keyItemMap.Range(func(key, value any) bool {
			keyItem := value.(*keyStatusItem)
			//买入,但未卖,且距买入5秒以上 ,todo或者卖出已成,已撤或部撤,但未卖满,距离上次卖成时间大于5秒
			if (keyItem.BStatus == trade.O_STATUS_DONE || keyItem.BStatus == trade.O_STATUS_PART_CANCEL) && keyItem.SStatus == "" && nowInt64-keyItem.BDStamp > 5 {
				needCheckKeys = append(needCheckKeys, key.(string))
			}
			return true
		})
		timeStr := lib.GetTime()

		if len(needCheckKeys) == 0 || !inTradeTime(timeStr) { //时间排除
			time.Sleep(time.Second)
			continue
		}
		holdInfo := td.CheckHold()
		if holdInfo.Code != "0" {
			log.Println("查询持仓异常:", holdInfo.Message)
			time.Sleep(time.Duration(*holdInterval) * time.Second)
			continue
		}
		codeHoldMap := map[string]trade.HoldItem{}
		for _, i := range holdInfo.HoldList {
			codeHoldMap[i.Code] = i
		}

		orderInfo := td.CheckOrder()
		if orderInfo.Code != "0" {
			log.Println("查询交易异常:", orderInfo.Message)
			time.Sleep(time.Duration(*holdInterval) * time.Second)
			continue
		}

		//查询code最后一笔卖成单
		codeLastSaleMap := map[string]trade.OrderItem{}
		for _, item := range orderInfo.List {
			//只看已成的卖单
			if item.Type != trade.O_SALE_TYPE || item.Status != trade.O_STATUS_DONE {
				continue
			}
			if last, ok := codeLastSaleMap[item.Code]; ok {
				if item.Time > last.Time {
					codeLastSaleMap[item.Code] = item
				}
			} else {
				codeLastSaleMap[item.Code] = item
			}
		}

		for _, key := range needCheckKeys {
			vT, _ := keyItemMap.Load(key)
			keyItem := vT.(*keyStatusItem)
			i := codeHoldMap[keyItem.Code]
			if i.HoldVol == 0 { //已清仓,修改相关字段
				keyItem.FsOid = "hand" //手动清仓标记
				keyItem.SOid = "hand"
				keyItem.SOStamp = nowInt64
				keyItem.SDStamp = nowInt64
				keyItem.SDTime = lib.GetTime()
				keyItem.SStatus = trade.O_STATUS_DONE
				keyItem.SDVolume = keyItem.BDVolume

				//用买入价占位,如无异常,当前清仓,则一定存在今日最后一次卖单
				keyItem.SOPrice = keyItem.BDPrice
				keyItem.SDPrice = keyItem.BDPrice

				//按最新一次卖成单成交情况填入
				if s, ok := codeLastSaleMap[keyItem.Code]; ok {
					keyItem.SOPrice = s.OrderPrice
					keyItem.SDPrice = s.DealPrice
				}

				//按最后一次卖成价计算收益
				keyItem.Earn = math.Round((keyItem.SDPrice-keyItem.BDPrice)*100) / 100 * keyItem.BDVolume

				//回写
				keyItemMap.Store(key, keyItem)
				log.Printf("检测到清仓:%s,%s,%s", keyItem.Name, keyItem.Code, keyItem.Key)
				//无回调
			}
		}

		time.Sleep(time.Duration(*holdInterval) * time.Second)
	}
}

func inTradeTime(timeStr string) bool {
	return (timeStr >= "09:30:00" && timeStr <= "11:30:00") || (timeStr >= "13:00:00" && timeStr <= "15:00:00")
}
