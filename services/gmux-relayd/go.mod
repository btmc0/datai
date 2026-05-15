module github.com/gmuxapp/gmux/services/gmux-relayd

go 1.26.1

require (
	github.com/gmuxapp/gmux/packages/relayproto v0.0.0
	nhooyr.io/websocket v1.8.17
)

replace github.com/gmuxapp/gmux/packages/relayproto => ../../packages/relayproto
