#!/bin/bash

cd src/hqCenter/
echo "开始编译hqCenter"
echo "正在编译windows平台..."
GOOS=windows GOARCH=amd64 go build -o ../../bin/hqCenter.exe
echo "正在编译mac平台..."
GOOS=darwin GOARCH=amd64 go build -o ../../bin/hqCenter.mac
echo "正在编译linux平台..."
GOOS=linux GOARCH=amd64 go build -o ../../bin/hqCenter.linux
echo "编译hqCenter完成"
cd -

cd src/bondTrigger/
echo "开始编译bondTrigger"
echo "正在编译windows平台..."
GOOS=windows GOARCH=amd64 go build -o ../../bin/bondTrigger.exe
echo "正在编译mac平台..."
GOOS=darwin GOARCH=amd64 go build -o ../../bin/bondTrigger.mac
echo "正在编译linux平台..."
GOOS=linux GOARCH=amd64 go build -o ../../bin/bondTrigger.linux
echo "编译bondTrigger完成"
cd -

cd src/orderHolder/
echo "开始编译orderHolder"
echo "正在编译windows平台..."
GOOS=windows GOARCH=amd64 go build -o ../../bin/orderHolder.exe
echo "正在编译mac平台..."
GOOS=darwin GOARCH=amd64 go build -o ../../bin/orderHolder.mac
echo "正在编译linux平台..."
GOOS=linux GOARCH=amd64 go build -o ../../bin/orderHolder.linux
echo "编译orderHolder完成"
cd -
