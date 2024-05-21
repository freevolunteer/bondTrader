#!python3
# -*- coding:utf-8 -*-
import argparse
from jvUtil import Trade


def getTicket(token, acc, pwd, ticket="", server="", writeFile=""):
    trade = Trade.Construct(token=token, server=server, ticket=ticket, acc=acc, pwd=pwd)
    ticket = trade.getTicket()
    print("获取到交易凭证:", ticket)
    check_hold = trade.check_hold()
    print("检查账户可用资金:", check_hold['usable'])

    # 写入文件,供交易调用
    if writeFile:
        with open(writeFile, 'w') as file:
            file.write(ticket)
            print("已写入到文件:",writeFile)


# 运行入口
if __name__ == '__main__':
    # 获取命令行参数
    parser = argparse.ArgumentParser(description='获取流通股信息至结果集,供行情调用')
    parser.add_argument('--token', type=str, default="")
    parser.add_argument('--ticket', type=str, default="")
    parser.add_argument('--acc', type=str, default="")
    parser.add_argument('--pwd', type=str, default="")
    parser.add_argument('--server', type=str, default="")
    parser.add_argument('--outFile', type=str, default="")
    args = parser.parse_args()
    token = args.token
    ticket = args.ticket
    acc = args.acc
    pwd = args.pwd
    server = args.server
    outFile = args.outFile

    # 筛选
    getTicket(token, acc, pwd, ticket, server, outFile)
