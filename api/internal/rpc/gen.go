//go:generate protoc --proto_path=$GOPATH/src:. --twirp_out=. --go_out=. rpc.proto

package rpc