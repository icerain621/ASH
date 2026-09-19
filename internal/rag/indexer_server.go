package rag

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"

	"google.golang.org/grpc"

	ashv1 "github.com/ash-repwiki/ash/proto/ash/v1"
)

// IndexerServer exposes RagIndexerService by delegating to the in-process RAG service (DX76).
type IndexerServer struct {
	ashv1.UnimplementedRagIndexerServiceServer
	rag *Service
}

func NewIndexerServer(rag *Service) *IndexerServer {
	return &IndexerServer{rag: rag}
}

func (s *IndexerServer) IndexRepo(ctx context.Context, req *ashv1.IndexRepoRequest) (*ashv1.IndexRepoResponse, error) {
	if req == nil {
		return &ashv1.IndexRepoResponse{Status: status("INVALID_REQUEST", "request is required")}, nil
	}
	out, err := s.rag.WithContext(ctx).Index(IndexRequest{
		RepoRoot: req.GetRepoRoot(),
		SpaceID:  spaceFromTrace(req.GetContext()),
		Embed:    req.GetEmbed(),
	})
	if err != nil {
		return &ashv1.IndexRepoResponse{Status: status("RAG_INDEX_FAILED", err.Error())}, nil
	}
	return &ashv1.IndexRepoResponse{
		Documents: int32(out.Documents),
		Chunks:    int32(out.Chunks),
		Embedded:  int32(out.Embedded),
		Status:    status("OK", "indexed"),
	}, nil
}

func (s *IndexerServer) Query(ctx context.Context, req *ashv1.QueryRequest) (*ashv1.QueryResponse, error) {
	if req == nil {
		return &ashv1.QueryResponse{Status: status("INVALID_REQUEST", "request is required")}, nil
	}
	if strings.TrimSpace(req.GetText()) == "" {
		return &ashv1.QueryResponse{Status: status("INVALID_REQUEST", "text is required")}, nil
	}
	out, err := s.rag.WithContext(ctx).Query(QueryRequest{
		RepoRoot:   req.GetRepoRoot(),
		Text:       req.GetText(),
		TopK:       int(req.GetTopK()),
		SpaceID:    spaceFromTrace(req.GetContext()),
		Prefer:     req.GetPrefer(),
		ExpandRefs: req.GetExpandRefs(),
	})
	if err != nil {
		return &ashv1.QueryResponse{Status: status("RAG_QUERY_FAILED", err.Error())}, nil
	}
	items := make([]*ashv1.QueryHit, 0, len(out.Items))
	for _, h := range out.Items {
		items = append(items, &ashv1.QueryHit{
			Ref: h.Ref, Path: h.Path, Symbol: h.Symbol,
			StartLine: int32(h.StartLine), EndLine: int32(h.EndLine),
			Digest: h.Digest, Score: h.Score, Snippet: h.Snippet,
		})
	}
	return &ashv1.QueryResponse{
		Items: items, RetrievalMode: out.RetrievalMode,
		FtsAvailable: out.FtsAvailable, VectorAvailable: out.VectorAvailable,
		VectorFallback: out.VectorFallback, PreferApplied: out.PreferApplied,
		Status: status("OK", "query"),
	}, nil
}

// IndexerRuntime is a listening RagIndexerService (DX76).
type IndexerRuntime struct {
	Addr     string
	Listener net.Listener
	Server   *grpc.Server

	done chan error
	once sync.Once
}

// StartIndexerServer listens on addr. Empty addr returns (nil, nil) — default off.
func StartIndexerServer(addr string, rag *Service, opts ...grpc.ServerOption) (*IndexerRuntime, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, nil
	}
	if rag == nil {
		return nil, errors.New("rag service is required")
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	srv := grpc.NewServer(opts...)
	ashv1.RegisterRagIndexerServiceServer(srv, NewIndexerServer(rag))
	rt := &IndexerRuntime{
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

func (r *IndexerRuntime) Done() <-chan error {
	if r == nil {
		ch := make(chan error)
		close(ch)
		return ch
	}
	return r.done
}

func (r *IndexerRuntime) Stop() {
	if r == nil {
		return
	}
	r.once.Do(func() {
		r.Server.Stop()
		_ = r.Listener.Close()
	})
}

func spaceFromTrace(ctx *ashv1.TraceContext) string {
	if ctx == nil || strings.TrimSpace(ctx.GetSpaceId()) == "" {
		return "local"
	}
	return strings.TrimSpace(ctx.GetSpaceId())
}

func status(code, message string) *ashv1.Status {
	return &ashv1.Status{Code: code, Message: message}
}
