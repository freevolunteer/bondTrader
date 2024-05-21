#windows平台多个命令请分别运行,日志收请自行处理

#启动行情服务器
./bin/hqCenter.exe --token=jvquantToken

#启动交易维护器
./bin/orderHolder.exe --token=jvquantToken --acc=资金账户 --pwd=资金密码

sleep 5
#启动策略,行情服务器需先启动
./bin/bondTrigger.exe