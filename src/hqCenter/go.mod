module hqCenter

require (
	hq v0.0.0
	lib v0.0.0
)

require (
	github.com/gorilla/websocket v1.5.1 // indirect
	golang.org/x/net v0.17.0 // indirect
)

replace lib => ../lib

replace hq => ../jvUtil/hangqing

go 1.18
