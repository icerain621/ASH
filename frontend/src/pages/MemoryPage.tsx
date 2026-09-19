import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Check, Clock, DatabaseZap, RefreshCw, Search, Send, X } from "lucide-react";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { KnowledgePanel } from "@/modules/knowledge/components/KnowledgePanel";
import { MemoryLinkPanel } from "@/modules/interactions/components/MemoryLinkPanel";
import {
  createCandidate,
  getMemoryRecord,
  getMemoryTTLQueue,
  listCandidates,
  queryMemory,
  reviewCandidate,
  sweepMemoryTTL,
  type GovernanceHints,
  type MemoryRecord,
} from "@/modules/memory/api/memory.api";
import { getCurrentSpaceId } from "@/services/http/client";
import { fmtTime, shortId } from "@/shared/utils/format";

type MemoryPillarTab = "memory" | "knowledge" | "links";
type MemoryPerspective = "layer" | "scenario" | "skill" | "tools" | "project";

const PILLAR_TABS: { id: MemoryPillarTab; label: string }[] = [
  { id: "memory", label: "记忆体" },
  { id: "links", label: "关联" },
  { id: "knowledge", label: "知识" },
];

const PERSPECTIVES: { id: MemoryPerspective; label: string }[] = [
  { id: "layer", label: "分层" },
  { id: "scenario", label: "场景" },
  { id: "skill", label: "Skill" },
  { id: "tools", label: "Tools" },
  { id: "project", label: "项目" },
];

const LAYER_FILTERS = ["全部", "L0", "L1", "L2"] as const;

const PERSPECTIVE_HINTS: Partial<Record<MemoryPerspective, string>> = {
  scenario: "这些候选还没有 scenario: 标签；带 Run 新建时会从场景名回填。",
  skill: "这些候选还没有 skill: 标签。",
  tools: "这些候选还没有 tool: 标签。",
  project: "这些候选还没有 scopeRepo。",
};

function tagValue(tags: string[] | undefined, prefix: string) {
  const hit = (tags ?? []).find((tag) => tag.startsWith(prefix));
  return hit ? hit.slice(prefix.length) : "";
}

function perspectiveKey(item: MemoryRecord, perspective: MemoryPerspective) {
  if (perspective === "scenario") return tagValue(item.tags, "scenario:") || "未标注";
  if (perspective === "skill") return tagValue(item.tags, "skill:") || "未标注";
  if (perspective === "tools") return tagValue(item.tags, "tool:") || "未标注";
  if (perspective === "project") return item.scopeRepo || "未标注";
  return "";
}

function memoryStatusLabel(status: string) {
  const labels: Record<string, string> = {
    approved: "已通过",
    candidate: "待审核",
    deprecated: "已弃用",
    rejected: "已拒绝",
  };
  return labels[status] || status;
}

function edgeKindLabel(kind: string) {
  const labels: Record<string, string> = {
    conflict: "冲突",
    duplicate: "重复",
    pending_duplicate: "待审重复",
    replaces: "替代",
    supports: "支持",
  };
  return labels[kind] || kind;
}

function formatGovernance(hints?: GovernanceHints) {
  if (!hints) return "";
  const parts: string[] = [];
  for (const item of hints.duplicates ?? []) {
    parts.push(`重复: ${item.title || shortId(item.memoryId)} (${edgeKindLabel(item.kind)})`);
  }
  for (const item of hints.conflicts ?? []) {
    parts.push(`冲突: ${item.title || shortId(item.memoryId)}`);
  }
  return parts.join("；");
}

function sortByLayer(items: MemoryRecord[]) {
  const order = (layer: string) => {
    if (layer === "L0") return 0;
    if (layer === "L1") return 1;
    if (layer === "L2") return 2;
    return 9;
  };
  return [...items].sort((a, b) => order(a.layer) - order(b.layer) || a.title.localeCompare(b.title));
}

