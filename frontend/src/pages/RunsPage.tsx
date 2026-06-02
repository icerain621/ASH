import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  createRun,
  getRun,
  getRunArtifacts,
  listRuns,
  replayRun,
  resumeRun,
  type RunSummary,
} from "@/modules/runs/api/runs.api";
import { useRunStream } from "@/services/sse/runStream";
import { fmtTime, shortId } from "@/shared/utils/format";

export function RunsPage() {
  const qc = useQueryClient();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const runsQuery = useQuery({
    queryKey: ["runs"],
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

  const streamLines = useRunStream(selectedId);

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
      await qc.invalidateQueries({ queryKey: ["runs", selectedId] });
    },
  });

  const replayMut = useMutation({
    mutationFn: () => replayRun(selectedId!, { mode: "exact" }),
    onSuccess: async (res) => {
      await qc.invalidateQueries({ queryKey: ["runs"] });
      setSelectedId(res.runId);
    },
  });

  const items = runsQuery.data?.items ?? [];
  const selected = detailQuery.data;
  const err =
    runsQuery.error?.message ||
    createMut.error?.message ||
    resumeMut.error?.message ||
    replayMut.error?.message;

  return (
    <section className="panel active">
      <div className="toolbar">
        <button className="btn" onClick={() => runsQuery.refetch()} disabled={runsQuery.isFetching}>
          刷新
        </button>
        <button className="btn primary" onClick={() => createMut.mutate()} disabled={createMut.isPending}>
          新建 Run (feature_delivery)
        </button>
        {selectedId && selected?.status === "failed" && (
          <button className="btn" onClick={() => resumeMut.mutate()} disabled={resumeMut.isPending}>
            Resume
          </button>
        )}
        {selectedId && (
          <button className="btn" onClick={() => replayMut.mutate()} disabled={replayMut.isPending}>
            Replay (exact)
          </button>
        )}
      </div>
      {err && <p className="error-text">{err}</p>}
      <div className="split">
        <div className="pane">
          <h2>Runs</h2>
          <table className="table">
            <thead>
              <tr>
                <th>Run ID</th>
                <th>Status</th>
                <th>Scenario</th>
                <th>Started</th>
              </tr>
            </thead>
            <tbody>
              {items.map((r: RunSummary) => (
                <tr
                  key={r.runId}
                  className={r.runId === selectedId ? "selected" : ""}
                  onClick={() => setSelectedId(r.runId)}
                >
                  <td title={r.runId}>{shortId(r.runId)}</td>
                  <td>{r.status}</td>
                  <td>{r.scenario?.name}</td>
                  <td>{fmtTime(r.startedAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="pane">
          <h2>
            Run 详情 <span className="muted">{selectedId || ""}</span>
          </h2>
          <pre className="code-block">
            {selected ? JSON.stringify(selected, null, 2) : "选择一条 Run"}
          </pre>
          <h3>事件流 (SSE)</h3>
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
          </div>
          <h3>Artifacts</h3>
          <pre className="code-block">
            {artifactsQuery.data
              ? JSON.stringify(artifactsQuery.data, null, 2)
              : selectedId
                ? "—"
                : "—"}
          </pre>
        </div>
      </div>
    </section>
  );
}
