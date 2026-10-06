import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { GitBranch, RefreshCcw, SearchCheck, XCircle, CheckCircle2 } from "lucide-react";
import { useMemo, useState } from "react";
import {
  adoptCIDiagnosis,
  diagnoseCIFailure,
  dismissCIDiagnosis,
  listCIDiagnoses,
  listCIJobs,
  listCIRuns,
  listRepoConnections,
  type CIDiagnosis,
} from "@/modules/closure/api/closure.api";
import { getReadyz } from "@/modules/health/api/health.api";
import { ApiError } from "@/services/http/client";

function formatCIError(err: unknown): string | null {
  if (!err) return null;
  if (err instanceof ApiError) {
    if (err.code === "CI_PROVIDER_UNAVAILABLE" || err.code === "CI_RUN_LIST_FAILED" || err.code === "CI_JOB_LIST_FAILED") {
      return `${err.code}: ${err.message}`;
    }
    return err.message || err.code;
  }
  if (err instanceof Error) return err.message;
  return String(err);
}

export function CIPage() {
  const qc = useQueryClient();
  const [connectionId, setConnectionId] = useState("");
  const [runId, setRunId] = useState("");
  const [jobId, setJobId] = useState("");
  const [decisionStatus, setDecisionStatus] = useState("");
  const [logText, setLogText] = useState("");

  const connectionsQuery = useQuery({ queryKey: ["repo-connections"], queryFn: listRepoConnections });
  const readyzQuery = useQuery({ queryKey: ["readyz"], queryFn: getReadyz, staleTime: 60_000 });
  const ciFixtureMode = (readyzQuery.data?.liveGateHints ?? []).some((h) => h.includes("ASH_CI_FIXTURE"));
  const activeConnectionId = connectionId || connectionsQuery.data?.items?.[0]?.id || "";
  const runsQuery = useQuery({
    queryKey: ["ci-runs", activeConnectionId],
    queryFn: () => listCIRuns({ connectionId: activeConnectionId, limit: 50 }),
    enabled: Boolean(activeConnectionId),
  });
  const activeRunId = runId || runsQuery.data?.items?.[0]?.id || "";
  const jobsQuery = useQuery({
    queryKey: ["ci-jobs", activeRunId],
    queryFn: () => listCIJobs({ runId: activeRunId, limit: 50 }),
    enabled: Boolean(activeRunId),
  });
  const diagnosesQuery = useQuery({
    queryKey: ["ci-diagnoses", activeConnectionId, activeRunId, jobId, decisionStatus],
    queryFn: () =>
      listCIDiagnoses({
        connectionId: activeConnectionId,
        runId: activeRunId,
        jobId,
        decisionStatus,
        limit: 50,
      }),
  });

  const syncRunsMut = useMutation({
    mutationFn: () => listCIRuns({ connectionId: activeConnectionId, sync: true, limit: 50 }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ci-runs", activeConnectionId] }),
  });
  const syncJobsMut = useMutation({
    mutationFn: () => listCIJobs({ runId: activeRunId, sync: true, limit: 50 }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ci-jobs", activeRunId] }),
  });
  const diagnoseMut = useMutation({
    mutationFn: (body: { logText?: string }) =>
      diagnoseCIFailure({
        connectionId: activeConnectionId,
        runId: activeRunId,
        jobId,
        logText: body.logText,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ci-diagnoses"] }),
  });
  const decideMut = useMutation({
    mutationFn: ({ item, decision }: { item: CIDiagnosis; decision: "adopt" | "dismiss" }) =>
      decision === "adopt"
        ? adoptCIDiagnosis(item.id, "诊断建议已采纳")
        : dismissCIDiagnosis(item.id, "当前不采纳该诊断"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ci-diagnoses"] });
      qc.invalidateQueries({ queryKey: ["metrics"] });
    },
  });

  const selectedDiagnosis = useMemo(
    () => diagnoseMut.data ?? diagnosesQuery.data?.items?.[0],
    [diagnoseMut.data, diagnosesQuery.data],
  );

  return (
    <section className="panel active">
      <div className="page-kicker">
        <GitBranch size={17} strokeWidth={1.8} />
        CI 诊断
      </div>
      <div className="page-heading">
        <div>
          <h1>CI 诊断控制台</h1>
          <p>同步 GitHub Actions 运行，拉取 job 日志诊断失败，并记录采纳或驳回证据。</p>
          {ciFixtureMode && (
            <p className="muted-line">
              Worker 已启用 <code>ASH_CI_FIXTURE=1</code>：sync 使用内置 fixture 数据，无需真实 GitHub token。
            </p>
          )}
        </div>
        <div className="toolbar metrics-toolbar">
          <label className="scenario-picker">
            Repo
            <select value={activeConnectionId} onChange={(e) => setConnectionId(e.target.value)}>
              {(connectionsQuery.data?.items ?? []).map((conn) => (
                <option key={conn.id} value={conn.id}>
                  {conn.owner}/{conn.repo}
                </option>
              ))}
              {!connectionsQuery.data?.items?.length && <option value="">无连接</option>}
            </select>
          </label>
          <button
            className="btn icon-btn"
            onClick={() => {
              const ok = window.confirm("确认从远程同步 Workflow Runs？");
              if (!ok) return;
              syncRunsMut.mutate();
            }}
            disabled={!activeConnectionId || syncRunsMut.isPending}
            title={
              !activeConnectionId
                ? "需要先选择 Repo 连接"
                : syncRunsMut.isPending
                  ? "同步中…"
                  : "同步 runs（需确认）"
            }
            data-testid="ci-sync-runs"
          >
            <RefreshCcw size={16} strokeWidth={1.8} />
            同步 runs
          </button>
          <button
            className="btn icon-btn"
            onClick={() => {
              const ok = window.confirm("确认从远程同步 Workflow Jobs？");
              if (!ok) return;
              syncJobsMut.mutate();
            }}
            disabled={!activeRunId || syncJobsMut.isPending}
            title={
              !activeRunId
                ? "需要先选择 Workflow Run"
                : syncJobsMut.isPending
                  ? "同步中…"
                  : "同步 jobs（需确认）"
            }
            data-testid="ci-sync-jobs"
          >
            <RefreshCcw size={16} strokeWidth={1.8} />
            同步 jobs
          </button>
        </div>
      </div>

      {(syncRunsMut.error || syncJobsMut.error || diagnoseMut.error || decideMut.error) && (
        <p className="error-text">
          {formatCIError(syncRunsMut.error) ||
            formatCIError(syncJobsMut.error) ||
            formatCIError(diagnoseMut.error) ||
            formatCIError(decideMut.error)}
        </p>
      )}

      <div className="split ops-split">
        <div className="pane">
          <div className="pane-title">
            <h2>Workflow Runs</h2>
            <span>{runsQuery.data?.items?.length ?? 0} 项</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>workflow</th>
                <th>status</th>
                <th>branch</th>
              </tr>
            </thead>
            <tbody>
              {(runsQuery.data?.items ?? []).length === 0 && (
                <tr className="empty-row">
                  <td colSpan={3}>暂无 CI run，可先同步。</td>
                </tr>
              )}
              {(runsQuery.data?.items ?? []).map((run) => (
                <tr key={run.id} className={run.id === activeRunId ? "selected" : ""} onClick={() => setRunId(run.id)}>
                  <td>{run.workflow || run.providerRunId}</td>
                  <td>
                    <StatusPill value={run.conclusion || run.status} />
                  </td>
                  <td>{run.branch || "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="pane">
          <div className="pane-title">
            <h2>Jobs</h2>
            <span>{jobsQuery.data?.items?.length ?? 0} 项</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>job</th>
                <th>status</th>
                <th>digest</th>
              </tr>
            </thead>
            <tbody>
              {(jobsQuery.data?.items ?? []).length === 0 && (
                <tr className="empty-row">
                  <td colSpan={3}>选择 run 后同步 jobs。</td>
                </tr>
              )}
              {(jobsQuery.data?.items ?? []).map((job) => (
                <tr key={job.id} className={job.id === jobId ? "selected" : ""} onClick={() => setJobId(job.id)}>
                  <td>{job.name || job.providerJobId}</td>
                  <td>
                    <StatusPill value={job.conclusion || job.status} />
                  </td>
                  <td>{job.logDigest ? "已记录" : "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="split ops-split">
        <div className="pane">
          <div className="pane-title">
            <h2>发起诊断</h2>
            <span>{jobId ? "job logs" : "manual log"}</span>
          </div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              const data = new FormData(event.currentTarget);
              const text = String(data.get("logText") || "");
              const ok = window.confirm(
                text.trim()
                  ? "确认根据粘贴日志发起 CI 失败诊断？"
                  : "确认根据所选连接/job 发起 CI 失败诊断？",
              );
              if (!ok) return;
              diagnoseMut.mutate({ logText: text });
            }}
          >
            <label>
              Decision
              <select value={decisionStatus} onChange={(e) => setDecisionStatus(e.target.value)}>
                <option value="">全部</option>
                <option value="pending">pending</option>
                <option value="adopted">adopted</option>
                <option value="dismissed">dismissed</option>
              </select>
            </label>
            <label className="wide-field">
              Log Text
              <textarea
                name="logText"
                rows={6}
                value={logText}
                onChange={(e) => setLogText(e.target.value)}
                placeholder={jobId ? "已选 job 时可留空，由后端拉取日志。" : "粘贴 CI 失败日志。"}
              />
            </label>
            <div className="row-actions">
              <button
                className="btn primary icon-btn"
                type="submit"
                data-testid="ci-diagnose-log"
                disabled={diagnoseMut.isPending || (!jobId && !activeConnectionId && !logText.trim())}
                title={
                  diagnoseMut.isPending
                    ? "诊断进行中"
                    : !jobId && !activeConnectionId && !logText.trim()
                      ? "需要粘贴失败日志，或选择 Repo 连接 / job"
                      : "诊断失败（需确认）"
                }
              >
                <SearchCheck size={16} strokeWidth={1.8} />
                诊断失败
              </button>
              <button
                className="btn icon-btn"
                type="button"
                data-testid="ci-diagnose-job"
                disabled={diagnoseMut.isPending || !jobId}
                title={
                  !jobId
                    ? "需要先选择 Workflow Job"
                    : diagnoseMut.isPending
                      ? "诊断进行中"
                      : "诊断选中 job（需确认）"
                }
                onClick={() => {
                  const ok = window.confirm("确认诊断所选 Workflow Job？");
                  if (!ok) return;
                  diagnoseMut.mutate({});
                }}
              >
                <SearchCheck size={16} strokeWidth={1.8} />
                诊断选中 job
              </button>
            </div>
          </form>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>最新结果</h2>
            <span>{selectedDiagnosis?.decisionStatus ?? "idle"}</span>
          </div>
          <pre className="code-block tall">
            {selectedDiagnosis ? JSON.stringify(selectedDiagnosis, null, 2) : "暂无诊断记录。"}
          </pre>
        </div>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>诊断历史</h2>
          <span>{diagnosesQuery.data?.items?.length ?? 0} 条</span>
        </div>
        <table className="table">
          <thead>
            <tr>
              <th>root cause</th>
              <th>confidence</th>
              <th>decision</th>
              <th>actions</th>
            </tr>
          </thead>
          <tbody>
            {(diagnosesQuery.data?.items ?? []).length === 0 && (
              <tr className="empty-row">
                <td colSpan={4}>暂无诊断历史。</td>
              </tr>
            )}
            {(diagnosesQuery.data?.items ?? []).map((item) => (
              <tr key={item.id}>
                <td>{item.rootCause}</td>
                <td>{Math.round(item.confidence * 100)}%</td>
                <td>
                  <StatusPill value={item.decisionStatus} />
                </td>
                <td>
                  <div className="row-actions">
                    <button
                      className="btn mini ok"
                      data-testid={`ci-diagnosis-adopt-${item.id}`}
                      onClick={() => {
                        const ok = window.confirm(
                          `确认采纳诊断「${item.rootCause || item.id}」？`,
                        );
                        if (!ok) return;
                        decideMut.mutate({ item, decision: "adopt" });
                      }}
                      disabled={item.decisionStatus === "adopted" || decideMut.isPending}
                      title={
                        item.decisionStatus === "adopted"
                          ? "已采纳"
                          : decideMut.isPending
                            ? "处理中…"
                            : "采纳诊断（需确认）"
                      }
                    >
                      <CheckCircle2 size={14} strokeWidth={1.8} />
                      采纳
                    </button>
                    <button
                      className="btn mini err"
                      data-testid={`ci-diagnosis-dismiss-${item.id}`}
                      onClick={() => {
                        const ok = window.confirm(
                          `确认驳回诊断「${item.rootCause || item.id}」？`,
                        );
                        if (!ok) return;
                        decideMut.mutate({ item, decision: "dismiss" });
                      }}
                      disabled={item.decisionStatus === "dismissed" || decideMut.isPending}
                      title={
                        item.decisionStatus === "dismissed"
                          ? "已驳回"
                          : decideMut.isPending
                            ? "处理中…"
                            : "驳回诊断（需确认）"
                      }
                    >
                      <XCircle size={14} strokeWidth={1.8} />
                      驳回
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function StatusPill({ value }: { value: string }) {
  const lowered = value.toLowerCase();
  const tone = ["success", "adopted", "resolved", "pass", "ok"].includes(lowered)
    ? "ok"
    : ["failure", "failed", "dismissed", "block", "error"].includes(lowered)
      ? "err"
      : "idle";
  return (
    <span className={`status-pill ${tone}`}>
      <span className="status-dot" />
      {value || "unknown"}
    </span>
  );
}
