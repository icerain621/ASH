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

function asRecord(payload: unknown): Record<string, unknown> {
  if (payload && typeof payload === "object" && !Array.isArray(payload)) {
    return payload as Record<string, unknown>;
  }
  return {};
}

function str(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

function isToolSelection(selection: ChatBubbleSelection): boolean {
  return (
    selection.kind === "tool.card" ||
    selection.kind === "tool.called" ||
    selection.kind === "tool.result" ||
    selection.kind === "tool" ||
    selection.type.startsWith("tool.")
  );
}

function isHookSelection(selection: ChatBubbleSelection): boolean {
  return selection.kind.startsWith("hook") || selection.type.startsWith("hook.");
}

const HOOK_DETAIL_KEYS = ["event", "tool", "action", "risk", "stepId", "ruleIndex", "reason"] as const;

/** Right details pane for selected chat/trajectory node. */
export function DetailsPane({ open, selection }: Props) {
  const payload = selection ? readablePayload(selection.payload) : null;
  const rec = asRecord(payload);
  const source = str(rec.source);
  const toolView = selection && isToolSelection(selection);
  const hookView = selection && isHookSelection(selection);

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

          {hookView ? (
            <dl className="agent-details-hook" data-testid="agent-details-hook">
              {HOOK_DETAIL_KEYS.map((key) => {
                const val = rec[key];
                if (val == null || val === "") return null;
                return (
                  <div key={key} className="agent-details-kv">
                    <dt>{key}</dt>
                    <dd>{str(val)}</dd>
                  </div>
                );
              })}
            </dl>
          ) : null}

          {toolView ? (
            <div className="agent-details-tool" data-testid="agent-details-tool">
              <p className="muted-line">
                {selection.toolName || str(rec.name) || str(rec.tool) || "tool"}
                {selection.toolStatus ? ` · ${selection.toolStatus}` : ""}
                {str(rec.durationMs) ? ` · ${str(rec.durationMs)}ms` : ""}
                {str(rec.failureClass) ? ` · ${str(rec.failureClass)}` : ""}
              </p>
              {(selection.toolInput || str(rec.input) || str(rec.args)) && (
                <div className="agent-tool-block" data-testid="agent-details-tool-input">
                  <span className="agent-tool-block-label">IN</span>
                  <pre>{selection.toolInput || str(rec.input) || str(rec.args)}</pre>
                </div>
              )}
              {(selection.toolOutput || str(rec.output) || str(rec.result) || str(rec.error)) && (
                <div className="agent-tool-block" data-testid="agent-details-tool-output">
                  <span className="agent-tool-block-label">OUT</span>
                  <pre>
                    {selection.toolOutput || str(rec.output) || str(rec.result) || str(rec.error)}
                  </pre>
                </div>
              )}
            </div>
          ) : hookView ? null : (
            <p className="muted-line">{selection.summary}</p>
          )}

          <pre className="code-block compact" data-testid="agent-details-raw-json">
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
