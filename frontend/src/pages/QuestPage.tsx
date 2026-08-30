import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { KanbanSquare, MessageSquarePlus, RefreshCcw, Star } from "lucide-react";
import { useMemo, useState } from "react";
import {
  createDiffComment,
  getQuestBoard,
  getRunDiff,
  listDiffComments,
  rateRunStep,
  type BoardItem,
  type DiffComment,
  type DiffFile,
} from "@/modules/quest/api/quest.api";
import { getRunTimeline, getRunTree, type RunTreeNode, type TimelineItem } from "@/modules/runs/api/runs.api";
import { getCurrentSpaceId } from "@/services/http/client";
import { shortId } from "@/shared/utils/format";

const COLUMNS: Array<{ key: string; title: string }> = [
  { key: "plans", title: "Plans" },
  { key: "running", title: "Running" },
  { key: "waiting_approval", title: "Waiting" },
  { key: "finished", title: "Finished" },
];

export function QuestPage() {
  const qc = useQueryClient();
  const spaceId = getCurrentSpaceId();
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const [selectedFile, setSelectedFile] = useState<string>("");
  const [draftComment, setDraftComment] = useState("");
  const [anchor, setAnchor] = useState<{ filePath: string; lineIndex: number; side: string } | null>(null);
  const [stepId, setStepId] = useState("");
  const [rating, setRating] = useState(4);
  const [message, setMessage] = useState("");

  const boardQuery = useQuery({
    queryKey: ["quest-board", spaceId],
    queryFn: () => getQuestBoard(80),
  });
  const diffQuery = useQuery({
    queryKey: ["quest-diff", selectedRunId],
    queryFn: () => getRunDiff(selectedRunId!),
    enabled: !!selectedRunId,
  });
  const commentsQuery = useQuery({
    queryKey: ["quest-diff-comments", selectedRunId],
    queryFn: () => listDiffComments(selectedRunId!),
    enabled: !!selectedRunId,
  });
  const timelineQuery = useQuery({
    queryKey: ["quest-timeline", selectedRunId],
    queryFn: () => getRunTimeline(selectedRunId!),
    enabled: !!selectedRunId,
  });
  const treeQuery = useQuery({
    queryKey: ["quest-run-tree", selectedRunId],
    queryFn: () => getRunTree(selectedRunId!),
    enabled: !!selectedRunId,
  });

  const commentMut = useMutation({
    mutationFn: () =>
      createDiffComment(selectedRunId!, {
        filePath: anchor!.filePath,
        lineIndex: anchor!.lineIndex,
        side: anchor!.side,
        body: draftComment.trim(),
      }),
    onSuccess: () => {
      setDraftComment("");
      setAnchor(null);
      setMessage("批注已保存");
      qc.invalidateQueries({ queryKey: ["quest-diff-comments", selectedRunId] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const rateMut = useMutation({
    mutationFn: () => rateRunStep(selectedRunId!, stepId.trim(), { rating, comment: "quest step rating" }),
    onSuccess: () => setMessage(`步骤 ${stepId} 已评分 ${rating}`),
    onError: (e: Error) => setMessage(e.message),
  });

  const files = diffQuery.data?.files ?? [];
  const activeFile: DiffFile | undefined = useMemo(() => {
    if (!files.length) return undefined;
    const path = selectedFile || files[0].path;
    return files.find((f) => f.path === path) ?? files[0];
  }, [files, selectedFile]);

  const commentsByLine = useMemo(() => {
    const map = new Map<string, DiffComment[]>();
    for (const c of commentsQuery.data?.items ?? []) {
      const key = `${c.filePath}#${c.lineIndex}`;
      const list = map.get(key) ?? [];
      list.push(c);
      map.set(key, list);
    }
    return map;
  }, [commentsQuery.data]);

  const stepIds = useMemo(() => {
    const set = new Set<string>();
    for (const item of timelineQuery.data?.items ?? []) {
      if (item.stepId) set.add(item.stepId);
    }
    return Array.from(set);
  }, [timelineQuery.data]);

  function selectItem(item: BoardItem) {
    if (item.kind === "run" && item.runId) {
      setSelectedRunId(item.runId);
      setSelectedFile("");
      setMessage(`审查 ${shortId(item.runId)}`);
    } else if (item.planId) {
      setMessage(`Plan ${shortId(item.planId)} · ${item.status}（在 Runs 页批准）`);
      setSelectedRunId(null);
    }
  }

  function renderTree(node: RunTreeNode, depth = 0) {
    const id = node.summary.runId;
    return (
      <li key={id} style={{ marginLeft: depth * 12 }}>
        <button
          type="button"
          className="btn linkish"
          onClick={() => {
            setSelectedRunId(id);
            setMessage(`树节点 ${shortId(id)} · depth ${node.summary.depth ?? 0}`);
          }}
          data-testid={`quest-tree-node-${id}`}
        >
          {shortId(id)} · {node.summary.status} · d{node.summary.depth ?? 0}
          {node.summary.parentRunId ? " · child" : " · root"}
        </button>
        {node.children?.length ? (
          <ul style={{ listStyle: "none", paddingLeft: 0 }}>{node.children.map((c) => renderTree(c, depth + 1))}</ul>
        ) : null}
      </li>
    );
  }

  return (
    <section className="panel active" data-testid="quest-page">
      <div className="page-kicker">
        <KanbanSquare size={17} strokeWidth={1.8} />
        Quest
      </div>
      <div className="page-heading">
        <div>
          <h1>Quest 工作台</h1>
          <p>看板跟踪 Plan/Run；深 Diff 行级批注；步骤评分；Sub-run 树。</p>
          <span className="scope-badge">Space: {spaceId}</span>
        </div>
        <button type="button" className="btn icon-btn" onClick={() => boardQuery.refetch()}>
          <RefreshCcw size={16} /> 刷新
        </button>
      </div>
      {message ? <p className="muted-line">{message}</p> : null}

      <div className="quest-board" data-testid="quest-board" style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: "0.75rem", marginBottom: "1rem" }}>
        {COLUMNS.map((col) => (
          <div key={col.key} className="pane">
            <div className="pane-title">
              <h2>{col.title}</h2>
              <span>{boardQuery.data?.columns?.[col.key]?.length ?? 0}</span>
            </div>
            <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
              {(boardQuery.data?.columns?.[col.key] ?? []).map((item) => (
                <li key={item.id} style={{ marginBottom: "0.5rem" }}>
                  <button
                    type="button"
                    className="btn"
                    style={{ width: "100%", textAlign: "left" }}
                    onClick={() => selectItem(item)}
                    data-testid={`quest-card-${item.kind}`}
                  >
                    <strong>{item.title}</strong>
                    <div className="muted-line">
                      {item.kind} · {item.status}
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>

      {selectedRunId ? (
        <div className="split ops-split">
          <div className="pane" data-testid="quest-run-tree">
            <div className="pane-title">
              <h2>Sub-run 树</h2>
              <span>{treeQuery.data?.rootRunId ? shortId(treeQuery.data.rootRunId) : "—"}</span>
            </div>
            {treeQuery.isError ? <p className="error-text">加载树失败</p> : null}
            {treeQuery.data?.tree ? (
              <ul style={{ listStyle: "none", padding: 0 }} data-testid="quest-run-tree-list">
                {renderTree(treeQuery.data.tree)}
              </ul>
            ) : (
              <p className="muted-line">选择 Run 后显示 spawn 树。</p>
            )}
          </div>
          <div className="pane" data-testid="quest-diff-pane">
            <div className="pane-title">
              <h2>Diff 审查</h2>
              <span>{shortId(selectedRunId)}</span>
            </div>
            {diffQuery.data?.contextRefs?.length ? (
              <p className="muted-line">contextRefs: {diffQuery.data.contextRefs.slice(0, 8).join(", ")}</p>
            ) : (
              <p className="muted-line">contextRefs: （空或未写入）</p>
            )}
            <div className="toolbar">
              <select
                value={activeFile?.path ?? ""}
                onChange={(e) => setSelectedFile(e.target.value)}
                data-testid="quest-diff-file"
              >
                {files.map((f) => (
                  <option key={f.path} value={f.path}>
                    {f.path}
                  </option>
                ))}
              </select>
            </div>
            {!files.length ? <p className="muted-line">无 diff 产物</p> : null}
            {activeFile?.hunks.map((hunk, hi) => (
              <div key={`${activeFile.path}-${hi}`} style={{ marginBottom: "0.75rem" }}>
                <pre className="code-block compact">{hunk.header}</pre>
                <div className="code-block" style={{ padding: 0 }}>
                  {hunk.lines.map((ln) => {
                    const key = `${activeFile.path}#${ln.index}`;
                    const notes = commentsByLine.get(key) ?? [];
                    const tone = ln.kind === "add" ? "#0a3" : ln.kind === "del" ? "#a30" : "inherit";
                    return (
                      <div key={ln.index}>
                        <button
                          type="button"
                          style={{
                            display: "block",
                            width: "100%",
                            textAlign: "left",
                            background: "transparent",
                            border: "none",
                            color: tone,
                            fontFamily: "inherit",
                            fontSize: "12px",
                            padding: "1px 8px",
                            cursor: "pointer",
                          }}
                          onClick={() =>
                            setAnchor({
                              filePath: activeFile.path,
                              lineIndex: ln.index,
                              side: ln.kind === "del" ? "old" : "new",
                            })
                          }
                          data-testid="quest-diff-line"
                        >
                          <span style={{ opacity: 0.5, marginRight: 8 }}>{ln.newNo ?? ln.oldNo ?? ""}</span>
                          {ln.text || " "}
                        </button>
                        {notes.map((n) => (
                          <div key={n.id} className="muted-line" style={{ paddingLeft: 24 }}>
                            💬 {n.body}
                          </div>
                        ))}
                      </div>
                    );
                  })}
                </div>
              </div>
            ))}
            {anchor ? (
              <div className="secret-form" data-testid="quest-comment-form">
                <label className="wide-field">
                  批注 · {anchor.filePath}:{anchor.lineIndex}
                  <textarea value={draftComment} onChange={(e) => setDraftComment(e.target.value)} rows={3} />
                </label>
                <button
                  type="button"
                  className="btn primary"
                  disabled={!draftComment.trim() || commentMut.isPending}
                  onClick={() => commentMut.mutate()}
                >
                  <MessageSquarePlus size={14} /> 发表批注
                </button>
              </div>
            ) : null}
          </div>

          <div className="pane">
            <div className="pane-title">
              <h2>步骤评分</h2>
              <span>{stepIds.length} steps</span>
            </div>
            <label className="scenario-picker">
              stepId
              <select value={stepId} onChange={(e) => setStepId(e.target.value)} data-testid="quest-step-select">
                <option value="">选择步骤</option>
                {stepIds.map((id) => (
                  <option key={id} value={id}>
                    {id}
                  </option>
                ))}
              </select>
            </label>
            <label className="scenario-picker">
              rating
              <input type="number" min={1} max={5} value={rating} onChange={(e) => setRating(Number(e.target.value))} />
            </label>
            <button
              type="button"
              className="btn primary"
              disabled={!stepId || rateMut.isPending}
              onClick={() => rateMut.mutate()}
              data-testid="quest-rate-step"
            >
              <Star size={14} /> 提交评分
            </button>
            <TimelineMini items={timelineQuery.data?.items ?? []} />
          </div>
        </div>
      ) : (
        <p className="muted-line">从看板选择一条 Run 以审查 Diff。</p>
      )}
    </section>
  );
}

function TimelineMini({ items }: { items: TimelineItem[] }) {
  const steps = items.filter((i) => i.type === "step.started" || i.type?.startsWith("step."));
  return (
    <ul className="muted-line" style={{ marginTop: "1rem" }}>
      {steps.slice(0, 12).map((i, idx) => (
        <li key={`${i.seq}-${idx}`}>
          {i.type} {i.stepId ? `· ${i.stepId}` : ""}
        </li>
      ))}
    </ul>
  );
}
