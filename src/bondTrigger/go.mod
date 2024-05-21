module bondTrigger

require (
	lib v0.0.0
	trade v0.0.0
)

replace (
	lib => ../lib
	trade => ../jvUtil/trade
)
go 1.18
