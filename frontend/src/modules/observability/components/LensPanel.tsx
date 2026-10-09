import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { getAgentLens, getGlobalLens, getMemoryLens, type ObserveLens } from "../api/lenses.api";
import { getCurrentSpaceId } from "@/services/http/client";

type Props = {
  lens: ObserveLens;
  runId?: string;
  memoryId?: string;
  compact?: boolean;
};

export function LensPanel({ lens, runId, memoryId, compact }: Props) {
  const spaceId = getCurrentSpaceId();
  const globalQ = useQuery({
    queryKey: ["observe-lens", "global", spaceId],
    queryFn: () => getGlobalLens(spaceId),
    enabled: lens === "global",
  });
  const agentQ = useQuery({
    queryKey: ["observe-lens", "agent", runId],
    queryFn: () => getAgentLens(runId!),
    enabled: lens === "agent" && Boolean(runId),
  });
  const memoryQ = useQuery({
    queryKey: ["observe-lens", "memory", memoryId],
    queryFn: () => getMemoryLens(memoryId!),
    enabled: lens === "memory" && Boolean(memoryId),
  });

  return (
    <div data-testid="lens-panel" data-lens={lens} className={compact ? "lens-panel compact" : "lens-panel"}>
      {lens === "global" ? (
        <section data-testid="lens-global">
          {globalQ.isLoading ? <p className="muted-line">加载全局镜头…</p> : null}
          {globalQ.data ? (
            <ul className="chip-list">
              <li data-testid="chip-running">在跑 {globalQ.data.runningRuns}</li>
              <li data-testid="chip-review">待评审记忆 {globalQ.data.reviewPending}</li>
              <li data-testid="chip-gate">门禁拒绝 {globalQ.data.gateRejects}</li>
              <li data-testid="chip-doctor">Doctor {globalQ.data.doctorChip}</li>
            </ul>
          ) : null}
          {globalQ.data?.templateUsage?.length ? (
            <p className="muted-line">
              模板用量：{globalQ.data.templateUsage.map((t) => `${t.id}×${t.count}`).join(", ")}
            </p>
          ) : null}
        </section>
      ) : null}

      {lens === "agent" ? (
        <section data-testid="lens-agent">
          {!runId ? <p className="muted-line">提供 run 参数查看 Agent 瀑布</p> : null}
          {agentQ.isLoading ? <p className="muted-line">加载 Agent 镜头…</p> : null}
          {agentQ.data ? (
            <>
              <p>
                Run <code data-testid="lens-agent-run">{agentQ.data.runId}</code> · Trace{" "}
                <code data-testid="lens-agent-trace">{agentQ.data.traceId}</code>
              </p>
              <p className="muted-line">承认事件 {agentQ.data.events?.length ?? 0} · spans {agentQ.data.waterfall?.spans?.length ?? 0}</p>
              <ul>
                {(agentQ.data.events ?? []).slice(0, compact ? 5 : 40).map((ev, i) => (
                  <li key={`${ev.type}-${ev.ts}-${i}`}>{ev.type}</li>
                ))}
              </ul>
              {memoryId ? null : (
                <Link
                  to="/observe"
                  search={{ lens: "global" }}
                  className="inline-link"
                  data-testid="lens-to-global"
                >
                  全局镜头
                </Link>
              )}
            </>
          ) : null}
        </section>
      ) : null}

      {lens === "memory" ? (
        <section data-testid="lens-memory">
          {!memoryId ? <p className="muted-line">提供 id 参数查看记忆谱系</p> : null}
          {memoryQ.isLoading ? <p className="muted-line">加载记忆镜头…</p> : null}
          {memoryQ.data ? (
            <>
              <p>
                Memory <code data-testid="lens-memory-id">{memoryQ.data.memoryId}</code> · {memoryQ.data.status}
                {memoryQ.data.redacted ? " · 已打码" : ""}
              </p>
              <ol data-testid="lens-memory-stages">
                {memoryQ.data.stages.map((st, i) => (
                  <li key={`${st.stage}-${i}`}>{st.stage}{st.detail ? ` — ${st.detail}` : ""}</li>
                ))}
              </ol>
            </>
          ) : null}
        </section>
      ) : null}
    </div>
  );
}
