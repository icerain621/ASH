import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, KanbanSquare, MessageSquarePlus, RefreshCcw, Star } from "lucide-react";
import { useMemo, useState } from "react";
import {
  createDiffComment,
  getQuestBoard,
  getRunDiff,
  listDiffComments,
  rateRunStep,
  rejectRunDiff,
  type BoardItem,
  type DiffComment,
  type DiffFile,
} from "@/modules/quest/api/quest.api";
import {
  approveGoalPlan,
  approveRun,
  cancelRun,
  createRunFromGoal,
  getGoalPlan,
  getRun,
  getRunArtifactAccess,
  getRunArtifacts,
  getRunTimeline,
  getRunTree,
  rejectGoalPlan,
  type ArtifactAccessResponse,
  type ArtifactItem,
  type GoalPlan,
  type RunTreeNode,
  type TimelineItem,
} from "@/modules/runs/api/runs.api";
import { getCurrentSpaceId } from "@/services/http/client";
import { useQuestBoardStream } from "@/services/sse/questBoardStream";
import { shortId } from "@/shared/utils/format";

const COLUMNS: Array<{ key: string; title: string }> = [
  { key: "plans", title: "Plans" },
  { key: "running", title: "Running" },
  { key: "waiting_approval", title: "Waiting" },
  { key: "finished", title: "Finished" },
];

function artifactItems(data: { artifacts?: ArtifactItem[]; manifest?: { artifacts?: ArtifactItem[] } } | undefined) {
  return data?.artifacts ?? data?.manifest?.artifacts ?? [];
}

function waitingGate(items: TimelineItem[] | undefined) {
  const item = [...(items ?? [])].reverse().find((entry) => entry.type === "gate.waiting_approval");
  const payload =
    item?.payload && typeof item.payload === "object" && !Array.isArray(item.payload)
      ? (item.payload as Record<string, unknown>)
      : null;
  if (!item || !payload) return null;
  return {
    gate: typeof payload.gate === "string" ? payload.gate : "human",
    reason: typeof payload.reason === "string" ? payload.reason : "",
    stepId: typeof payload.stepId === "string" ? payload.stepId : "",
    tool: typeof payload.tool === "string" ? payload.tool : "",
  };
}

