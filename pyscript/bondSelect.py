#!python3
# -*- coding:utf-8 -*-
import argparse
import json

from jvUtil import SQL

def selectBond(token, server="", writeFile=""):
    sql = SQL.Construct(token, server)
    r = sql.bond()
    selectDict = {}
    if 'data' in r and 'list' in r['data'] and "fields" in r['data']:
        field = r["data"]['fields']
        list = r["data"]['list']
        for i in list:
            if len(i) == len(field):
                map = dict(zip(field, i))
                # 按转债信息筛选
                if float(map["转股溢价(昨收)"]) < 200:
                    selectDict[map["转债代码"]] = map["正股代码"]
    # 写入选债文件,供行情调用
    if writeFile:
        with open(writeFile, 'w') as file:
            file.write(json.dumps(selectDict))
            print("已写入到文件:", writeFile)


# 运行入口
if __name__ == '__main__':
    # 获取命令行参数
    parser = argparse.ArgumentParser(description='筛选转债至结果集,供行情调用')
    parser.add_argument('--token', type=str, default="")
    parser.add_argument('--server', type=str, default="")
    parser.add_argument('--outFile', type=str, default="")
    args = parser.parse_args()
    token = args.token
    server = args.server
    outFile = args.outFile

    # 筛选
    selectBond(token, server, outFile)
