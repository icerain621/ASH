import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
} from "@tanstack/react-table";
import { CheckCircle, Download, ExternalLink, GitBranch, Play, RefreshCw, RotateCcw, Square, Terminal } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import {
  approveRun,
  cancelRun,
  createRun,
  getRun,
  getRunArtifactAccess,
  getRunAgentTasks,
  getRunArtifacts,
  getRunCheckpointAccess,
  getRunCheckpoints,
  getRunQualityMetrics,
  getRunTimeline,
  getRunToolCalls,
  getRunWaterfall,
  listRuns,
  replayRun,
  resumeRun,
  type ArtifactAccessResponse,
  type ArtifactItem,
  type AgentTask,
  type Checkpoint,
  type CheckpointAccessResponse,
  type QualityMetric,
  type RunSummary,
  type TimelineItem,
  type ToolCall,
  type WaterfallSpan,
} from "@/modules/runs/api/runs.api";
import { getCurrentSpaceId } from "@/services/http/client";
import { useRunStream } from "@/services/sse/runStream";
import { fmtTime, shortId } from "@/shared/utils/format";

const runColumns: ColumnDef<RunSummary>[] = [
  {
    accessorKey: "runId",
    header: "运行 ID",
    cell: ({ row }) => <span title={row.original.runId}>{shortId(row.original.runId)}</span>,
  },
  {
    accessorKey: "status",
    header: "状态",
    cell: ({ row }) => (
      <span className={"status-pill " + statusTone(row.original.status)}>
        <span className="status-dot" />
        {statusLabel(row.original.status)}
      </span>
    ),
  },
  {
    id: "scenario",
    header: "场景",
    cell: ({ row }) => row.original.scenario?.name || "-",
  },
  {
    accessorKey: "spaceId",
    header: "空间",
    cell: ({ row }) => row.original.spaceId || "local",
  },
  {
    accessorKey: "startedAt",
    header: "开始时间",
    cell: ({ row }) => fmtTime(row.original.startedAt),
  },
];

function statusTone(status: string) {
  if (status === "completed" || status === "succeeded" || status === "finished") return "ok";
  if (status === "failed" || status === "cancelled" || status === "canceled") return "err";
  if (status === "running") return "live";
  return "idle";
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    cancelled: "已取消",
    canceled: "已取消",
    completed: "已完成",
    failed: "失败",
    finished: "已完成",
    pending: "等待中",
    running: "运行中",
    succeeded: "成功",
    waiting_approval: "待审批",
  };
  return labels[status] || status;
}

function riskLabel(risk: string) {
  const labels: Record<string, string> = {
    high: "高",
    low: "低",
    medium: "中",
    none: "无",
  };
  return labels[risk] || risk;
}

function metricLabel(name: string) {
  const labels: Record<string, string> = {
    agent_failure_rate: "Agent 失败率",
    agent_tasks_total: "Agent 任务数",
    artifact_quality_failed_total: "产物质量失败",
    artifact_quality_passed: "产物质量通过",
    artifacts_total: "产物数",
    citation_bound_total: "引用命中",
    citation_hit_rate: "引用命中率",
    citation_missing_total: "缺失引用",
    model_cost_micros_total: "模型成本",
    run_recovered: "恢复执行",
    steps_total: "步骤数",
    tool_calls_total: "工具调用数",
    tool_failure_rate: "工具失败率",
  };
  return labels[name] || name;
}

function metricValue(metric: QualityMetric) {
  if (metric.unit === "ratio") return `${(metric.value * 100).toFixed(1)}%`;
  if (metric.unit === "bool") return metric.value > 0 ? "是" : "否";
  if (Number.isInteger(metric.value)) return String(metric.value);
  return metric.value.toFixed(3);
}

