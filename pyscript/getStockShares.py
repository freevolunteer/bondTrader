#!python3
# -*- coding:utf-8 -*-
import argparse
import json

from jvUtil import SQL

def getShare(token, server="", writeFile=""):
    sql = SQL.Construct(token, server)
    r = sql.industry()
    shareDict = {}
    if 'data' in r and 'list' in r['data'] and "fields" in r['data']:
        field = r["data"]['fields']
        list = r["data"]['list']
        for i in list:
            if len(i) == len(field):
                map = dict(zip(field, i))
                if(map["流通股数"]!="-"):
                    shareDict[map["股票代码"]] = int(map["流通股数"])
    # 写入文件,供行情调用
    if writeFile:
        with open(writeFile, 'w') as file:
            file.write(json.dumps(shareDict))
            print("已写入到文件:", writeFile)


# 运行入口
if __name__ == '__main__':
    # 获取命令行参数
    parser = argparse.ArgumentParser(description='获取流通股信息至结果集,供行情调用')
    parser.add_argument('--token', type=str, default="")
    parser.add_argument('--server', type=str, default="")
    parser.add_argument('--outFile', type=str, default="")
    args = parser.parse_args()
    token = args.token
    server = args.server
    outFile = args.outFile

    # 筛选
    getShare(token, server, outFile)
