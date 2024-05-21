#!python3
# -*- coding:utf-8 -*-
import time
import requests


# 交易相关封装
class Construct:
    __token = ""
    __server_req_url = "http://jvquant.com/query/server?market=ab&type=trade&token="
    __server = ""
    __trade_acc = ""
    __trade_pwd = ""
    __ticket = ""

    def __log(self, *args):
        print(time.strftime('%H:%M:%S', time.localtime(time.time())), args)

    def __init__(self, token, server="", ticket="", acc="", pwd="", logHandle=""):
        self.__token = token
        if ticket == "" and (acc == "" or pwd == ""):
            msg = "交易初始化失败:请指定ticket或交易账户"
            print(msg)
            exit(-1)
        # 指定日志处理方法
        if logHandle != "":
            self.__log = logHandle
        # 如未指定server,则发起server查询
        if server != "":
            self.__server = server
        else:
            self.__getSerAddr()

        self.__trade_acc = acc
        self.__trade_pwd = pwd
        self.__ticket = ticket
        if ticket == "":
            self.login()

    def __getSerAddr(self):
        url = self.__server_req_url + self.__token
        try:
            res = requests.get(url=url).json()
        except Exception as e:
            self.__log(e)
            return
        if ("code" in res and res["code"] == "0" and "server" in res):
            self.__server = res["server"]
            print("获取交易服务器地址成功:" + self.__server)
        else:
            msg = "获取交易服务器地址失败:" + res.text
            self.__log(msg)
            exit(-1)

    def getTicket(self):
        return self.__ticket

    def login(self):
        url = "%s/login?&token=%s&acc=%s&pass=%s" % (
            self.__server, self.__token, self.__trade_acc, self.__trade_pwd)
        try:
            res = requests.get(url=url).json()
        except Exception as e:
            self.__log("登录请求失败:", url, e)
            exit(-1)

        if ("code" in res and res["code"] == "0" and "ticket" in res):
            self.__ticket = res["ticket"]
            print("获取交易凭证成功:" + self.__ticket)
        else:
            self.__log("获取交易凭证失败:", res)
            exit(-1)

    def check_order(self):
        url = "%s/check_order?&token=%s&ticket=%s" % (
            self.__server, self.__token, self.__ticket)
        try:
            res = requests.get(url=url).json()
        except Exception as e:
            self.__log(e)
            return
        return res

    def check_hold(self):
        url = "%s/check_hold?&token=%s&ticket=%s" % (
            self.__server, self.__token, self.__ticket)
        try:
            res = requests.get(url=url).json()
        except Exception as e:
            self.__log(e)
            return
        return res

    def buy(self, code, name, price, vol):
        url = "%s/buy?&token=%s&ticket=%s&code=%s&name=%s&price=%s&volume=%s" % (
            self.__server, self.__token, self.__ticket, code, name, price, vol)
        try:
            res = requests.get(url=url).json()
        except Exception as e:
            self.__log(e)
            return
        return res

    def sale(self, code, name, price, vol):
        url = "%s/sale?&token=%s&ticket=%s&code=%s&name=%s&price=%s&volume=%s" % (
            self.__server, self.__token, self.__ticket, code, name, price, vol)
        try:
            res = requests.get(url=url).json()
        except Exception as e:
            self.__log(e)
            return
        return res

    def cancel(self, order_id):
        url = "%s/cancel?&token=%s&ticket=%s&order_id=%s" % (
            self.__server, self.__token, self.__ticket, order_id)
        try:
            res = requests.get(url=url).json()
        except Exception as e:
            self.__log(e)
            return
        return res
