import { IntentBar, type IntentPayload } from "./IntentBar";

type Props = {
  mode: "prompt" | "gate";
  busy?: boolean;
  canStop?: boolean;
  gateReason?: string;
  onIntent: (payload: IntentPayload) => void;
};

/** Sticky bottom composer — IntentBar with chat-shell card chrome. */
export function ChatComposer({ mode, busy, canStop, gateReason, onIntent }: Props) {
  return (
    <div className="agent-chat-composer" data-testid="agent-chat-composer">
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
