//go:generate protoc --proto_path=$GOPATH/src:. --twirp_out=. --go_out=. api.proto

package api
