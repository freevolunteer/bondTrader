#!python3
# -*- coding:utf-8 -*-
import websocket
import time
import datetime
import zlib
import threading
import logging
from jvUtil import *

# from jvtrader import *
# 1.获取正股和可转债映射
# 2.初始化交易账户
# 3.监听正股/转债行情
# 4.判断交易买点/卖点
# 5.维护当前持仓信息


# 全局常量配置
JqToken = ""  #token
TradeAcc = "" #资金账户
TradePwd = ""
exitTime = "11:30:00"


def main():
    # trader = Trade.Construct(JqToken,TradeAcc,TradePwd)
    # print(trader.check_hold())
    # print(trader.check_order())
    # print(trader.buy("600519","贵州茅台","1700","100"))
    # print(trader.sale("600519","贵州茅台","1700","100"))
    # print(trader.cancel("1233"))

    # hq = HanqQing.Construct(logHandle, JqToken, onRevLv1, onRevLv2)
    # hq.addLv1(["600519"])
    # time.sleep(2)
    # hq.addLv2(["600519"])
    # threading.Thread(target=ctl, args=(hq,)).start()
    # time.sleep(2)
    # hq.close()

    # sql=SQL.Construct(logHandle=logHandle,token=JqToken)
    # print(sql.bond())
    # print(sql.industry())
    # print(sql.qeury("主板"))
    # print(sql.kline("600519"))
    print('end')


def ctl(hq):
    while (1):
        now = datetime.datetime.now()
        timeStr = now.strftime('%Y-%m-%d')
        if (timeStr > exitTime):
            hq.close()
            return
        time.sleep(1)


def onRevLv1(code, hqMap):
    print(code, hqMap)


def onRevLv2(code, hqMap):
    print(code, hqMap)


def logHandle(*args):
    print(time.strftime('%H:%M:%S', time.localtime(time.time())), args)


# 运行入口
if __name__ == '__main__':
    try:
        main()
    except Exception as e:
        logHandle(e)
