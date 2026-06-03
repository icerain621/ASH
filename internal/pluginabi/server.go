package pluginabi

import (
	"errors"
	"net"
	"sync"

	"google.golang.org/grpc"

	"github.com/ash-repwiki/ash/internal/store"
	ashv1 "github.com/ash-repwiki/ash/proto/ash/v1"
)

type RegistryRuntime struct {
	Addr     string
	Listener net.Listener
	Server   *grpc.Server

	done chan error
	once sync.Once
}

func StartRegistryServer(addr string, db *store.DB, opts ...grpc.ServerOption) (*RegistryRuntime, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	srv := grpc.NewServer(opts...)
	ashv1.RegisterPluginRegistryServiceServer(srv, NewRegistryServer(db))
	rt := &RegistryRuntime{
		Addr:     lis.Addr().String(),
		Listener: lis,
		Server:   srv,
		done:     make(chan error, 1),
	}
	go func() {
		err := srv.Serve(lis)
		if errors.Is(err, grpc.ErrServerStopped) {
			err = nil
		}
		rt.done <- err
		close(rt.done)
	}()
	return rt, nil
}

func (r *RegistryRuntime) Done() <-chan error {
	if r == nil {
		ch := make(chan error)
		close(ch)
		return ch
	}
	return r.done
}

func (r *RegistryRuntime) Stop() {
	if r == nil {
		return
	}
	r.once.Do(func() {
		r.Server.Stop()
		_ = r.Listener.Close()
	})
}
