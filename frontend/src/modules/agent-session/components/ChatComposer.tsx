import { useState } from "react";
import { sessionFollowUpQueue, type AgentSessionView } from "../api/session.api";
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

/** Sticky bottom composer — DSH InputBar: + menu, seats, model/effort, send. */
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
  const queueItems = sessionFollowUpQueue(session?.meta);
  const [closeSeatsSignal, setCloseSeatsSignal] = useState(0);
  const [closePlusSignal, setClosePlusSignal] = useState(0);
  const plusItems =
    mode === "prompt" && (onOpenTools || onOpenMcp || onOpenSkills) ? (
      <>
        {onOpenTools ? (
          <li>
            <button
              type="button"
              className="agent-composer-plus-item"
              data-testid="agent-composer-tools"
              disabled={busy}
              role="menuitem"
              title={busy ? "会话忙，稍后再打开 Tools" : "打开 Tools / 风险面板"}
              onClick={onOpenTools}
            >
              Tools
            </button>
          </li>
        ) : null}
        {onOpenMcp ? (
          <li>
            <button
              type="button"
              className="agent-composer-plus-item"
              data-testid="agent-composer-mcp"
              disabled={busy}
              role="menuitem"
              title={busy ? "会话忙，稍后再打开 MCP" : "打开 MCP 工具面板"}
              onClick={onOpenMcp}
            >
              MCP
            </button>
          </li>
        ) : null}
        {onOpenSkills ? (
          <li>
            <button
              type="button"
              className="agent-composer-plus-item"
              data-testid="agent-composer-skills"
              disabled={busy}
              role="menuitem"
              title={busy ? "会话忙，稍后再打开 Skills" : "打开 Skills 目录"}
              onClick={onOpenSkills}
            >
              Skills
            </button>
          </li>
        ) : null}
      </>
    ) : null;

  return (
    <div className="agent-chat-composer" data-testid="agent-chat-composer" data-composer-card="">
      <IntentBar
        mode={mode}
        busy={busy}
        canStop={canStop}
        gateReason={gateReason}
        gateTool={gateTool}
        queueItems={queueItems}
        toolbarStart={
          mode === "prompt" ? (
            <ChatSeats
              session={session}
              disabled={busy}
              closeMenuSignal={closeSeatsSignal}
              onMenuOpened={() => setClosePlusSignal((n) => n + 1)}
            />
          ) : null
        }
        plusItems={plusItems}
        closePlusSignal={closePlusSignal}
        onPlusOpened={() => setCloseSeatsSignal((n) => n + 1)}
        onIntent={onIntent}
      />
    </div>
  );
}
