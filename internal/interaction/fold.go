package interaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ash-repwiki/ash/internal/events"
)

// FoldEvents is a pure projection: same inputs ⇒ same nodes/links/digest.
func FoldEvents(sessionID, threadID, runID, spaceID string, evs []events.Envelope) FoldResult {
	nodes := make([]TimelineNode, 0, len(evs))
	links := make([]MemoryLink, 0)
	var headSeq int64
	for _, ev := range evs {
		vis := events.NormalizeVisibility(ev.Type, ev.Visibility)
		if vis == events.VisibilityAudit {
			continue
		}
		p := payloadMap(ev.Payload)
		node := TimelineNode{
			ID: ev.ID, Seq: ev.Seq, TS: ev.TS, Type: ev.Type,
			Visibility: vis, Severity: ev.Severity,
			StepID: strField(p, "stepId"), Payload: ev.Payload,
		}
		nodes = append(nodes, node)
		if ev.Seq > headSeq {
			headSeq = ev.Seq
		}
		links = append(links, extractLinks(sessionID, threadID, runID, spaceID, ev, p)...)
	}
	sort.SliceStable(links, func(i, j int) bool {
		if links[i].EventSeq != links[j].EventSeq {
			return links[i].EventSeq < links[j].EventSeq
		}
		if links[i].LinkType != links[j].LinkType {
			return links[i].LinkType < links[j].LinkType
		}
		return links[i].MemoryID < links[j].MemoryID
	})
	digest := computeDigest(nodes, links)
	return FoldResult{
		SessionID: sessionID, ThreadID: threadID, RunID: runID, SpaceID: spaceID,
		Nodes: nodes, Links: links, Digest: digest, HeadSeq: headSeq,
	}
}

func extractLinks(sessionID, threadID, runID, spaceID string, ev events.Envelope, p map[string]any) []MemoryLink {
	var out []MemoryLink
	switch ev.Type {
	case "memory.hit_used":
		for _, id := range stringSlice(p["recordIds"]) {
			out = append(out, makeLink(sessionID, threadID, runID, spaceID, ev, id, LinkHitUsed, ""))
		}
	case "memory.injected":
		for _, id := range stringSlice(p["recordIds"]) {
			out = append(out, makeLink(sessionID, threadID, runID, spaceID, ev, id, LinkContextRef, ""))
		}
	case "memory.candidate":
		if id := strField(p, "candidateId"); id != "" {
			out = append(out, makeLink(sessionID, threadID, runID, spaceID, ev, id, LinkCandidateOut, ""))
		}
		if id := strField(p, "recordId"); id != "" {
			out = append(out, makeLink(sessionID, threadID, runID, spaceID, ev, id, LinkCandidateOut, ""))
		}
	case "knowledge.injected", "skills.injected":
		for _, ref := range stringSlice(p["refs"]) {
			if memID, ok := parseMemoryRef(ref); ok {
				out = append(out, makeLink(sessionID, threadID, runID, spaceID, ev, memID, LinkContextRef, ""))
			}
		}
	}
	return out
}

func makeLink(sessionID, threadID, runID, spaceID string, ev events.Envelope, memoryID, linkType, layer string) MemoryLink {
	dig := shortDigest(fmt.Sprintf("%s|%d|%s|%s", linkType, ev.Seq, memoryID, ev.ID))
	return MemoryLink{
		ID: "ml_" + dig[:16],
		SpaceID: spaceID, SessionID: sessionID, ThreadID: threadID, RunID: runID,
		EventSeq: ev.Seq, EventID: ev.ID, MemoryID: memoryID, LinkType: linkType,
		Layer: layer, TS: ev.TS, Digest: dig,
	}
}

func parseMemoryRef(ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, "memory:") {
		id := strings.TrimSpace(strings.TrimPrefix(ref, "memory:"))
		return id, id != ""
	}
	return "", false
}

func computeDigest(nodes []TimelineNode, links []MemoryLink) string {
	type nodeDig struct {
		Seq  int64  `json:"seq"`
		Type string `json:"type"`
		Vis  string `json:"visibility"`
		PD   string `json:"payloadDigest"`
	}
	type linkDig struct {
		Seq      int64  `json:"eventSeq"`
		Type     string `json:"linkType"`
		MemoryID string `json:"memoryId"`
	}
	nd := make([]nodeDig, 0, len(nodes))
	for _, n := range nodes {
		nd = append(nd, nodeDig{Seq: n.Seq, Type: n.Type, Vis: n.Visibility, PD: shortDigest(string(n.Payload))})
	}
	ld := make([]linkDig, 0, len(links))
	for _, l := range links {
		ld = append(ld, linkDig{Seq: l.EventSeq, Type: l.LinkType, MemoryID: l.MemoryID})
	}
	raw, _ := json.Marshal(struct {
		Nodes []nodeDig `json:"nodes"`
		Links []linkDig `json:"links"`
	}{Nodes: nd, Links: ld})
	sum := sha256.Sum256(raw)
	return "thd_" + hex.EncodeToString(sum[:16])
}

func shortDigest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func payloadMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]any{}
	}
	return m
}

func strField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	default:
		return nil
	}
}
