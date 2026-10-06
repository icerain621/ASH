import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, KanbanSquare, MessageSquarePlus, RefreshCcw, Settings } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import {
  createDiffComment,
  getQuestBoard,
  getRunDiff,
  listDiffComments,
  rejectRunDiff,
  type BoardItem,
  type DiffComment,
  type DiffFile,
} from "@/modules/quest/api/quest.api";
import { AgentChatShell } from "@/modules/agent-session/components/AgentChatShell";
import { McpToolsPanel } from "@/modules/agent-session/components/McpToolsPanel";
import { SkillsCatalogPanel } from "@/modules/agent-session/components/SkillsCatalogPanel";
import { ToolsRiskPanel } from "@/modules/agent-session/components/ToolsRiskPanel";
import type { AgentSessionView } from "@/modules/agent-session/api/session.api";
import { getAgentSession, submitSessionIntent } from "@/modules/agent-session/api/session.api";
import { MemoryLinkPanel } from "@/modules/interactions/components/MemoryLinkPanel";
import { ThreadTimeline } from "@/modules/interactions/components/ThreadTimeline";
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
  { key: "plans", title: "计划" },
  { key: "running", title: "运行中" },
  { key: "waiting_approval", title: "待审批" },
  { key: "finished", title: "已结束" },
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
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const [highlightSeq, setHighlightSeq] = useState<number | null>(null);
  const [selectedFile, setSelectedFile] = useState<string>("");
  const [draftComment, setDraftComment] = useState("");
  const [anchor, setAnchor] = useState<{ filePath: string; lineIndex: number; side: string } | null>(null);
  const [message, setMessage] = useState("");
  const [goalText, setGoalText] = useState("");
  const [goalRepo, setGoalRepo] = useState(".");
  const [activePlan, setActivePlan] = useState<GoalPlan | null>(null);
  const [artifactAccess, setArtifactAccess] = useState<ArtifactAccessResponse | null>(null);
  const [taskBoardOpen, setTaskBoardOpen] = useState(false);
  const [mcpOpen, setMcpOpen] = useState(false);
  const [toolsOpen, setToolsOpen] = useState(false);
  const [skillsOpen, setSkillsOpen] = useState(false);
  const goalInputRef = useRef<HTMLInputElement>(null);

  const toolsSessionQuery = useQuery({
    queryKey: ["agent-session", selectedSessionId],
    queryFn: () => getAgentSession(selectedSessionId!),
    enabled: Boolean(toolsOpen && selectedSessionId),
  });

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
      setTaskBoardOpen(true);
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

  function selectAgentSession(session: AgentSessionView | null) {
    setSelectedSessionId(session?.id ?? null);
    setHighlightSeq(null);
    setSelectedFile("");
    setArtifactAccess(null);
    const runId = session?.runId?.trim() || null;
    setSelectedRunId(runId);
    if (!session) return;
    const label = (session.title || "").trim() || (session.goal || "").trim() || shortId(session.id);
    if (runId) {
      setActivePlan(null);
      setMessage(`会话 ${label} · run ${shortId(runId)}`);
    } else {
      setMessage(`会话 ${label}`);
    }
  }

  async function selectItem(item: BoardItem) {
    if (item.kind === "run" && item.runId) {
      setSelectedRunId(item.runId);
      setSelectedSessionId(null);
      setHighlightSeq(null);
      setSelectedFile("");
      setActivePlan(null);
      setArtifactAccess(null);
      setMessage(`审查 ${shortId(item.runId)}`);
      return;
    }
    const planId = item.planId || (item.kind === "plan" ? item.id : "");
    if (planId) {
      setSelectedRunId(null);
      setHighlightSeq(null);
      setTaskBoardOpen(true);
      try {
        const plan = await getGoalPlan(planId);
        setActivePlan(plan);
        setMessage(`Plan ${shortId(plan.id)} · ${plan.status}`);
      } catch (e) {
        setMessage(e instanceof Error ? e.message : String(e));
      }
    }
  }

  function openGoalCompose() {
    setActivePlan(null);
    setHighlightSeq(null);
    setArtifactAccess(null);
    setTaskBoardOpen(true);
    setMessage("从目标创建：填写 Goal 以生成 Plan");
    requestAnimationFrame(() => goalInputRef.current?.focus());
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
        Agent
      </div>
      <div className="page-heading">
        <div>
          <h1>Agent</h1>
          <p>大 Chat 为主视线；左侧历史会话；看板与「从目标创建」收入次级任务板。</p>
          <span className="scope-badge">Space: {spaceId}</span>
        </div>
        <div className="row-actions">
          <details className="nav-dropdown" data-testid="agent-settings-menu">
            <summary className="nav-dropdown-trigger">
              <Settings size={14} /> 设置
            </summary>
            <div className="nav-dropdown-menu" role="menu">
              <button
                type="button"
                className="nav-dropdown-item"
                role="menuitem"
                data-testid="agent-settings-tools"
                title="打开 Tools / 风险面板"
                onClick={() => setToolsOpen(true)}
              >
                Tools
              </button>
              <button
                type="button"
                className="nav-dropdown-item"
                role="menuitem"
                data-testid="agent-settings-skills"
                title="打开 Skills 面板"
                onClick={() => setSkillsOpen(true)}
              >
                Skills
              </button>
              <button
                type="button"
                className="nav-dropdown-item"
                role="menuitem"
                data-testid="agent-settings-mcp"
                title="打开 MCP 面板"
                onClick={() => setMcpOpen(true)}
              >
                MCP
              </button>
            </div>
          </details>
          <button
            type="button"
            className="btn icon-btn"
            onClick={() => boardQuery.refetch()}
            title={boardQuery.isFetching ? "刷新中…" : "刷新任务看板"}
          >
            <RefreshCcw size={16} /> 刷新
          </button>
        </div>
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

      <div className="agent-home" data-testid="agent-home">
        <AgentChatShell
          selectedSessionId={selectedSessionId}
          onSelectSession={selectAgentSession}
          runStatus={runStatus}
          gateReason={gate?.reason}
          onOpenTools={() => setToolsOpen(true)}
          onOpenMcp={() => setMcpOpen(true)}
          onOpenSkills={() => setSkillsOpen(true)}
          onIntentSuccess={() => {
            void qc.invalidateQueries({ queryKey: ["quest-run", selectedRunId] });
            void qc.invalidateQueries({ queryKey: ["quest-timeline", selectedRunId] });
            void qc.invalidateQueries({ queryKey: ["quest-board", spaceId] });
            void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
          }}
        />
        <McpToolsPanel
          open={mcpOpen}
          onClose={() => setMcpOpen(false)}
          sessionId={selectedSessionId}
        />
        <ToolsRiskPanel
          open={toolsOpen}
          onClose={() => setToolsOpen(false)}
          session={toolsSessionQuery.data ?? null}
        />
        <SkillsCatalogPanel
          open={skillsOpen}
          onClose={() => setSkillsOpen(false)}
          canRun={Boolean(selectedSessionId)}
          onRunSkill={(slash) => {
            if (!selectedSessionId) {
              setMessage("请先选择或新建会话再运行 Skill");
              return;
            }
            void submitSessionIntent(selectedSessionId, { action: "command", command: slash })
              .then(() => {
                setMessage(`已运行 ${slash}`);
                setSkillsOpen(false);
                void qc.invalidateQueries({ queryKey: ["agent-session", selectedSessionId] });
                void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
              })
              .catch((e: Error) => setMessage(e.message || "Skill 运行失败"));
          }}
        />
      </div>

      {selectedRunId ? (
        <div className="agent-run-ops" data-testid="agent-run-ops">
          {runStatus === "waiting_approval" && !selectedSessionId ? (
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
                  title={
                    fullRejected
                      ? "已拒绝，无法批准"
                      : approveGateMut.isPending
                        ? "批准中…"
                        : "批准门禁并继续（需确认）"
                  }
                  onClick={() => {
                    if (fullRejected) return;
                    const ok = window.confirm("确认批准门禁并继续当前 Run？");
                    if (!ok) return;
                    approveGateMut.mutate();
                  }}
                  data-testid="quest-gate-approve"
                >
                  批准并继续
                </button>
                <button
                  type="button"
                  className="btn mini err"
                  disabled={!canCancel || cancelMut.isPending}
                  title={!canCancel ? "当前 Run 不可取消" : cancelMut.isPending ? "取消中…" : "取消 Run（需确认）"}
                  onClick={() => {
                    if (!canCancel) return;
                    const ok = window.confirm("确认取消当前 Run？");
                    if (!ok) return;
                    cancelMut.mutate();
                  }}
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
                  title={
                    !activeFile?.path
                      ? "需要选择文件"
                      : fileRejected || fullRejected
                        ? "已拒绝"
                        : rejectDiffMut.isPending
                          ? "拒绝中…"
                          : "拒绝当前文件（需确认）"
                  }
                  onClick={() => {
                    if (!activeFile?.path || fileRejected || fullRejected) return;
                    const ok = window.confirm(`确认拒绝文件「${activeFile.path}」？`);
                    if (!ok) return;
                    rejectDiffMut.mutate({
                      scope: "file",
                      filePath: activeFile.path,
                      reason: `reject file ${activeFile.path}`,
                    });
                  }}
                  data-testid="quest-diff-reject-file"
                >
                  拒绝当前文件
                </button>
                <button
                  type="button"
                  className="btn mini err"
                  disabled={fullRejected || rejectDiffMut.isPending}
                  title={
                    fullRejected
                      ? "已拒绝全部"
                      : rejectDiffMut.isPending
                        ? "拒绝中…"
                        : "拒绝全部 Diff（需确认）"
                  }
                  onClick={() => {
                    if (fullRejected) return;
                    const ok = window.confirm("确认拒绝全部 Diff？");
                    if (!ok) return;
                    rejectDiffMut.mutate({ scope: "all", reason: "reject full diff" });
                  }}
                  data-testid="quest-diff-reject-all"
                >
                  拒绝全部 Diff
                </button>
                {runStatus === "waiting_approval" ? (
                  <button
                    type="button"
                    className="btn mini ok"
                    disabled={approveGateMut.isPending || fullRejected}
                    title={
                      fullRejected
                        ? "已拒绝，无法批准"
                        : approveGateMut.isPending
                          ? "批准中…"
                          : "批准门禁并继续（需确认）"
                    }
                    onClick={() => {
                      if (fullRejected) return;
                      const ok = window.confirm("确认批准门禁并继续当前 Run？");
                      if (!ok) return;
                      approveGateMut.mutate();
                    }}
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
              {activeFile?.hunks?.map((hunk, hi) => (
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
                    data-testid="quest-comment-submit"
                    disabled={!draftComment.trim() || commentMut.isPending}
                    title={
                      !draftComment.trim()
                        ? "需要填写批注内容"
                        : commentMut.isPending
                          ? "提交中…"
                          : "发表批注（需确认）"
                    }
                    onClick={() => {
                      const ok = window.confirm(
                        `确认在 ${anchor!.filePath}:${anchor!.lineIndex} 发表批注？`,
                      );
                      if (!ok) return;
                      commentMut.mutate();
                    }}
                  >
                    <MessageSquarePlus size={14} /> 发表批注
                  </button>
                </div>
              ) : null}
            </div>

            <div className="pane" data-testid="quest-timeline-pane">
              <div className="pane-title">
                <h2>步骤时间线</h2>
                <span className="muted">评分请进「评审管控」</span>
              </div>
              <p className="muted-line">本地步骤评分已从 Agent 使用面移除（GV02 去厚）。</p>
              <TimelineMini items={timelineQuery.data?.items ?? []} />
              <ThreadTimeline runId={selectedRunId} highlightSeq={highlightSeq} onSelectSeq={setHighlightSeq} />
              <MemoryLinkPanel
                runId={selectedRunId}
                highlightSeq={highlightSeq}
                onSelectLinkSeq={setHighlightSeq}
              />
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
        </div>
      ) : null}

      <div className="agent-task-board-section">
        <button
          type="button"
          className="btn"
          data-testid="agent-task-board-toggle"
          aria-expanded={taskBoardOpen}
          title={taskBoardOpen ? "收起任务板" : "展开任务板（看板 / 从目标创建）"}
          onClick={() => setTaskBoardOpen((open) => !open)}
        >
          {taskBoardOpen ? "收起任务板" : "展开任务板"} · 看板 / 从目标创建
        </button>
        {!taskBoardOpen ? (
          <button
            type="button"
            className="btn linkish"
            style={{ marginLeft: "0.75rem" }}
            onClick={openGoalCompose}
            data-testid="agent-open-goal-compose"
            title="打开从目标创建 Plan"
          >
            从目标创建
          </button>
        ) : null}
        {taskBoardOpen ? (
          <div className="agent-task-board" data-testid="agent-task-board">
            <div className="pane" data-testid="quest-wb-compose" style={{ marginBottom: "1rem" }}>
              <div className="pane-title">
                <h2>从目标创建</h2>
                <span>{activePlan ? activePlan.status : "draft plan"}</span>
              </div>
              <div className="secret-form">
                <label className="wide-field">
                  Goal
                  <input
                    ref={goalInputRef}
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
                  title={
                    !goalText.trim()
                      ? "需要填写 Goal"
                      : fromGoalMut.isPending
                        ? "生成中…"
                        : "根据 Goal 生成 Plan（需确认）"
                  }
                  onClick={() => {
                    if (!goalText.trim()) return;
                    const snippet = goalText.trim().slice(0, 80);
                    const ok = window.confirm(`确认根据 Goal「${snippet}」生成 Plan？`);
                    if (!ok) return;
                    fromGoalMut.mutate();
                  }}
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
                  {activePlan.status !== "draft" ? (
                    <p className="muted-line" data-testid="quest-wb-plan-status-hint">
                      {activePlan.status === "rejected"
                        ? "此 Plan 已拒绝，不可再批准启动；可重新填写 Goal 生成新 Plan。"
                        : activePlan.status === "approved" || activePlan.status === "running"
                          ? `此 Plan 状态为 ${activePlan.status}，无需再次批准。`
                          : `此 Plan 状态为 ${activePlan.status}。`}
                    </p>
                  ) : null}
                  <pre className="code-block compact">
                    {JSON.stringify({ inputs: activePlan.inputs, steps: activePlan.steps }, null, 2)}
                  </pre>
                  {activePlan.status === "draft" ? (
                    <div className="row-actions">
                      <button
                        className="btn mini ok"
                        type="button"
                        disabled={approvePlanMut.isPending}
                        title={approvePlanMut.isPending ? "启动中…" : "批准并启动此 Plan（需确认）"}
                        onClick={() => {
                          const ok = window.confirm(
                            "确认批准 Plan 并启动 Run？将按所选场景写入工作区并执行。",
                          );
                          if (!ok) return;
                          approvePlanMut.mutate(activePlan.id);
                        }}
                        data-testid="quest-wb-approve"
                      >
                        批准并启动
                      </button>
                      <button
                        className="btn mini err"
                        type="button"
                        disabled={rejectPlanMut.isPending}
                        title={rejectPlanMut.isPending ? "拒绝中…" : "拒绝此 Plan（需确认）"}
                        onClick={() => {
                          const ok = window.confirm("确认拒绝此 Plan？");
                          if (!ok) return;
                          rejectPlanMut.mutate(activePlan.id);
                        }}
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
                          title={`打开 ${item.kind}「${item.title}」· ${item.status}`}
                        >
                          <strong>{item.title}</strong>
                          <div className="muted-line">
                            {item.kind} · {item.status}
                          </div>
                        </button>
                      </li>
                    ))}
                    {(boardQuery.data?.columns?.[col.key] ?? []).length === 0 ? (
                      <li className="muted-line">暂无</li>
                    ) : null}
                  </ul>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </div>
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
