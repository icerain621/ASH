import type { AgentSessionView } from "../api/session.api";
import { permissionStripLabel, resolvePermissionMode } from "../permissionModeLabels";
import { IntentBar, type IntentPayload } from "./IntentBar";
import { ChatSeats } from "./ChatSeats";

type Props = {
  mode: "prompt" | "gate";
  busy?: boolean;
  canStop?: boolean;
  gateReason?: string;
  gateTool?: string;
  session?: AgentSessionView | null;
  onIntent: (payload: IntentPayload) => void;
  onOpenTools?: () => void;
  onOpenMcp?: () => void;
  onOpenSkills?: () => void;
};

/** Sticky bottom composer — seats + IntentBar with chat-shell card chrome. */
export function ChatComposer({
  mode,
  busy,
  canStop,
  gateReason,
  gateTool,
  session = null,
  onIntent,
  onOpenTools,
  onOpenMcp,
  onOpenSkills,
}: Props) {
  const hasShortcuts = Boolean(onOpenTools || onOpenMcp || onOpenSkills);

  return (
    <div className="agent-chat-composer" data-testid="agent-chat-composer">
      {mode === "prompt" ? <ChatSeats session={session} disabled={busy} /> : null}
      {mode === "prompt" && session?.id ? (
        <p className="muted-line agent-chat-permission-strip" data-testid="agent-chat-permission-strip">
          审批：{permissionStripLabel(resolvePermissionMode(session.permissionMode))}
        </p>
      ) : null}
      {hasShortcuts && mode === "prompt" ? (
        <div className="agent-composer-shortcuts" data-testid="agent-composer-shortcuts">
          {onOpenTools ? (
            <button
              type="button"
              className="btn mini"
              data-testid="agent-composer-tools"
              disabled={busy}
              onClick={onOpenTools}
            >
              Tools
            </button>
          ) : null}
          {onOpenMcp ? (
            <button
              type="button"
              className="btn mini"
              data-testid="agent-composer-mcp"
              disabled={busy}
              onClick={onOpenMcp}
            >
              MCP
            </button>
          ) : null}
          {onOpenSkills ? (
            <button
              type="button"
              className="btn mini"
              data-testid="agent-composer-skills"
              disabled={busy}
              onClick={onOpenSkills}
            >
              Skills
            </button>
          ) : null}
        </div>
      ) : null}
      <IntentBar
        mode={mode}
        busy={busy}
        canStop={canStop}
        gateReason={gateReason}
        gateTool={gateTool}
        onIntent={onIntent}
      />
    </div>
  );
}
