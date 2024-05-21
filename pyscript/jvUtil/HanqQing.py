#!python3
# -*- coding:utf-8 -*-
import time
import datetime
import websocket
import zlib
import requests
import threading

# 行情订阅推送封装
class Construct:
    __token = ""
    __server_req_url = "http://jvquant.com/query/server?market=ab&type=websocket&token="
    __ws_ser_addr = ""
    __ws_conn = ""
    __lv1_field = ["time", "name", "price", "ratio", "volume", "amount", "b1", "b1p", "b2", "b2p", "b3", "b3p", "b4",
                   "b4p", "b5", "b5p", "s1", "s1p", "s2", "s2p", "s3", "s3p", "s4", "s4p", "s5", "s5p"]
    __lv2_field = ["time", "oid", "price", "vol"]

    def __init__(self, logHandle, token, onRevLv1, onRevLv2):
        if logHandle == "" or token == "" or onRevLv1 == "" or onRevLv2 == "":
            msg = "行情初始化失败:logHandle或token或onRevLv1或onRevLv2必要参数缺失。"
            print(msg)
            exit(-1)
        self.__log = logHandle
        self.__token = token
        self.__deal_lv1 = onRevLv1
        self.__deal_lv2 = onRevLv2
        self.__getSerAddr()
        self.__conn_event = threading.Event()
        self.th_handle = threading.Thread(target=self.__conn)
        self.th_handle.start()
        self.__conn_event.wait()

    def __getSerAddr(self):
        url = self.__server_req_url + self.__token
        try:
            res = requests.get(url=url)
        except Exception as e:
            self.__log(e)
            return
        if (res.json()["code"] == "0"):
            self.__ws_ser_addr = res.json()["server"]
            print("获取行情服务器地址成功:" + self.__ws_ser_addr)
        else:
            msg = "获取行情服务器地址失败:" + res.text
            self.__log(msg)
            exit(-1)

    def __conn(self):
        wsUrl = self.__ws_ser_addr + "?token=" + self.__token
        self.__ws_conn = websocket.WebSocketApp(wsUrl,
                                                on_open=self.__on_open,
                                                on_data=self.__on_message,
                                                on_error=self.__on_error,
                                                on_close=self.__on_close)
        self.__ws_conn.run_forever()
        self.__conn_event.set()
        self.__log("ws thread exited")

    def addLv1(self, codeArr):
        cmd = "add="
        lv1Codes = []
        for code in codeArr:
            lv1Codes.append("lv1_" + code)

        cmd = cmd + ",".join(lv1Codes)
        self.__log("cmd:" + cmd)
        self.__ws_conn.send(cmd)

    def addLv2(self, codeArr):
        cmd = "add="
        lv1Codes = []
        for code in codeArr:
            lv1Codes.append("lv2_" + code)

        cmd = cmd + ",".join(lv1Codes)
        self.__log("cmd:" + cmd)
        self.__ws_conn.send(cmd)

    def dealLv1(self, data):
        self.__deal_lv1(data)

    def dealLv2(self, data):
        self.__deal_lv1(data)

    def __on_open(self, ws):
        self.__conn_event.set()
        self.__log("行情连接已创建")

    def __on_error(self, ws, error):
        self.__log("行情处理error:", error)

    def __on_close(self, ws, code, msg):
        self.__log("行情服务未连接")

    def close(self):
        self.__ws_conn.close()

    def __on_message(self, ws, message, type, flag):
        # 命令返回文本消息
        if type == websocket.ABNF.OPCODE_TEXT:
            self.__log("Text响应:" + message)
        # 行情推送压缩二进制消息，在此解压缩
        if type == websocket.ABNF.OPCODE_BINARY:
            now = datetime.datetime.now()
            nStamp = time.mktime(now.timetuple())
            date = now.strftime('%Y-%m-%d')

            rb = zlib.decompress(message, -zlib.MAX_WBITS)
            text = rb.decode("utf-8")
            # self.__log("Binary响应:" + text)
            ex1 = text.split("\n")
            for e1 in ex1:
                ex2 = e1.split("=")
                if len(ex2) != 2:
                    continue
                code = ex2[0]
                hqs = ex2[1]
                if code.startswith("lv1_"):
                    code = code.replace("lv1_", "")
                    hq = hqs.split(",")
                    if len(hq) == len(self.__lv1_field):
                        hqMap = dict(zip(self.__lv1_field, hq))
                        timeStr = hqMap['time']
                        date_obj = datetime.datetime.strptime(date + ' ' + timeStr, '%Y-%m-%d %H:%M:%S')
                        tStamp = int(time.mktime(date_obj.timetuple()))
                        if abs(tStamp - nStamp) <= 2:
                            self.__deal_lv1(code, hqMap)

                if code.startswith("lv2_"):
                    code = code.replace("lv2_", "")
                    hqArr = hqs.split("|")
                    for hq in hqArr:
                        hqEx = hq.split(",")
                        if len(hqEx) == len(self.__lv2_field):
                            hqMap = dict(zip(self.__lv2_field, hqEx))
                            timeEx = hqMap['time'].split('.')
                            if len(timeEx) == 2:
                                timeStr = timeEx[0]
                                date_obj = datetime.datetime.strptime(date + ' ' + timeStr, '%Y-%m-%d %H:%M:%S')
                                tStamp = int(time.mktime(date_obj.timetuple()))
                                if abs(tStamp - nStamp) <= 2:
                                    self.__deal_lv2(code, hqMap)