function gateTitle(gate: { gate: string; tool?: string }): string {
  switch (gate.gate) {
    case "citation":
      return "引用门禁";
    case "tool_risk":
      return gate.tool ? `危险工具审批 · ${gate.tool}` : "危险工具审批";
    case "human":
      return "人工步骤审批";
    default:
      return `审批门禁 · ${gate.gate}`;
  }
}

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
  const [goalText, setGoalText] = useState("");
  const [goalRepo, setGoalRepo] = useState(".");
  const [activePlan, setActivePlan] = useState<GoalPlan | null>(null);
  const [artifactAccess, setArtifactAccess] = useState<ArtifactAccessResponse | null>(null);

  const boardQuery = useQuery({
    queryKey: ["quest-board", spaceId],
    queryFn: () => getQuestBoard(80),
  });
  const { status: boardStreamStatus } = useQuestBoardStream(spaceId, {
    onBoardEvent: () => {
      void qc.invalidateQueries({ queryKey: ["quest-board", spaceId] });
    },
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
  const runQuery = useQuery({
    queryKey: ["quest-run", selectedRunId],
    queryFn: () => getRun(selectedRunId!),
    enabled: !!selectedRunId,
  });
  const artifactsQuery = useQuery({
    queryKey: ["quest-artifacts", selectedRunId],
    queryFn: () => getRunArtifacts(selectedRunId!),
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

  const fromGoalMut = useMutation({
    mutationFn: () =>
      createRunFromGoal({
        goal: goalText.trim(),
        repoRoot: goalRepo.trim() || undefined,
        spaceId,
      }),
    onSuccess: (plan) => {
      setActivePlan(plan);
      setSelectedRunId(null);
      setMessage(`Plan ${shortId(plan.id)} · ${plan.status}`);
      qc.invalidateQueries({ queryKey: ["quest-board", spaceId] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const approvePlanMut = useMutation({
    mutationFn: (planId: string) =>
      approveGoalPlan(planId, { actorId: "console", reason: "approved from Quest workbench" }),
    onSuccess: (plan) => {
      setActivePlan(plan);
      if (plan.runId) {
        setSelectedRunId(plan.runId);
        setSelectedFile("");
        setMessage(`已批准并启动 ${shortId(plan.runId)}`);
      } else {
        setMessage(`Plan ${shortId(plan.id)} 已批准`);
      }
      qc.invalidateQueries({ queryKey: ["quest-board", spaceId] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const rejectPlanMut = useMutation({
    mutationFn: (planId: string) =>
      rejectGoalPlan(planId, { actorId: "console", reason: "rejected from Quest workbench" }),
    onSuccess: (plan) => {
      setActivePlan(plan);
      setMessage(`Plan ${shortId(plan.id)} 已拒绝`);
      qc.invalidateQueries({ queryKey: ["quest-board", spaceId] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const rejectDiffMut = useMutation({
    mutationFn: (body: { scope: "file" | "all"; filePath?: string; reason: string }) =>
      rejectRunDiff(selectedRunId!, { ...body, actorId: "console" }),
    onSuccess: (resp) => {
      setMessage(
        resp.canceled
          ? `已全量拒绝 Diff · Run ${resp.status}`
          : `已拒绝文件 ${resp.filePath}`,
      );
      qc.invalidateQueries({ queryKey: ["quest-diff", selectedRunId] });
      qc.invalidateQueries({ queryKey: ["quest-diff-comments", selectedRunId] });
      qc.invalidateQueries({ queryKey: ["quest-run", selectedRunId] });
      qc.invalidateQueries({ queryKey: ["quest-board", spaceId] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const approveGateMut = useMutation({
    mutationFn: () => approveRun(selectedRunId!, { actorId: "console", reason: "approved from Quest Diff" }),
    onSuccess: () => {
      setMessage(`已批准门禁 · ${shortId(selectedRunId!)}`);
      qc.invalidateQueries({ queryKey: ["quest-run", selectedRunId] });
      qc.invalidateQueries({ queryKey: ["quest-board", spaceId] });
      qc.invalidateQueries({ queryKey: ["quest-timeline", selectedRunId] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const cancelMut = useMutation({
    mutationFn: () => cancelRun(selectedRunId!),
    onSuccess: (resp) => {
      setMessage(`已取消 Run · ${resp.status}`);
      qc.invalidateQueries({ queryKey: ["quest-run", selectedRunId] });
      qc.invalidateQueries({ queryKey: ["quest-board", spaceId] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const artifactAccessMut = useMutation({
    mutationFn: (name: string) => getRunArtifactAccess(selectedRunId!, name),
    onSuccess: (resp) => {
      setArtifactAccess(resp);
      setMessage(`产物链接已生成 · ${resp.name}`);
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const files = diffQuery.data?.files ?? [];
  const artifacts = useMemo(() => artifactItems(artifactsQuery.data), [artifactsQuery.data]);
  const gate = useMemo(() => waitingGate(timelineQuery.data?.items), [timelineQuery.data]);
  const rejectedPaths = diffQuery.data?.rejectedPaths ?? [];
  const fullRejected = rejectedPaths.includes("*");
  const runStatus = runQuery.data?.status;
  const canCancel =
    !!selectedRunId && !!runStatus && !["finished", "failed", "canceled"].includes(runStatus);
  const activeFile: DiffFile | undefined = useMemo(() => {
    if (!files.length) return undefined;
    const path = selectedFile || files[0].path;
    return files.find((f) => f.path === path) ?? files[0];
  }, [files, selectedFile]);
  const fileRejected = !!(activeFile?.path && rejectedPaths.includes(activeFile.path));

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

  async function selectItem(item: BoardItem) {
    if (item.kind === "run" && item.runId) {
      setSelectedRunId(item.runId);
      setSelectedFile("");
      setActivePlan(null);
      setArtifactAccess(null);
      setMessage(`审查 ${shortId(item.runId)}`);
      return;
    }
    const planId = item.planId || (item.kind === "plan" ? item.id : "");
    if (planId) {
      setSelectedRunId(null);
      try {
        const plan = await getGoalPlan(planId);
        setActivePlan(plan);
        setMessage(`Plan ${shortId(plan.id)} · ${plan.status}`);
      } catch (e) {
        setMessage(e instanceof Error ? e.message : String(e));
      }
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
          <p>从目标生成 Plan 并批准；看板跟踪 Plan/Run；深 Diff 行级批注；步骤评分；Sub-run 树。</p>
          <span className="scope-badge">Space: {spaceId}</span>
        </div>
        <button type="button" className="btn icon-btn" onClick={() => boardQuery.refetch()}>
          <RefreshCcw size={16} /> 刷新
        </button>
      </div>
      <p className="muted-line" data-testid="quest-stream-status" data-stream-status={boardStreamStatus}>
        看板 live：
        {boardStreamStatus === "open"
          ? "已连接"
          : boardStreamStatus === "reconnecting"
            ? "重连中"
            : boardStreamStatus === "polling"
              ? "轮询回退"
              : boardStreamStatus}
      </p>
      {message ? <p className="muted-line">{message}</p> : null}

      <div className="pane" data-testid="quest-wb-compose" style={{ marginBottom: "1rem" }}>
        <div className="pane-title">
          <h2>从目标创建</h2>
          <span>{activePlan ? activePlan.status : "draft plan"}</span>
        </div>
        <div className="secret-form">
          <label className="wide-field">
            Goal
            <input
              value={goalText}
              onChange={(e) => setGoalText(e.target.value)}
              placeholder="例如：紧急热修线上支付 或 Add dark mode"
              data-testid="quest-wb-goal-input"
            />
          </label>
          <label>
            repoRoot
            <input
              value={goalRepo}
              onChange={(e) => setGoalRepo(e.target.value)}
              data-testid="quest-wb-repo-input"
            />
          </label>
          <button
            className="btn primary"
            type="button"
            disabled={fromGoalMut.isPending || !goalText.trim()}
            onClick={() => fromGoalMut.mutate()}
            data-testid="quest-wb-route"
          >
            生成 Plan
          </button>
        </div>
        {activePlan ? (
          <div data-testid="quest-wb-plan-preview">
            <p className="muted-line">
              {activePlan.scenarioName}@{activePlan.scenarioVersion} · {activePlan.routeReason} ·{" "}
              {activePlan.steps?.length ?? 0} steps
            </p>
            <pre className="code-block compact">
              {JSON.stringify({ inputs: activePlan.inputs, steps: activePlan.steps }, null, 2)}
            </pre>
            {activePlan.status === "draft" ? (
              <div className="row-actions">
                <button
                  className="btn mini ok"
                  type="button"
                  disabled={approvePlanMut.isPending}
                  onClick={() => approvePlanMut.mutate(activePlan.id)}
                  data-testid="quest-wb-approve"
                >
                  批准并启动
                </button>
                <button
                  className="btn mini err"
                  type="button"
                  disabled={rejectPlanMut.isPending}
                  onClick={() => rejectPlanMut.mutate(activePlan.id)}
                  data-testid="quest-wb-reject"
                >
                  拒绝
                </button>
              </div>
            ) : null}
          </div>
        ) : null}
      </div>

      <div
        className="quest-board"
        data-testid="quest-board"
        style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: "0.75rem", marginBottom: "1rem" }}
      >
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
                    onClick={() => void selectItem(item)}
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
        <>
          {runStatus === "waiting_approval" ? (
            <div className="pane" data-testid="quest-gate-panel" style={{ marginBottom: "1rem" }}>
              <div className="pane-title">
                <h2>等待审批</h2>
                <span>{gate ? gateTitle(gate) : "waiting_approval"}</span>
              </div>
              <p className="muted-line" data-testid="quest-gate-detail">
                {gate?.reason || "Run 处于 waiting_approval，可批准继续或取消。"}
                {gate?.stepId ? ` · step ${gate.stepId}` : ""}
              </p>
              <div className="row-actions">
                <button
                  type="button"
                  className="btn mini ok"
                  disabled={approveGateMut.isPending || fullRejected}
                  onClick={() => approveGateMut.mutate()}
                  data-testid="quest-gate-approve"
                >
                  批准并继续
                </button>
                <button
                  type="button"
                  className="btn mini err"
                  disabled={!canCancel || cancelMut.isPending}
                  onClick={() => cancelMut.mutate()}
                  data-testid="quest-gate-cancel"
                >
                  取消 Run
                </button>
              </div>
            </div>
          ) : null}
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
              <span>
                {shortId(selectedRunId)}
                {runStatus ? ` · ${runStatus}` : ""}
              </span>
            </div>
            <div className="row-actions" data-testid="quest-diff-actions" style={{ marginBottom: "0.5rem" }}>
              <button
                type="button"
                className="btn mini err"
                disabled={!activeFile?.path || fileRejected || fullRejected || rejectDiffMut.isPending}
                onClick={() =>
                  rejectDiffMut.mutate({
                    scope: "file",
                    filePath: activeFile!.path,
                    reason: `reject file ${activeFile!.path}`,
                  })
                }
                data-testid="quest-diff-reject-file"
              >
                拒绝当前文件
              </button>
              <button
                type="button"
                className="btn mini err"
                disabled={fullRejected || rejectDiffMut.isPending}
                onClick={() => rejectDiffMut.mutate({ scope: "all", reason: "reject full diff" })}
                data-testid="quest-diff-reject-all"
              >
                拒绝全部 Diff
              </button>
              {runStatus === "waiting_approval" ? (
                <button
                  type="button"
                  className="btn mini ok"
                  disabled={approveGateMut.isPending || fullRejected}
                  onClick={() => approveGateMut.mutate()}
                  data-testid="quest-diff-approve-gate"
                >
                  批准并继续
                </button>
              ) : null}
            </div>
            {rejectedPaths.length ? (
              <p className="muted-line" data-testid="quest-diff-rejected">
                已拒绝：{rejectedPaths.join(", ")}
              </p>
            ) : null}
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

          <div className="pane" data-testid="quest-artifacts-pane">
            <div className="pane-title">
              <h2>产物</h2>
              <span>{artifactsQuery.isFetching ? "加载中" : `${artifacts.length} 个`}</span>
            </div>
            <table className="table compact">
              <thead>
                <tr>
                  <th>名称</th>
                  <th>类型</th>
                  <th>Digest</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {artifacts.map((artifact) => (
                  <tr key={`${artifact.type}-${artifact.name}`}>
                    <td title={artifact.uri}>{artifact.name}</td>
                    <td>{artifact.type}</td>
                    <td title={artifact.digest}>{artifact.digest ? shortId(artifact.digest) : "—"}</td>
                    <td>
                      <button
                        type="button"
                        className="btn icon-btn mini"
                        data-testid={`quest-artifact-link-${artifact.name}`}
                        disabled={artifactAccessMut.isPending}
                        onClick={() => artifactAccessMut.mutate(artifact.name)}
                      >
                        <Download size={14} /> 链接
                      </button>
                    </td>
                  </tr>
                ))}
                {!artifacts.length ? (
                  <tr className="empty-row">
                    <td colSpan={4}>暂无产物。</td>
                  </tr>
                ) : null}
              </tbody>
            </table>
            {artifactAccess ? (
              <p className="muted-line" data-testid="quest-artifact-access">
                {artifactAccess.name}: <code>{artifactAccess.signedUrl}</code>
              </p>
            ) : null}
          </div>
        </div>
        </>
      ) : (
        <p className="muted-line">
          {activePlan
            ? "批准 Plan 后可审查 Diff，或从看板选择一条 Run。"
            : "从看板选择一条 Run 以审查 Diff，或先生成 Plan。"}
        </p>
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
