package interaction

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/store"
)

// ErrReplayDigestMismatch is returned when a sealed thread's live fold digest
// does not match the archived digest (REPLAY_DIGEST_MISMATCH).
var ErrReplayDigestMismatch = errors.New("REPLAY_DIGEST_MISMATCH")

// ReplayResult is a read-only re-fold plus digest check.
type ReplayResult struct {
	ThreadID       string         `json:"threadId"`
	OK             bool           `json:"ok"`
	Digest         string         `json:"digest"`
	SealedDigest   string         `json:"sealedDigest,omitempty"`
	MismatchSeq    int64          `json:"mismatchSeq,omitempty"`
	Nodes          []TimelineNode `json:"nodes"`
	Links          []MemoryLink   `json:"links"`
	HeadSeq        int64          `json:"headSeq"`
	Status         string         `json:"status"`
}

// CompareResult diffs two thread folds (nodes + memory links).
type CompareResult struct {
	LeftThreadID  string   `json:"leftThreadId"`
	RightThreadID string   `json:"rightThreadId"`
	LeftDigest    string   `json:"leftDigest"`
	RightDigest   string   `json:"rightDigest"`
	NodesAdded    []string `json:"nodesAdded"`
	NodesRemoved  []string `json:"nodesRemoved"`
	NodesChanged  []string `json:"nodesChanged"`
	LinksAdded    []string `json:"linksAdded"`
	LinksRemoved  []string `json:"linksRemoved"`
}

// Seal freezes the current fold digest onto the thread and marks status=sealed.
func (s *Service) Seal(threadID string) (*Thread, error) {
	th, err := s.GetThread(threadID)
	if err != nil {
		return nil, err
	}
	fold, err := s.FoldThread(threadID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"status":     ThreadStatusSealed,
		"digest":     fold.Digest,
		"head_seq":   fold.HeadSeq,
		"updated_at": now,
	}
	if err := s.q().Model(&store.InteractionThread{}).Where("id = ?", th.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	th.Status = ThreadStatusSealed
	th.Digest = fold.Digest
	th.HeadSeq = fold.HeadSeq
	th.UpdatedAt = now.Unix()
	s.emitInteraction(th.RunID, "interaction.thread_sealed", map[string]any{
		"sessionId": th.SessionID,
		"threadId":  th.ID,
		"digest":    fold.Digest,
		"headSeq":   fold.HeadSeq,
		"spaceId":   th.SpaceID,
		"linkCount": len(fold.Links),
	})
	return th, nil
}

// Replay re-folds the thread and checks against the sealed digest when present.
func (s *Service) Replay(threadID string) (*ReplayResult, error) {
	th, err := s.GetThread(threadID)
	if err != nil {
		return nil, err
	}
	fold, err := s.FoldThread(threadID)
	if err != nil {
		return nil, err
	}
	out := &ReplayResult{
		ThreadID: th.ID, Digest: fold.Digest, SealedDigest: th.Digest,
		Nodes: fold.Nodes, Links: fold.Links, HeadSeq: fold.HeadSeq, Status: th.Status,
		OK: true,
	}
	if th.Status == ThreadStatusSealed && strings.TrimSpace(th.Digest) != "" && th.Digest != fold.Digest {
		out.OK = false
		out.MismatchSeq = firstMismatchSeq(th.Digest, fold)
		s.emitInteraction(th.RunID, "interaction.replay_mismatch", map[string]any{
			"sessionId":    th.SessionID,
			"threadId":     th.ID,
			"sealedDigest": th.Digest,
			"liveDigest":   fold.Digest,
			"mismatchSeq":  out.MismatchSeq,
			"spaceId":      th.SpaceID,
		})
		return out, fmt.Errorf("%w: sealed=%s live=%s", ErrReplayDigestMismatch, th.Digest, fold.Digest)
	}
	return out, nil
}

func (s *Service) emitInteraction(runID, eventType string, payload map[string]any) {
	if s == nil || s.events == nil || strings.TrimSpace(runID) == "" {
		return
	}
	trace := "interaction"
	var rec store.RunRecord
	if err := s.q().First(&rec, "id = ?", runID).Error; err == nil && strings.TrimSpace(rec.TraceID) != "" {
		trace = rec.TraceID
	}
	_, _ = s.events.Append(runID, trace, eventType, "info", payload, events.WithVisibility(events.VisibilityUIOnly))
}

func firstMismatchSeq(_ string, fold *FoldResult) int64 {
	if fold == nil || len(fold.Nodes) == 0 {
		return 0
	}
	// Without sealed node archive, report head as first divergence hint.
	return fold.HeadSeq
}

// Compare diffs two threads' folds by event identity (seq+type+payloadDigest) and links.
func (s *Service) Compare(leftThreadID, rightThreadID string) (*CompareResult, error) {
	leftID := strings.TrimSpace(leftThreadID)
	rightID := strings.TrimSpace(rightThreadID)
	if leftID == "" || rightID == "" {
		return nil, fmt.Errorf("left and right threadId are required")
	}
	left, err := s.FoldThread(leftID)
	if err != nil {
		return nil, err
	}
	right, err := s.FoldThread(rightID)
	if err != nil {
		return nil, err
	}
	return compareFolds(left, right), nil
}

func compareFolds(left, right *FoldResult) *CompareResult {
	out := &CompareResult{
		LeftThreadID: left.ThreadID, RightThreadID: right.ThreadID,
		LeftDigest: left.Digest, RightDigest: right.Digest,
	}
	lNodes := indexNodes(left.Nodes)
	rNodes := indexNodes(right.Nodes)
	for key, ln := range lNodes {
		rn, ok := rNodes[key]
		if !ok {
			out.NodesRemoved = append(out.NodesRemoved, key)
			continue
		}
		if shortDigest(string(ln.Payload)) != shortDigest(string(rn.Payload)) || ln.Visibility != rn.Visibility {
			out.NodesChanged = append(out.NodesChanged, key)
		}
	}
	for key := range rNodes {
		if _, ok := lNodes[key]; !ok {
			out.NodesAdded = append(out.NodesAdded, key)
		}
	}
	lLinks := indexLinks(left.Links)
	rLinks := indexLinks(right.Links)
	for key := range lLinks {
		if _, ok := rLinks[key]; !ok {
			out.LinksRemoved = append(out.LinksRemoved, key)
		}
	}
	for key := range rLinks {
		if _, ok := lLinks[key]; !ok {
			out.LinksAdded = append(out.LinksAdded, key)
		}
	}
	sort.Strings(out.NodesAdded)
	sort.Strings(out.NodesRemoved)
	sort.Strings(out.NodesChanged)
	sort.Strings(out.LinksAdded)
	sort.Strings(out.LinksRemoved)
	return out
}

func indexNodes(nodes []TimelineNode) map[string]TimelineNode {
	out := make(map[string]TimelineNode, len(nodes))
	for _, n := range nodes {
		key := fmt.Sprintf("%d|%s", n.Seq, n.Type)
		out[key] = n
	}
	return out
}

func indexLinks(links []MemoryLink) map[string]MemoryLink {
	out := make(map[string]MemoryLink, len(links))
	for _, l := range links {
		key := fmt.Sprintf("%d|%s|%s", l.EventSeq, l.LinkType, l.MemoryID)
		out[key] = l
	}
	return out
}
