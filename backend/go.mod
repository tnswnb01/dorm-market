module dormmarket

go 1.22

require (
	github.com/lib/pq v1.12.3
	golang.org/x/crypto v0.31.0
)

require github.com/gorilla/websocket v1.5.3

replace golang.org/x/crypto => github.com/golang/crypto v0.31.0
