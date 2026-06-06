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

export function CIPage() {
  const qc = useQueryClient();
  const [connectionId, setConnectionId] = useState("");
  const [runId, setRunId] = useState("");
  const [jobId, setJobId] = useState("");
  const [decisionStatus, setDecisionStatus] = useState("");

  const connectionsQuery = useQuery({ queryKey: ["repo-connections"], queryFn: listRepoConnections });
  const activeConnectionId = connectionId || connectionsQuery.data?.items[0]?.id || "";
  const runsQuery = useQuery({
    queryKey: ["ci-runs", activeConnectionId],
    queryFn: () => listCIRuns({ connectionId: activeConnectionId, limit: 50 }),
    enabled: Boolean(activeConnectionId),
  });
  const activeRunId = runId || runsQuery.data?.items[0]?.id || "";
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
    enabled: Boolean(activeConnectionId),
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

  const selectedDiagnosis = useMemo(() => diagnosesQuery.data?.items[0], [diagnosesQuery.data]);

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
              {!connectionsQuery.data?.items.length && <option value="">无连接</option>}
            </select>
          </label>
          <button className="btn icon-btn" onClick={() => syncRunsMut.mutate()} disabled={!activeConnectionId || syncRunsMut.isPending}>
            <RefreshCcw size={16} strokeWidth={1.8} />
            同步 runs
          </button>
          <button className="btn icon-btn" onClick={() => syncJobsMut.mutate()} disabled={!activeRunId || syncJobsMut.isPending}>
            <RefreshCcw size={16} strokeWidth={1.8} />
            同步 jobs
          </button>
        </div>
      </div>

      {(syncRunsMut.error || syncJobsMut.error || diagnoseMut.error || decideMut.error) && (
        <p className="error-text">
          {(syncRunsMut.error || syncJobsMut.error || diagnoseMut.error || decideMut.error as Error)?.message}
        </p>
      )}

      <div className="split ops-split">
        <div className="pane">
          <div className="pane-title">
            <h2>Workflow Runs</h2>
            <span>{runsQuery.data?.items.length ?? 0} 项</span>
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
            <span>{jobsQuery.data?.items.length ?? 0} 项</span>
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
              diagnoseMut.mutate({ logText: String(data.get("logText") || "") });
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
              <textarea name="logText" rows={6} placeholder={jobId ? "已选 job 时可留空，由后端拉取日志。" : "粘贴 CI 失败日志。"} />
            </label>
            <button className="btn primary icon-btn" type="submit" disabled={diagnoseMut.isPending || (!jobId && !activeConnectionId)}>
              <SearchCheck size={16} strokeWidth={1.8} />
              诊断失败
            </button>
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
          <span>{diagnosesQuery.data?.items.length ?? 0} 条</span>
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
                    <button className="btn mini ok" onClick={() => decideMut.mutate({ item, decision: "adopt" })} disabled={item.decisionStatus === "adopted"}>
                      <CheckCircle2 size={14} strokeWidth={1.8} />
                      采纳
                    </button>
                    <button className="btn mini err" onClick={() => decideMut.mutate({ item, decision: "dismiss" })} disabled={item.decisionStatus === "dismissed"}>
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
