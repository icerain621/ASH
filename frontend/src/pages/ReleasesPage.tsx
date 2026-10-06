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
  const [releaseVersion, setReleaseVersion] = useState("");
  const [releaseTitle, setReleaseTitle] = useState("");
  const [rollbackScenario, setRollbackScenario] = useState("");
  const releasesQuery = useQuery({ queryKey: ["releases"], queryFn: () => listReleases({ limit: 50 }) });
  const activeReleaseId = releaseId || releasesQuery.data?.items?.[0]?.id || "";
  const checklistQuery = useQuery({
    queryKey: ["release-checklist", activeReleaseId],
    queryFn: () => getReleaseChecklist(activeReleaseId),
    enabled: Boolean(activeReleaseId),
  });
  const activeRelease = useMemo(
    () => (releasesQuery.data?.items ?? []).find((item) => item.id === activeReleaseId),
    [releasesQuery.data, activeReleaseId],
  );
  const createMut = useMutation({
    mutationFn: createRelease,
    onSuccess: (rel) => {
      setReleaseId(rel.id);
      setReleaseVersion("");
      setReleaseTitle("");
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
  const rollbackMut = useMutation({
    mutationFn: (body: { scenario: string; status?: string; durationMs?: number; notes?: string }) =>
      createRollbackDrill(activeReleaseId, body),
    onSuccess: () => setRollbackScenario(""),
  });
  const rollbackReady = Boolean(activeReleaseId) && Boolean(rollbackScenario.trim());

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
              {!releasesQuery.data?.items?.length && <option value="">无发布</option>}
            </select>
          </label>
          <button
            className="btn icon-btn"
            onClick={() => releasesQuery.refetch()}
            title="刷新发布列表"
          >
            <RefreshCcw size={16} strokeWidth={1.8} />
            刷新
          </button>
          <button
            className="btn primary icon-btn"
            data-testid="release-gate-run"
            onClick={() => {
              if (!activeReleaseId) return;
              const ok = window.confirm(`确认对 release「${activeReleaseId}」运行 gate？`);
              if (!ok) return;
              gateMut.mutate();
            }}
            disabled={!activeReleaseId || gateMut.isPending}
            title={
              !activeReleaseId
                ? "需要先选择或创建 release"
                : gateMut.isPending
                  ? "运行中…"
                  : "运行 gate（需确认）"
            }
          >
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
              if (!releaseVersion.trim() || !releaseTitle.trim()) return;
              const ok = window.confirm(
                `确认创建 release「${releaseVersion.trim()} · ${releaseTitle.trim()}」？`,
              );
              if (!ok) return;
              const form = new FormData(event.currentTarget);
              createMut.mutate({
                version: releaseVersion.trim(),
                title: releaseTitle.trim(),
                canaryStrategy: String(form.get("canaryStrategy") || ""),
              });
            }}
          >
            <label>
              Version
              <input
                name="version"
                placeholder="v0.4.0"
                required
                value={releaseVersion}
                onChange={(e) => setReleaseVersion(e.target.value)}
                data-testid="release-version"
              />
            </label>
            <label>
              Title
              <input
                name="title"
                placeholder="MVP release"
                required
                value={releaseTitle}
                onChange={(e) => setReleaseTitle(e.target.value)}
                data-testid="release-title"
              />
            </label>
            <label className="wide-field">
              Canary Strategy
              <textarea name="canaryStrategy" rows={4} placeholder="按 space/project 分批观察错误率、低分反馈和 active alerts。" />
            </label>
            <button
              className="btn primary icon-btn"
              type="submit"
              disabled={createMut.isPending || !releaseVersion.trim() || !releaseTitle.trim()}
              title={
                !releaseVersion.trim()
                  ? "需要填写 Version"
                  : !releaseTitle.trim()
                    ? "需要填写 Title"
                    : createMut.isPending
                      ? "创建中…"
                      : "创建 release（需确认）"
              }
              data-testid="release-create"
            >
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
            <span>{(checklistQuery.data?.items ?? []).filter((item) => item.status === "done").length} done</span>
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
                    <button
                      className="btn mini"
                      onClick={() => checklistMut.mutate(item)}
                      disabled={checklistMut.isPending || !activeReleaseId}
                      title={
                        !activeReleaseId
                          ? "需要先选择 release"
                          : checklistMut.isPending
                            ? "更新中…"
                            : item.status === "done"
                              ? "撤销完成标记"
                              : "标记清单项为完成"
                      }
                      data-testid={`release-checklist-toggle-${item.id}`}
                    >
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
            if (!rollbackReady) return;
            const ok = window.confirm(
              `确认记录回滚演练「${rollbackScenario.trim()}」？`,
            );
            if (!ok) return;
            const form = new FormData(event.currentTarget);
            rollbackMut.mutate({
              scenario: rollbackScenario.trim(),
              status: String(form.get("status") || "recorded"),
              durationMs: Number(form.get("durationMs") || 0),
              notes: String(form.get("notes") || ""),
            });
          }}
        >
          <input
            name="scenario"
            data-testid="release-rollback-scenario"
            placeholder="rollback image / switch database URL"
            value={rollbackScenario}
            onChange={(e) => setRollbackScenario(e.target.value)}
          />
          <select name="status" defaultValue="passed">
            <option value="passed">passed</option>
            <option value="recorded">recorded</option>
            <option value="failed">failed</option>
          </select>
          <input name="durationMs" type="number" min="0" placeholder="duration ms" />
          <input name="notes" placeholder="evidence / notes" />
          <button
            className="btn icon-btn"
            type="submit"
            data-testid="release-rollback-submit"
            disabled={!rollbackReady || rollbackMut.isPending}
            title={
              !activeReleaseId
                ? "需要先选择或创建 release"
                : !rollbackScenario.trim()
                  ? "需要填写回滚场景"
                  : rollbackMut.isPending
                    ? "记录中…"
                    : "记录回滚演练（需确认）"
            }
          >
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