export function MemoryPage() {
  const qc = useQueryClient();
  const activeSpaceId = getCurrentSpaceId();
  const [pillarTab, setPillarTab] = useState<MemoryPillarTab>("memory");
  const [perspective, setPerspective] = useState<MemoryPerspective>("layer");
  const [layerFilter, setLayerFilter] = useState<(typeof LAYER_FILTERS)[number]>("全部");
  const [runId, setRunId] = useState("");
  const [linkRunId, setLinkRunId] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [governanceMsg, setGovernanceMsg] = useState("");
  const [ttlMsg, setTtlMsg] = useState("");
  const [queryText, setQueryText] = useState("doctor release");

  useEffect(() => {
    const q = new URLSearchParams(window.location.search);
    if (q.get("tab") === "links") setPillarTab("links");
    const rid = q.get("runId")?.trim();
    if (rid) {
      setLinkRunId(rid);
      setRunId(rid);
    }
  }, []);

  const candidatesQuery = useQuery({
    queryKey: ["memory", "candidates", activeSpaceId],
    queryFn: () => listCandidates(50),
  });

  const ttlQueueQuery = useQuery({
    queryKey: ["memory", "ttl-queue", activeSpaceId],
    queryFn: () => getMemoryTTLQueue(50),
  });

  const detailQuery = useQuery({
    queryKey: ["memory", "record", selectedId],
    queryFn: () => getMemoryRecord(selectedId!),
    enabled: !!selectedId,
  });

  const queryMut = useMutation({
    mutationFn: () => queryMemory({ text: queryText, topK: 10, scope: { repo: "ash" } }),
  });

  const createMut = useMutation({
    mutationFn: createCandidate,
    onSuccess: (res) => {
      setGovernanceMsg(formatGovernance(res.governance));
      qc.invalidateQueries({ queryKey: ["memory", "candidates"] });
    },
  });

  const reviewMut = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: "approve" | "reject" }) =>
      reviewCandidate(id, {
        decision,
        reason: "reviewed from UI",
        policyProfile: "default",
        reviewerId: "ui",
        ...(runId ? { runId } : {}),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["memory", "candidates"] });
      qc.invalidateQueries({ queryKey: ["memory", "ttl-queue"] });
      if (selectedId) qc.invalidateQueries({ queryKey: ["memory", "record", selectedId] });
    },
  });

  const ttlSweepMut = useMutation({
    mutationFn: () => sweepMemoryTTL({ dryRun: false }),
    onSuccess: (res) => {
      setTtlMsg(
        res.deprecated > 0
          ? `已弃用 ${res.deprecated} 条到期记忆；复核队列 ${res.reviewDue} 条。`
          : `无到期记录需弃用；复核队列 ${res.reviewDue} 条。`,
      );
      qc.invalidateQueries({ queryKey: ["memory", "ttl-queue"] });
      qc.invalidateQueries({ queryKey: ["memory", "candidates"] });
      if (selectedId) qc.invalidateQueries({ queryKey: ["memory", "record", selectedId] });
    },
    onError: (err: Error) => setTtlMsg(err.message),
  });

  const onSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const layer = String(fd.get("layer") || "L1");
    const body: Record<string, unknown> = {
      layer,
      title: fd.get("title"),
      body: fd.get("body"),
      scopeRepo: "ash",
    };
    const formRunId = String(fd.get("runId") || "");
    if (formRunId) body.runId = formRunId;
    const tags: string[] = [];
    const scenario = String(fd.get("scenario") || "").trim();
    const skill = String(fd.get("skill") || "").trim();
    const tool = String(fd.get("tool") || "").trim();
    if (scenario) tags.push(`scenario:${scenario}`);
    if (skill) tags.push(`skill:${skill}`);
    if (tool) tags.push(`tool:${tool}`);
    if (tags.length) body.tags = tags;
    const ref = String(fd.get("evidenceRef") || "");
    if (layer !== "L0" && ref) {
      body.evidence = [{ kind: "file", ref }];
    }
    createMut.mutate(body, {
      onSuccess: () => e.currentTarget.reset(),
    });
  };

  const rawItems = candidatesQuery.data?.items ?? [];
  const layerCounts = useMemo(() => {
    const counts = { 全部: rawItems.length, L0: 0, L1: 0, L2: 0 };
    for (const item of rawItems) {
      if (item.layer === "L0" || item.layer === "L1" || item.layer === "L2") {
        counts[item.layer] += 1;
      }
    }
    return counts;
  }, [rawItems]);
  const items = useMemo(() => {
    const scoped =
      perspective === "layer" && layerFilter !== "全部"
        ? rawItems.filter((m) => m.layer === layerFilter)
        : rawItems;
    if (perspective === "layer") return sortByLayer(scoped);
    return [...scoped].sort((a, b) => perspectiveKey(a, perspective).localeCompare(perspectiveKey(b, perspective)));
  }, [perspective, layerFilter, rawItems]);
  const unlabeled =
    perspective !== "layer" && items.every((item) => perspectiveKey(item, perspective) === "未标注");
  const perspectiveHint = unlabeled ? PERSPECTIVE_HINTS[perspective] : undefined;
  const selected = detailQuery.data;
  const ttlQueue = ttlQueueQuery.data;
  const ttlReviewItems = ttlQueue?.reviewDue ?? [];

  return (
    <section className="panel active">
      <div className="page-kicker">
        <DatabaseZap size={17} strokeWidth={1.8} />
        记忆审核
      </div>
      <div className="page-heading">
        <div>
          <h1>记忆</h1>
          <p>审核记忆候选、查看治理边关系，并检索已批准记忆；TTL 到期前进入复核队列。</p>
          <span className="scope-badge">Space: {activeSpaceId}</span>
        </div>
        <div className="toolbar">
          <Link to="/reviews" className="btn icon-btn" data-testid="memory-goto-reviews">
            去评审
          </Link>
          {pillarTab === "memory" && (
            <>
              <button className="btn icon-btn" onClick={() => candidatesQuery.refetch()}>
                <RefreshCw size={16} strokeWidth={1.8} />
                刷新候选
              </button>
              <button className="btn icon-btn" onClick={() => ttlQueueQuery.refetch()}>
                <Clock size={16} strokeWidth={1.8} />
                刷新 TTL
              </button>
            </>
          )}
        </div>
      </div>

      <div
        className="work-mode"
        role="tablist"
        aria-label="记忆板块切换"
        data-testid="memory-pillar-tabs"
        style={{ marginBottom: "0.85rem" }}
      >
        {PILLAR_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={pillarTab === tab.id}
            className={pillarTab === tab.id ? "work-mode-btn active" : "work-mode-btn"}
            data-testid={`memory-tab-${tab.id}`}
            onClick={() => setPillarTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {pillarTab === "knowledge" ? (
        <KnowledgePanel embedded />
      ) : pillarTab === "links" ? (
        <div className="pane" data-testid="memory-links-pane">
          <div className="pane-title">
            <h2>MemoryLink</h2>
          </div>
          <p className="muted-line">按 Run 查看 Thread 关联的记忆；可从 Agent Chat「记忆」深链进入。</p>
          <label className="scenario-picker" style={{ display: "block", marginBottom: 8 }}>
            Run ID
            <input
              data-testid="memory-links-run"
              value={linkRunId}
              placeholder="run_…"
              onChange={(e) => setLinkRunId(e.target.value)}
            />
          </label>
          {linkRunId.trim() ? (
            <MemoryLinkPanel runId={linkRunId.trim()} />
          ) : (
            <p className="muted-line">输入 Run ID 后加载 MemoryLink。</p>
          )}
        </div>
      ) : (
        <>
          <div
            className="work-mode"
            role="group"
            aria-label="记忆视角"
            data-testid="memory-perspective"
            style={{ marginBottom: "0.85rem" }}
          >
            {PERSPECTIVES.map((item) => (
              <button
                key={item.id}
                type="button"
                aria-pressed={perspective === item.id}
                className={perspective === item.id ? "work-mode-btn active" : "work-mode-btn"}
                data-testid={`memory-perspective-${item.id}`}
                onClick={() => {
                  setPerspective(item.id);
                  if (item.id !== "layer") setLayerFilter("全部");
                }}
              >
                {item.label}
              </button>
            ))}
          </div>
          {perspectiveHint && (
            <p className="muted-line" data-testid="memory-perspective-hint">
              {perspectiveHint}
            </p>
          )}
          {perspective === "layer" && (
            <div
              className="toolbar"
              data-testid="memory-layer-filter"
              style={{ marginBottom: "0.75rem", gap: "0.35rem" }}
            >
              {LAYER_FILTERS.map((layer) => (
                <button
                  key={layer}
                  type="button"
                  className={layerFilter === layer ? "btn mini primary" : "btn mini"}
                  aria-pressed={layerFilter === layer}
                  data-testid={`memory-layer-${layer}`}
                  onClick={() => setLayerFilter(layer)}
                >
                  {layer}
                  <span className="muted" style={{ marginLeft: 4 }}>
                    {layerCounts[layer]}
                  </span>
                </button>
              ))}
            </div>
          )}

          <div className="split">
            <div className="pane">
              <div className="pane-title">
                <h2>候选列表</h2>
                <span>{items.length} 条</span>
              </div>
              <table className="table">
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>层级</th>
                    {perspective !== "layer" ? <th>分组</th> : null}
                    <th>标题</th>
                    <th>状态</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((m) => (
                    <tr
                      key={m.id}
                      className={m.id === selectedId ? "selected" : ""}
                      onClick={() => setSelectedId(m.id)}
                    >
                      <td title={m.id}>{shortId(m.id)}</td>
                      <td>{m.layer}</td>
                      {perspective !== "layer" ? (
                        <td data-testid="memory-perspective-group">{perspectiveKey(m, perspective)}</td>
                      ) : null}
                      <td>{m.title}</td>
                      <td>{memoryStatusLabel(m.status)}</td>
                      <td>
                        <div className="row-actions" onClick={(e) => e.stopPropagation()}>
                          <a
                            className="btn small"
                            data-testid={`memory-thick-review-${m.id}`}
                            href={`/ui/reviews?queue=memory&memoryId=${encodeURIComponent(m.id)}`}
                            title="到管控台厚评审"
                          >
                            厚审
                          </a>
                          {m.status === "candidate" && (
                            <>
                              <button
                                className="btn small icon-only ok"
                                aria-label="通过候选"
                                onClick={() => reviewMut.mutate({ id: m.id, decision: "approve" })}
                              >
                                <Check size={14} strokeWidth={2} />
                              </button>
                              <button
                                className="btn small icon-only err"
                                aria-label="拒绝候选"
                                onClick={() => reviewMut.mutate({ id: m.id, decision: "reject" })}
                              >
                                <X size={14} strokeWidth={2} />
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                  {!items.length && (
                    <tr className="empty-row">
                      <td colSpan={perspective === "layer" ? 5 : 6}>暂无记忆候选。</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            <div className="pane">
              <div className="pane-title">
                <h2>治理详情</h2>
                <span>{selected ? shortId(selected.id) : "未选择"}</span>
              </div>
              {selected ? (
                <>
                  <p className="muted-line">dedupe: {selected.dedupeKey ? shortId(selected.dedupeKey) : "-"}</p>
                  <table className="table">
                    <thead>
                      <tr>
                        <th>关系</th>
                        <th>目标</th>
                        <th>说明</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(selected.edges ?? []).map((edge) => (
                        <tr key={edge.id}>
                          <td>{edgeKindLabel(edge.kind)}</td>
                          <td title={edge.toId}>{shortId(edge.toId)}</td>
                          <td title={edge.reason}>{edge.reason || "-"}</td>
                        </tr>
                      ))}
                      {!selected.edges?.length && (
                        <tr className="empty-row">
                          <td colSpan={3}>暂无治理边。</td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </>
              ) : (
                <p className="muted-line">选择一条记忆查看去重/冲突/替代关系。</p>
              )}
            </div>
            <div className="pane">
              <div className="pane-title">
                <h2>新建候选</h2>
                <span>手动</span>
              </div>
              <form className="form" onSubmit={onSubmit}>
                <label>
                  运行 ID <span className="muted">(可选，关联 SSE)</span>
                  <input name="runId" placeholder="run_..." value={runId} onChange={(e) => setRunId(e.target.value)} />
                </label>
                <label>
                  层级
                  <select name="layer" defaultValue="L1">
                    <option>L0</option>
                    <option>L1</option>
                    <option>L2</option>
                  </select>
                </label>
                <label>
                  标题
                  <input name="title" required defaultValue="M1 治理规则" />
                </label>
                <label>
                  内容
                  <textarea name="body" rows={3} required defaultValue="合并前始终运行测试与 doctor。" />
                </label>
                <label>
                  场景 / Skill / 工具 <span className="muted">(可选，用于视角分组)</span>
                  <input name="scenario" placeholder="scenario，如 hotfix" />
                  <input name="skill" placeholder="skill" />
                  <input name="tool" placeholder="tool" />
                </label>
                <label>
                  证据引用 (L1+)
                  <input name="evidenceRef" defaultValue="doc/README.md" />
                </label>
                {governanceMsg && <p className="warn-text">{governanceMsg}</p>}
                <button type="submit" className="btn primary icon-btn" disabled={createMut.isPending}>
                  <Send size={16} strokeWidth={1.8} />
                  提交候选
                </button>
              </form>
            </div>
          </div>
          <div className="split">
            <div className="pane">
              <div className="pane-title">
                <h2>
                  <Clock size={15} strokeWidth={1.8} />
                  TTL 复核队列
                </h2>
                <span>
                  {ttlQueueQuery.isFetching
                    ? "加载中"
                    : `${ttlQueue?.reviewDueCount ?? 0} 待复核 · ${ttlQueue?.expiredPendingCount ?? 0} 待 sweep`}
                </span>
              </div>
              {ttlQueueQuery.isError && (
                <p className="error-text">{(ttlQueueQuery.error as Error).message}</p>
              )}
              {(ttlQueue?.expiredPendingCount ?? 0) > 0 && (
                <p className="warn-text">
                  有 {ttlQueue!.expiredPendingCount} 条已过期记忆待弃用。
                  <button
                    className="btn mini"
                    type="button"
                    disabled={ttlSweepMut.isPending}
                    onClick={() => ttlSweepMut.mutate()}
                    style={{ marginLeft: "0.5rem" }}
                  >
                    {ttlSweepMut.isPending ? "处理中…" : "执行 TTL sweep"}
                  </button>
                </p>
              )}
              {ttlMsg && <p className="muted-line">{ttlMsg}</p>}
              <p className="muted-line">
                到期前 {ttlQueue?.reviewLeadDays ?? 7} 天内进入复核；过期后需 sweep 弃用且不可检索。
              </p>
              <table className="table">
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>层级</th>
                    <th>标题</th>
                    <th>剩余天数</th>
                    <th>到期时间</th>
                  </tr>
                </thead>
                <tbody>
                  {ttlReviewItems.map((row) => (
                    <tr
                      key={row.recordId}
                      className={row.recordId === selectedId ? "selected" : ""}
                      onClick={() => setSelectedId(row.recordId)}
                    >
                      <td title={row.recordId}>{shortId(row.recordId)}</td>
                      <td>{row.layer}</td>
                      <td>{row.title}</td>
                      <td className={row.daysRemaining <= 3 ? "error-text" : undefined}>
                        {row.daysRemaining} 天
                      </td>
                      <td>{fmtTime(row.expiresAtMs)}</td>
                    </tr>
                  ))}
                  {!ttlReviewItems.length && (
                    <tr className="empty-row">
                      <td colSpan={5}>暂无 TTL 复核项。</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            <div className="pane">
              <div className="pane-title">
                <h2>
                  <Search size={15} strokeWidth={1.8} />
                  记忆检索
                </h2>
                <button className="btn mini" type="button" onClick={() => queryMut.mutate()} disabled={queryMut.isPending}>
                  查询
                </button>
              </div>
              <label>
                关键词
                <input value={queryText} onChange={(e) => setQueryText(e.target.value)} />
              </label>
              <table className="table">
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>标题</th>
                    <th>层级</th>
                  </tr>
                </thead>
                <tbody>
                  {(queryMut.data?.items ?? []).map((row: MemoryRecord) => (
                    <tr key={row.id} onClick={() => setSelectedId(row.id)}>
                      <td>{shortId(row.id)}</td>
                      <td>{row.title}</td>
                      <td>{row.layer}</td>
                    </tr>
                  ))}
                  {!queryMut.data?.items.length && (
                    <tr className="empty-row">
                      <td colSpan={3}>尚无检索结果。</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </section>
  );
}
