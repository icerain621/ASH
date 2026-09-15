import type { ChatBubbleSelection } from "./ChatTranscript";

type Props = {
  open: boolean;
  selection: ChatBubbleSelection | null;
};

function readablePayload(payload: unknown): unknown {
  if (typeof payload !== "string") return payload ?? null;
  const trimmed = payload.trim();
  if (!trimmed) return payload;
  try {
    return JSON.parse(trimmed) as unknown;
  } catch {
    return payload;
  }
}

/** Right details pane for selected chat/trajectory node. */
export function DetailsPane({ open, selection }: Props) {
  const payload = selection ? readablePayload(selection.payload) : null;
  const source =
    payload && typeof payload === "object" && payload !== null && "source" in payload
      ? String((payload as Record<string, unknown>).source ?? "")
      : "";
  return (
    <aside
      className={`agent-chat-details${open ? " open" : ""}`}
      data-testid="agent-chat-details"
      data-open={open ? "1" : "0"}
      hidden={!open}
    >
      <div className="pane-title">
        <h2>Details</h2>
        <span>{selection ? selection.kind : "—"}</span>
      </div>
      {!selection ? (
        <p className="muted-line">点击气泡查看摘要与 JSON</p>
      ) : (
        <div className="agent-chat-details-body" data-testid="agent-chat-details-body">
          <p>
            <strong>{selection.title}</strong>
          </p>
          {source ? (
            <p className="muted-line" data-testid="agent-chat-details-source">
              source: {source}
            </p>
          ) : null}
          <p className="muted-line">{selection.summary}</p>
          <pre className="code-block compact">
            {JSON.stringify(
              {
                id: selection.id,
                type: selection.type,
                kind: selection.kind,
                payload,
              },
              null,
              2,
            )}
          </pre>
        </div>
      )}
    </aside>
  );
}
