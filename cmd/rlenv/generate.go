//go:generate protoc --go_out=paths=source_relative:envpb --go-grpc_out=paths=source_relative:envpb --proto_path=../../rl/proto battlesnake_env.proto

package main
