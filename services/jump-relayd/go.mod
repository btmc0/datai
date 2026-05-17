module github.com/sting8k/jump/services/jump-relayd

go 1.26.1

require (
	github.com/sting8k/jump/packages/relayproto v0.0.0
	nhooyr.io/websocket v1.8.17
)

replace github.com/sting8k/jump/packages/relayproto => ../../packages/relayproto
