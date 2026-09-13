import { useQuery } from "@tanstack/react-query";
import { listInteractionSessionThreads } from "../api/interactions.api";

type Props = {
  sessionId: string;
};

/** Read-only Session → Threads list (GV §7). */
export function SessionThreadsList({ sessionId }: Props) {
  const q = useQuery({
    queryKey: ["interaction-session-threads", sessionId],
    queryFn: () => listInteractionSessionThreads(sessionId),
    enabled: Boolean(sessionId),
  });

  const items = q.data?.items ?? [];

  return (
    <div data-testid="session-threads-list" style={{ marginTop: "0.75rem" }}>
      <div className="pane-title" style={{ marginBottom: "0.35rem" }}>
        <h3 style={{ margin: 0, fontSize: "0.95rem" }}>Threads</h3>
        <span className="muted-line">{items.length ? `${items.length}` : q.isPending ? "…" : "0"}</span>
      </div>
      {q.isError ? (
        <p className="error-text" data-testid="session-threads-error">
          {(q.error as Error).message}
        </p>
      ) : null}
      {!q.isPending && items.length === 0 ? (
        <p className="muted-line" data-testid="session-threads-empty">
          尚无 Thread
        </p>
      ) : null}
      {items.length > 0 ? (
        <ul style={{ listStyle: "none", margin: 0, padding: 0 }}>
          {items.map((th) => (
            <li
              key={th.id}
              data-testid="session-thread-item"
              className="muted-line"
              style={{ fontFamily: "ui-monospace, monospace", fontSize: "0.8rem", marginBottom: "0.25rem" }}
            >
              {th.kind}/{th.status} · {th.id.slice(0, 12)} · run {th.runId.slice(0, 10)}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
