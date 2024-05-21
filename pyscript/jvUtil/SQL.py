#!python3
# -*- coding:utf-8 -*-
import time
import requests


# 查询相关封装
class Construct:
    __acc = ""
    __token = ""
    __server_req_url = "http://jvquant.com/query/server?market=ab&type=sql&token="
    __sql_ser_addr = ""

    def __log(self, *args):
        print(time.strftime('%H:%M:%S', time.localtime(time.time())), args)

    def __init__(self, token, server="", logHandle=""):
        self.__token = token
        # 指定日志处理方法
        if logHandle != "":
            self.__log = logHandle
        # 如未指定server,则发起server查询
        if server != "":
            self.__server = server
        else:
            self.__getSerAddr()

    def __getSerAddr(self):
        url = self.__server_req_url + self.__token
        try:
            res = requests.get(url=url)
        except Exception as e:
            self.__log(e)
            return
        if (res.json()["code"] == "0"):
            self.__server = res.json()["server"]
            print("获取SQL服务器地址成功:" + self.__server)
        else:
            msg = "获取SQL服务器地址失败:" + res.text
            self.__log(msg)
            exit(-1)

    def qeury(self, query, page=1):
        url = "%s/sql?&mode=sql&acc=%s&token=%s&query=%s&page=%s" % (
            self.__server, self.__acc, self.__token, query, page)
        try:
            res = requests.get(url=url)
        except Exception as e:
            self.__log(e)
            return
        return res.json()

    def kline(self, code, cate="stock", type="day", fq="前复权", limit="240"):
        url = "%s/sql?&mode=kline&acc=%s&token=%s&cate=%s&code=%s&type=%s&fq=%s&limit=%s" % (
            self.__server, self.__acc, self.__token, cate, code, type, fq, limit)
        try:
            res = requests.get(url=url)
        except Exception as e:
            self.__log(e)
            return
        return res.json()

    def bond(self):
        url = "%s/sql?&mode=bond&acc=%s&token=%s" % (
            self.__server, self.__acc, self.__token)
        try:
            res = requests.get(url=url)
        except Exception as e:
            self.__log(e)
            return
        return res.json()

    def industry(self):
        url = "%s/sql?&mode=industry&acc=%s&token=%s" % (
            self.__server, self.__acc, self.__token)
        try:
            res = requests.get(url=url)
        except Exception as e:
            self.__log(e)
            return
        return res.json()