function bytesLabel(value?: number) {
  if (!value || value <= 0) return "-";
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

function artifactProducerLabel(artifact: ArtifactItem) {
  const producer = artifact.producer;
  if (!producer) return "-";
  if (producer.role && producer.stepId) return `${producer.role} / ${producer.stepId}`;
  return producer.stepId || producer.role || "-";
}

function artifactProducerTitle(artifact: ArtifactItem) {
  const producer = artifact.producer;
  if (!producer) return "";
  const parts = [
    producer.stepId ? `step: ${producer.stepId}` : "",
    producer.role ? `role: ${producer.role}` : "",
    producer.agentTaskId ? `agent: ${producer.agentTaskId}` : "",
    producer.eventRange ? `events: ${producer.eventRange}` : "",
  ].filter(Boolean);
  return parts.join("\n");
}

function artifactItems(data: { artifacts?: ArtifactItem[]; manifest?: { artifacts?: ArtifactItem[] } } | undefined) {
  return data?.artifacts ?? data?.manifest?.artifacts ?? [];
}

function durationLabel(ms?: number) {
  if (!ms || ms <= 0) return "-";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function waterfallBounds(spans: WaterfallSpan[]) {
  const starts = spans.map((span) => span.startTs || 0).filter((value) => value > 0);
  const ends = spans.map((span) => span.endTs || span.startTs || 0).filter((value) => value > 0);
  const start = starts.length ? Math.min(...starts) : 0;
  const end = ends.length ? Math.max(...ends) : start;
  return { start, total: Math.max(1, end - start) };
}

function spanOffset(span: WaterfallSpan, start: number, total: number) {
  if (!span.startTs || span.startTs < start) return 0;
  return Math.min(96, Math.max(0, ((span.startTs - start) / total) * 100));
}

function spanWidth(span: WaterfallSpan, total: number) {
  const width = ((span.durationMs || 1) / total) * 100;
  return Math.min(100, Math.max(3, width));
}

function spanTone(status: string) {
  if (status === "success" || status === "finished" || status === "routed") return "ok";
  if (status === "failed") return "err";
  if (status === "running") return "live";
  return "idle";
}

function isTerminalStatus(status?: string) {
  return status === "finished" || status === "failed" || status === "canceled" || status === "cancelled";
}

function payloadRecord(payload: unknown) {
  return payload && typeof payload === "object" && !Array.isArray(payload)
    ? (payload as Record<string, unknown>)
    : null;
}

function waitingGate(items: TimelineItem[] | undefined) {
  const item = [...(items ?? [])].reverse().find((entry) => entry.type === "gate.waiting_approval");
  const payload = payloadRecord(item?.payload);
  if (!item || !payload) return null;
  return {
    gate: typeof payload.gate === "string" ? payload.gate : "human",
    reason: typeof payload.reason === "string" ? payload.reason : "",
    stepId: typeof payload.stepId === "string" ? payload.stepId : "",
  };
}

export function RunsPage() {
  const qc = useQueryClient();
  const activeSpaceId = getCurrentSpaceId();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [artifactAccess, setArtifactAccess] = useState<ArtifactAccessResponse | null>(null);
  const [checkpointAccess, setCheckpointAccess] = useState<CheckpointAccessResponse | null>(null);
  const [replayMode, setReplayMode] = useState<"exact" | "latest_memory">("exact");
  const [actionMessage, setActionMessage] = useState<string | null>(null);

  const runsQuery = useQuery({
    queryKey: ["runs", activeSpaceId],
    queryFn: () => listRuns(30),
  });

  const detailQuery = useQuery({
    queryKey: ["runs", selectedId],
    queryFn: () => getRun(selectedId!),
    enabled: !!selectedId,
  });

  const artifactsQuery = useQuery({
    queryKey: ["runs", selectedId, "artifacts"],
    queryFn: () => getRunArtifacts(selectedId!),
    enabled: !!selectedId,
  });

  const checkpointsQuery = useQuery({
    queryKey: ["runs", selectedId, "checkpoints"],
    queryFn: () => getRunCheckpoints(selectedId!),
    enabled: !!selectedId,
  });

  const timelineQuery = useQuery({
    queryKey: ["runs", selectedId, "timeline"],
    queryFn: () => getRunTimeline(selectedId!),
    enabled: !!selectedId,
  });

  const toolsQuery = useQuery({
    queryKey: ["runs", selectedId, "tool-calls"],
    queryFn: () => getRunToolCalls(selectedId!),
    enabled: !!selectedId,
  });

  const agentsQuery = useQuery({
    queryKey: ["runs", selectedId, "agent-tasks"],
    queryFn: () => getRunAgentTasks(selectedId!),
    enabled: !!selectedId,
  });

  const qualityQuery = useQuery({
    queryKey: ["runs", selectedId, "quality-metrics"],
    queryFn: () => getRunQualityMetrics(selectedId!),
    enabled: !!selectedId,
  });

  const waterfallQuery = useQuery({
    queryKey: ["runs", selectedId, "waterfall"],
    queryFn: () => getRunWaterfall(selectedId!),
    enabled: !!selectedId,
  });

  const streamLines = useRunStream(selectedId);
  const artifacts = useMemo(() => artifactItems(artifactsQuery.data), [artifactsQuery.data]);
  const checkpoints = checkpointsQuery.data?.items ?? [];

  useEffect(() => {
    setArtifactAccess(null);
    setCheckpointAccess(null);
    setActionMessage(null);
  }, [selectedId]);

  const refreshRunQueries = async (runId: string | null) => {
    await qc.invalidateQueries({ queryKey: ["runs"] });
    if (!runId) return;
    await Promise.all([
      qc.invalidateQueries({ queryKey: ["runs", runId] }),
      qc.invalidateQueries({ queryKey: ["runs", runId, "artifacts"] }),
      qc.invalidateQueries({ queryKey: ["runs", runId, "checkpoints"] }),
      qc.invalidateQueries({ queryKey: ["runs", runId, "timeline"] }),
      qc.invalidateQueries({ queryKey: ["runs", runId, "tool-calls"] }),
      qc.invalidateQueries({ queryKey: ["runs", runId, "agent-tasks"] }),
      qc.invalidateQueries({ queryKey: ["runs", runId, "quality-metrics"] }),
      qc.invalidateQueries({ queryKey: ["runs", runId, "waterfall"] }),
    ]);
  };

  const createMut = useMutation({
    mutationFn: () =>
      createRun({
        scenario: { name: "feature_delivery", scenarioVersion: "1.0.0" },
        inputs: {
          issueOrSpec: "UI demo run " + new Date().toISOString(),
          repoRoot: ".",
        },
      }),
    onSuccess: async (res) => {
      await qc.invalidateQueries({ queryKey: ["runs"] });
      setSelectedId(res.runId);
    },
  });

  const resumeMut = useMutation({
    mutationFn: () => resumeRun(selectedId!),
    onSuccess: async () => {
      setActionMessage("继续执行已提交");
      await refreshRunQueries(selectedId);
    },
  });

  const replayMut = useMutation({
    mutationFn: () => replayRun(selectedId!, { mode: replayMode }),
    onSuccess: async (res) => {
      setActionMessage(`已创建重放运行 ${shortId(res.runId)}`);
      await refreshRunQueries(selectedId);
      setSelectedId(res.runId);
    },
  });

  const cancelMut = useMutation({
    mutationFn: () => cancelRun(selectedId!),
    onSuccess: async () => {
      setActionMessage("取消请求已提交");
      await refreshRunQueries(selectedId);
    },
  });

  const approveMut = useMutation({
    mutationFn: () =>
      approveRun(selectedId!, {
        actorId: "console",
        reason: "Approved from ASH Console",
      }),
    onSuccess: async () => {
      setActionMessage("审批已通过，运行继续推进");
      await refreshRunQueries(selectedId);
    },
  });

  const artifactAccessMut = useMutation({
    mutationFn: (artifactName: string) => getRunArtifactAccess(selectedId!, artifactName),
    onSuccess: (res) => {
      setArtifactAccess(res);
    },
  });

  const checkpointAccessMut = useMutation({
    mutationFn: (checkpointId: string) => getRunCheckpointAccess(selectedId!, checkpointId),
    onSuccess: (res) => {
      setCheckpointAccess(res);
    },
  });

  const items = runsQuery.data?.items ?? [];
  const table = useReactTable({
    data: items,
    columns: runColumns,
    getCoreRowModel: getCoreRowModel(),
  });

  const selected = detailQuery.data;
  const err =
    runsQuery.error?.message ||
    createMut.error?.message ||
    resumeMut.error?.message ||
    replayMut.error?.message ||
    cancelMut.error?.message ||
    approveMut.error?.message ||
    artifactAccessMut.error?.message ||
    checkpointAccessMut.error?.message;
  const gate = waitingGate(timelineQuery.data?.items);
  const canCancel = selectedId && selected && !isTerminalStatus(selected.status);
  const canApprove = selectedId && selected?.status === "waiting_approval";
  const waterfallSpans = waterfallQuery.data?.spans ?? [];
  const waterfall = waterfallBounds(waterfallSpans);

  return (
    <section className="panel active">
      <div className="page-kicker">
        <Terminal size={17} strokeWidth={1.8} />
        智能体运行时
      </div>
      <div className="page-heading">
        <div>
          <h1>运行</h1>
          <p>查看场景执行、实时事件流和生成产物。</p>
          <span className="scope-badge">Space: {activeSpaceId}</span>
        </div>
        <div className="toolbar">
          <button className="btn icon-btn" onClick={() => runsQuery.refetch()} disabled={runsQuery.isFetching}>
            <RefreshCw size={16} strokeWidth={1.8} />
            刷新
          </button>
          <button className="btn primary icon-btn" onClick={() => createMut.mutate()} disabled={createMut.isPending}>
            <Play size={16} strokeWidth={1.8} />
            新建运行
          </button>
        </div>
      </div>
      {err && <p className="error-text">{err}</p>}
      <div className="split runs-grid">
        <div className="pane">
          <div className="pane-title">
            <h2>执行队列</h2>
            <span>{items.length} 条运行</span>
          </div>
          <table className="table">
            <thead>
              {table.getHeaderGroups().map((headerGroup) => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <th key={header.id}>
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody>
              {table.getRowModel().rows.map((row) => (
                <tr
                  key={row.original.runId}
                  className={row.original.runId === selectedId ? "selected" : ""}
                  onClick={() => setSelectedId(row.original.runId)}
                >
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))}
              {!table.getRowModel().rows.length && (
                <tr className="empty-row">
                  <td colSpan={runColumns.length}>暂无运行记录。</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className="pane detail-pane">
          <div className="pane-title">
            <h2>运行详情</h2>
            <span>{selectedId ? shortId(selectedId) : "未选择"}</span>
          </div>
          <pre className="code-block">
            {selected ? JSON.stringify(selected, null, 2) : "选择一条运行记录"}
          </pre>
          <div className="pane-title subhead">
            <h3>门禁与控制</h3>
            <span>{selected ? statusLabel(selected.status) : "未选择"}</span>
          </div>
          <div className="run-control">
            <div className="run-control-status">
              {selected ? (
                <span className={"status-pill " + statusTone(selected.status)}>
                  <span className="status-dot" />
                  {statusLabel(selected.status)}
                </span>
              ) : (
                <span className="muted">选择运行后可操作。</span>
              )}
              {gate && (
                <div className="gate-summary">
                  <strong>{gate.gate === "citation" ? "引用门禁" : "人工门禁"}</strong>
                  <span>{gate.stepId || "-"}</span>
                  {gate.reason && <p>{gate.reason}</p>}
                </div>
              )}
              {actionMessage && <p className="action-message">{actionMessage}</p>}
            </div>
            <div className="run-control-actions">
              {canApprove && (
                <button className="btn ok icon-btn" onClick={() => approveMut.mutate()} disabled={approveMut.isPending}>
                  <CheckCircle size={16} strokeWidth={1.8} />
                  通过
                </button>
              )}
              {selectedId && selected?.status === "failed" && (
                <button className="btn icon-btn" onClick={() => resumeMut.mutate()} disabled={resumeMut.isPending}>
                  <RotateCcw size={16} strokeWidth={1.8} />
                  继续
                </button>
              )}
              {selectedId && (
                <>
                  <select
                    className="control-select"
                    value={replayMode}
                    onChange={(event) => setReplayMode(event.target.value as "exact" | "latest_memory")}
                    aria-label="重放模式"
                  >
                    <option value="exact">exact</option>
                    <option value="latest_memory">latest_memory</option>
                  </select>
                  <button className="btn icon-btn" onClick={() => replayMut.mutate()} disabled={replayMut.isPending}>
                    <GitBranch size={16} strokeWidth={1.8} />
                    重放
                  </button>
                </>
              )}
              {canCancel && (
                <button className="btn err icon-btn" onClick={() => cancelMut.mutate()} disabled={cancelMut.isPending}>
                  <Square size={16} strokeWidth={1.8} />
                  取消
                </button>
              )}
            </div>
          </div>
          <div className="pane-title subhead">
            <h3>质量指标</h3>
            <span>{qualityQuery.data?.items.length ?? 0} 项</span>
          </div>
          <table className="table compact">
            <thead>
              <tr>
                <th>指标</th>
                <th>值</th>
                <th>单位</th>
              </tr>
            </thead>
            <tbody>
              {(qualityQuery.data?.items ?? []).map((metric: QualityMetric) => (
                <tr key={metric.id}>
                  <td>{metricLabel(metric.name)}</td>
                  <td>{metricValue(metric)}</td>
                  <td>{metric.unit || "-"}</td>
                </tr>
              ))}
              {selectedId && !(qualityQuery.data?.items.length) && (
                <tr className="empty-row">
                  <td colSpan={3}>暂无质量指标。</td>
                </tr>
              )}
            </tbody>
          </table>
          <div className="pane-title subhead">
            <h3>事件流 (SSE)</h3>
            <span>{streamLines.length} 条事件</span>
          </div>
          <div className="event-log">
            {streamLines.map((line) => (
              <div
                key={line.id}
                className={
                  "event-line" +
                  (line.type.startsWith("memory.") ? " memory" : "") +
                  (line.type.includes("failed") ? " error" : "")
                }
              >
                <span className="type">{line.type}</span> {line.payload}
              </div>
            ))}
            {!streamLines.length && <div className="event-line muted">等待事件流。</div>}
          </div>
          <div className="pane-title subhead">
            <h3>时间线</h3>
            <span>{timelineQuery.data?.items.length ?? 0} 条记录</span>
          </div>
          <div className="event-log">
            {(timelineQuery.data?.items ?? []).slice(-32).map((item: TimelineItem) => (
              <div
                key={`${item.seq}-${item.type}`}
                className={"event-line" + (item.severity === "error" ? " error" : "")}
              >
                <span className="type">{item.type}</span>{" "}
                <span className="muted">#{item.seq}</span>{" "}
                {item.payload ? JSON.stringify(item.payload) : ""}
              </div>
            ))}
            {selectedId && !(timelineQuery.data?.items.length) && <div className="event-line muted">暂无时间线记录。</div>}
          </div>
          <div className="pane-title subhead">
            <h3>瀑布视图</h3>
            <span>{waterfallSpans.length} 个 span</span>
          </div>
          <div className="waterfall">
            {waterfallSpans.map((span) => (
              <div key={span.id} className="waterfall-row">
                <div className="waterfall-meta">
                  <span className={"span-type " + span.type}>{span.type}</span>
                  <strong title={span.name}>{span.name}</strong>
                  <span className="muted">{statusLabel(span.status)}</span>
                  <span className="muted">{durationLabel(span.durationMs)}</span>
                </div>
                <div className="waterfall-track" aria-hidden="true">
                  <span
                    className={"waterfall-bar " + spanTone(span.status)}
                    style={{
                      marginLeft: `${spanOffset(span, waterfall.start, waterfall.total)}%`,
                      width: `${spanWidth(span, waterfall.total)}%`,
                    }}
                  />
                </div>
              </div>
            ))}
            {selectedId && !waterfallSpans.length && (
              <div className="waterfall-empty muted">暂无瀑布数据。</div>
            )}
            {!!(waterfallQuery.data?.failures?.length) && (
              <div className="failure-list">
                {waterfallQuery.data.failures.map((failure) => (
                  <div key={`${failure.type}-${failure.ref}-${failure.code || failure.message}`} className="failure-item">
                    <span>{failure.type}</span>
                    <strong>{failure.ref}</strong>
                    <em>{failure.code || failure.message || "failed"}</em>
                  </div>
                ))}
              </div>
            )}
          </div>
          <div className="pane-title subhead">
            <h3>智能体任务</h3>
            <span>{agentsQuery.data?.items.length ?? 0} 个任务</span>
          </div>
          <table className="table compact">
            <thead>
              <tr>
                <th>步骤</th>
                <th>适配器</th>
                <th>状态</th>
                <th>ExecGo</th>
              </tr>
            </thead>
            <tbody>
              {(agentsQuery.data?.items ?? []).map((task: AgentTask) => (
                <tr key={task.id}>
                  <td>{task.stepId || "-"}</td>
                  <td>{task.adapter}</td>
                  <td>{statusLabel(task.status)}</td>
                  <td title={task.execGoTaskId}>{task.execGoTaskId ? shortId(task.execGoTaskId) : "-"}</td>
                </tr>
              ))}
              {selectedId && !(agentsQuery.data?.items.length) && (
                <tr className="empty-row">
                  <td colSpan={4}>暂无智能体任务。</td>
                </tr>
              )}
            </tbody>
          </table>
          <div className="pane-title subhead">
            <h3>工具调用</h3>
            <span>{toolsQuery.data?.items.length ?? 0} 次调用</span>
          </div>
          <table className="table compact">
            <thead>
              <tr>
                <th>步骤</th>
                <th>工具</th>
                <th>风险</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {(toolsQuery.data?.items ?? []).map((call: ToolCall) => (
                <tr key={call.id}>
                  <td>{call.stepId || "-"}</td>
                  <td>{call.tool}</td>
                  <td>{riskLabel(String(call.risk))}</td>
                  <td>{statusLabel(call.status)}</td>
                </tr>
              ))}
              {selectedId && !(toolsQuery.data?.items.length) && (
                <tr className="empty-row">
                  <td colSpan={4}>暂无工具调用。</td>
                </tr>
              )}
            </tbody>
          </table>
          <div className="pane-title subhead">
            <h3>产物</h3>
            <span>{artifactsQuery.isFetching ? "加载中" : `${artifacts.length} 个产物`}</span>
          </div>
          <table className="table compact artifacts-table">
            <thead>
              <tr>
                <th>名称</th>
                <th>类型</th>
                <th>来源</th>
                <th>大小</th>
                <th>Digest</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {artifacts.map((artifact) => (
                <tr key={`${artifact.type}-${artifact.name}`}>
                  <td title={artifact.uri}>{artifact.name}</td>
                  <td>{artifact.type}</td>
                  <td title={artifactProducerTitle(artifact)}>{artifactProducerLabel(artifact)}</td>
                  <td>{bytesLabel(artifact.sizeBytes)}</td>
                  <td title={artifact.digest}>{artifact.digest ? shortId(artifact.digest) : "-"}</td>
                  <td>
                    <button
                      className="btn icon-btn mini"
                      onClick={() => artifactAccessMut.mutate(artifact.name)}
                      disabled={!selectedId || artifactAccessMut.isPending}
                      title="获取访问链接"
                    >
                      <Download size={14} strokeWidth={1.8} />
                      链接
                    </button>
                  </td>
                </tr>
              ))}
              {selectedId && !artifacts.length && (
                <tr className="empty-row">
                  <td colSpan={6}>暂无产物。</td>
                </tr>
              )}
            </tbody>
          </table>
          {artifactAccess && (
            <div className="artifact-access">
              <span title={artifactAccess.name}>{artifactAccess.name}</span>
              <code title={artifactAccess.signedUrl}>{artifactAccess.signedUrl}</code>
              {artifactAccess.signedUrl.startsWith("http") && (
                <a className="btn icon-btn mini" href={artifactAccess.signedUrl} target="_blank" rel="noreferrer">
                  <ExternalLink size={14} strokeWidth={1.8} />
                  打开
                </a>
              )}
            </div>
          )}
          <div className="pane-title subhead">
            <h3>检查点</h3>
            <span>{checkpointsQuery.isFetching ? "加载中" : `${checkpoints.length} 个快照`}</span>
          </div>
          <table className="table compact artifacts-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>步骤</th>
                <th>策略</th>
                <th>大小</th>
                <th>Digest</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {checkpoints.map((checkpoint: Checkpoint) => (
                <tr key={checkpoint.id}>
                  <td title={checkpoint.id}>{shortId(checkpoint.id)}</td>
                  <td>{checkpoint.stepId || "-"}</td>
                  <td>{checkpoint.strategy || "-"}</td>
                  <td>{bytesLabel(checkpoint.sizeBytes)}</td>
                  <td title={checkpoint.snapshotDigest}>
                    {checkpoint.snapshotDigest ? shortId(checkpoint.snapshotDigest) : "-"}
                  </td>
                  <td>
                    <button
                      className="btn icon-btn mini"
                      onClick={() => checkpointAccessMut.mutate(checkpoint.id)}
                      disabled={!selectedId || checkpointAccessMut.isPending}
                      title="获取检查点访问链接"
                    >
                      <Download size={14} strokeWidth={1.8} />
                      链接
                    </button>
                  </td>
                </tr>
              ))}
              {selectedId && !checkpoints.length && (
                <tr className="empty-row">
                  <td colSpan={6}>暂无检查点。</td>
                </tr>
              )}
            </tbody>
          </table>
          {checkpointAccess && (
            <div className="artifact-access">
              <span title={checkpointAccess.checkpointId}>{shortId(checkpointAccess.checkpointId)}</span>
              <code title={checkpointAccess.signedUrl}>{checkpointAccess.signedUrl}</code>
              {checkpointAccess.signedUrl.startsWith("http") && (
                <a className="btn icon-btn mini" href={checkpointAccess.signedUrl} target="_blank" rel="noreferrer">
                  <ExternalLink size={14} strokeWidth={1.8} />
                  打开
                </a>
              )}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
