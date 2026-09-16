import { useState, type MouseEvent } from "react";
import type { MergedChatBubble } from "./conversationNodes";

type Props = {
  item: MergedChatBubble;
  active?: boolean;
  onSelect?: () => void;
};

function statusLabel(status: MergedChatBubble["toolStatus"]): string {
  if (status === "running") return "执行中";
  if (status === "error") return "失败";
  if (status === "ok") return "完成";
  return "";
}

/** Expandable in-stream tool card (DSH ToolRow-aligned, no Cordis). */
export function ToolCallBubble({ item, active = false, onSelect }: Props) {
  const [open, setOpen] = useState(item.toolStatus === "error");
  const label = statusLabel(item.toolStatus);
  const hasBody = Boolean(item.toolInput || item.toolOutput || item.summary);

  const toggle = (e: MouseEvent) => {
    e.stopPropagation();
    setOpen((v) => !v);
  };

  return (
    <button
      type="button"
      className={`agent-chat-bubble role-tool agent-tool-card${active ? " active" : ""}${
        item.toolStatus ? ` tool-${item.toolStatus}` : ""
      }${open ? " expanded" : ""}`}
      data-testid="agent-chat-bubble-tool"
      data-role="tool"
      data-node-kind={item.kind}
      data-tool-status={item.toolStatus || undefined}
      data-expanded={open ? "1" : "0"}
      data-active={active ? "1" : "0"}
      onClick={() => onSelect?.()}
    >
      <div className="agent-chat-bubble-meta">
        <span className="agent-tool-dot" data-status={item.toolStatus || "running"} aria-hidden />
        <strong>{item.title}</strong>
        {label ? (
          <span className="agent-tool-status" data-testid="agent-tool-status" data-status={item.toolStatus}>
            {label}
          </span>
        ) : null}
        {hasBody ? (
          <span
            className="agent-tool-chevron"
            data-testid="agent-tool-toggle"
            role="presentation"
            onClick={toggle}
          >
            {open ? "▾" : "▸"}
          </span>
        ) : null}
      </div>
      {!open ? (
        <div className="agent-chat-bubble-body agent-tool-preview">
          {(item.toolOutput || item.toolInput || item.summary || "").slice(0, 120)}
          {(item.toolOutput || item.toolInput || item.summary || "").length > 120 ? "…" : ""}
        </div>
      ) : (
        <div className="agent-tool-io" data-testid="agent-tool-io">
          {item.toolInput ? (
            <div className="agent-tool-block" data-testid="agent-tool-input">
              <span className="agent-tool-block-label">IN</span>
              <pre>{item.toolInput}</pre>
            </div>
          ) : null}
          {item.toolOutput || item.toolStatus === "running" ? (
            <div className="agent-tool-block" data-testid="agent-tool-output">
              <span className="agent-tool-block-label">OUT</span>
              <pre>{item.toolOutput || "…"}</pre>
            </div>
          ) : null}
          {!item.toolInput && !item.toolOutput ? (
            <div className="agent-chat-bubble-body">{item.summary}</div>
          ) : null}
        </div>
      )}
    </button>
  );
}
