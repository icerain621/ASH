import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ClipboardCheck, PlayCircle, RefreshCcw, RotateCcw, Send } from "lucide-react";
import { useMemo, useState } from "react";
import {
  createRelease,
  createRollbackDrill,
  evaluateReleaseGate,
  getReleaseChecklist,
  listReleases,
  patchReleaseChecklist,
  type ReleaseChecklistItem,
} from "@/modules/closure/api/closure.api";

export function ReleasesPage() {
  const qc = useQueryClient();
  const [releaseId, setReleaseId] = useState("");
  const releasesQuery = useQuery({ queryKey: ["releases"], queryFn: () => listReleases({ limit: 50 }) });
  const activeReleaseId = releaseId || releasesQuery.data?.items[0]?.id || "";
  const checklistQuery = useQuery({
    queryKey: ["release-checklist", activeReleaseId],
    queryFn: () => getReleaseChecklist(activeReleaseId),
    enabled: Boolean(activeReleaseId),
  });
  const activeRelease = useMemo(
    () => releasesQuery.data?.items.find((item) => item.id === activeReleaseId),
    [releasesQuery.data, activeReleaseId],
  );
  const createMut = useMutation({
    mutationFn: createRelease,
    onSuccess: (rel) => {
      setReleaseId(rel.id);
      qc.invalidateQueries({ queryKey: ["releases"] });
    },
  });
  const checklistMut = useMutation({
    mutationFn: (item: ReleaseChecklistItem) =>
      patchReleaseChecklist(activeReleaseId, [{ id: item.id, status: item.status === "done" ? "pending" : "done", evidenceRef: item.evidenceRef || "ui:checked" }]),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["release-checklist", activeReleaseId] }),
  });
  const gateMut = useMutation({
    mutationFn: () => evaluateReleaseGate(activeReleaseId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["releases"] }),
  });
  const rollbackMut = useMutation({ mutationFn: (body: { scenario: string; status?: string; durationMs?: number; notes?: string }) => createRollbackDrill(activeReleaseId, body) });

  return (
    <section className="panel active">
      <div className="page-kicker">
        <ClipboardCheck size={17} strokeWidth={1.8} />
        Releases
      </div>
      <div className="page-heading">
        <div>
          <h1>发布与灰度回滚</h1>
          <p>管理发布清单、生产切换门禁、灰度观察证据和回滚演练记录。</p>
        </div>
        <div className="toolbar metrics-toolbar">
          <label className="scenario-picker">
            Release
            <select value={activeReleaseId} onChange={(e) => setReleaseId(e.target.value)}>
              {(releasesQuery.data?.items ?? []).map((release) => (
                <option key={release.id} value={release.id}>
                  {release.version}
                </option>
              ))}
              {!releasesQuery.data?.items.length && <option value="">无发布</option>}
            </select>
          </label>
          <button className="btn icon-btn" onClick={() => releasesQuery.refetch()}>
            <RefreshCcw size={16} strokeWidth={1.8} />
            刷新
          </button>
          <button className="btn primary icon-btn" onClick={() => gateMut.mutate()} disabled={!activeReleaseId || gateMut.isPending}>
            <PlayCircle size={16} strokeWidth={1.8} />
            运行 gate
          </button>
        </div>
      </div>

      {(releasesQuery.error || checklistQuery.error || createMut.error || checklistMut.error || gateMut.error || rollbackMut.error) && (
        <p className="error-text">
          {(releasesQuery.error || checklistQuery.error || createMut.error || checklistMut.error || gateMut.error || rollbackMut.error as Error)?.message}
        </p>
      )}

      <div className="split ops-split">
        <div className="pane">
          <div className="pane-title">
            <h2>创建发布</h2>
            <span>{createMut.isPending ? "saving" : "draft"}</span>
          </div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              const form = new FormData(event.currentTarget);
              createMut.mutate({
                version: String(form.get("version") || ""),
                title: String(form.get("title") || ""),
                canaryStrategy: String(form.get("canaryStrategy") || ""),
              });
            }}
          >
            <label>
              Version
              <input name="version" placeholder="v0.4.0" required />
            </label>
            <label>
              Title
              <input name="title" placeholder="MVP release" />
            </label>
            <label className="wide-field">
              Canary Strategy
              <textarea name="canaryStrategy" rows={4} placeholder="按 space/project 分批观察错误率、低分反馈和 active alerts。" />
            </label>
            <button className="btn primary icon-btn" type="submit" disabled={createMut.isPending}>
              <Send size={16} strokeWidth={1.8} />
              创建 release
            </button>
          </form>
        </div>

        <div className="pane">
          <div className="pane-title">
            <h2>当前发布</h2>
            <span>{activeRelease?.gateStatus ?? "pending"}</span>
          </div>
          <pre className="code-block tall">
            {activeRelease
              ? JSON.stringify(
                  {
                    id: activeRelease.id,
                    version: activeRelease.version,
                    status: activeRelease.status,
                    gateStatus: activeRelease.gateStatus,
                    canaryStrategy: activeRelease.canaryStrategy,
                  },
                  null,
                  2,
                )
              : "暂无发布记录。"}
          </pre>
        </div>
      </div>

      <div className="split ops-split">
        <div className="pane">
          <div className="pane-title">
            <h2>MVP Checklist</h2>
            <span>{checklistQuery.data?.items.filter((item) => item.status === "done").length ?? 0} done</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>item</th>
                <th>status</th>
                <th>toggle</th>
              </tr>
            </thead>
            <tbody>
              {(checklistQuery.data?.items ?? []).length === 0 && (
                <tr className="empty-row">
                  <td colSpan={3}>创建发布后生成清单。</td>
                </tr>
              )}
              {(checklistQuery.data?.items ?? []).map((item) => (
                <tr key={item.id}>
                  <td>{item.label}</td>
                  <td>
                    <StatusPill value={item.status} />
                  </td>
                  <td>
                    <button className="btn mini" onClick={() => checklistMut.mutate(item)} disabled={checklistMut.isPending}>
                      {item.status === "done" ? "undo" : "done"}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="pane">
          <div className="pane-title">
            <h2>Gate 结果</h2>
            <span>{gateMut.data?.overall ?? "idle"}</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>gate</th>
                <th>status</th>
                <th>message</th>
              </tr>
            </thead>
            <tbody>
              {(gateMut.data?.results ?? []).length === 0 && (
                <tr className="empty-row">
                  <td colSpan={3}>运行 gate 后显示结果。</td>
                </tr>
              )}
              {(gateMut.data?.results ?? []).map((item) => (
                <tr key={item.id}>
                  <td>{item.gateKey}</td>
                  <td>
                    <StatusPill value={item.status} />
                  </td>
                  <td>{item.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>回滚演练</h2>
          <span>{rollbackMut.isSuccess ? "recorded" : "ready"}</span>
        </div>
        <form
          className="inline-form release-drill-form"
          onSubmit={(event) => {
            event.preventDefault();
            const form = new FormData(event.currentTarget);
            rollbackMut.mutate({
              scenario: String(form.get("scenario") || ""),
              status: String(form.get("status") || "recorded"),
              durationMs: Number(form.get("durationMs") || 0),
              notes: String(form.get("notes") || ""),
            });
          }}
        >
          <input name="scenario" placeholder="rollback image / switch database URL" required />
          <select name="status" defaultValue="passed">
            <option value="passed">passed</option>
            <option value="recorded">recorded</option>
            <option value="failed">failed</option>
          </select>
          <input name="durationMs" type="number" min="0" placeholder="duration ms" />
          <input name="notes" placeholder="evidence / notes" />
          <button className="btn icon-btn" type="submit" disabled={!activeReleaseId || rollbackMut.isPending}>
            <RotateCcw size={16} strokeWidth={1.8} />
            记录
          </button>
        </form>
      </div>
    </section>
  );
}

function StatusPill({ value }: { value: string }) {
  const lowered = value.toLowerCase();
  const tone = ["pass", "done", "passed", "ok"].includes(lowered) ? "ok" : ["block", "failed"].includes(lowered) ? "err" : "idle";
  return (
    <span className={`status-pill ${tone}`}>
      <span className="status-dot" />
      {value || "unknown"}
    </span>
  );
}
