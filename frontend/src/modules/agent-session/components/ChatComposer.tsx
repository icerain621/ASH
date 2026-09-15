import type { AgentSessionView } from "../api/session.api";
import { IntentBar, type IntentPayload } from "./IntentBar";
import { ChatSeats } from "./ChatSeats";

type Props = {
  mode: "prompt" | "gate";
  busy?: boolean;
  canStop?: boolean;
  gateReason?: string;
  session?: AgentSessionView | null;
  onIntent: (payload: IntentPayload) => void;
};

/** Sticky bottom composer — seats + IntentBar with chat-shell card chrome. */
export function ChatComposer({
  mode,
  busy,
  canStop,
  gateReason,
  session = null,
  onIntent,
}: Props) {
  return (
    <div className="agent-chat-composer" data-testid="agent-chat-composer">
      {mode === "prompt" ? <ChatSeats session={session} disabled={busy} /> : null}
      <IntentBar
        mode={mode}
        busy={busy}
        canStop={canStop}
        gateReason={gateReason}
        onIntent={onIntent}
      />
    </div>
  );
}
