module github.com/sting8k/jump/packages/adapter

go 1.26.1

require (
	github.com/sting8k/jump/packages/paths v0.0.0
	golang.org/x/sys v0.42.0
	nhooyr.io/websocket v1.8.17
)

replace github.com/sting8k/jump/packages/paths => ../paths
