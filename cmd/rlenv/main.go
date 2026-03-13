package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/bodist/haruko/cmd/rlenv/envpb"
	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 50051, "gRPC server port")
	numEnvs := flag.Int("num-envs", 128, "default number of parallel environments")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	envServer := NewEnvServer(*numEnvs)
	pb.RegisterBattlesnakeEnvServer(srv, envServer)

	log.Printf("rlenv gRPC server starting on :%d (default envs=%d)", *port, *numEnvs)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// EnvServer implements the BattlesnakeEnv gRPC service.
type EnvServer struct {
	pb.UnimplementedBattlesnakeEnvServer
	mgr            *EnvManager
	defaultNumEnvs int
}

// NewEnvServer creates a new EnvServer with the given default environment count.
func NewEnvServer(defaultNumEnvs int) *EnvServer {
	return &EnvServer{
		defaultNumEnvs: defaultNumEnvs,
	}
}
