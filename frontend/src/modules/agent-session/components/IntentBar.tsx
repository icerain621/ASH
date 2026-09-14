import { useState } from "react";
import type { SessionIntentAction } from "../api/session.api";

export type IntentPayload = {
  action: SessionIntentAction;
  prompt?: string;
  reason?: string;
};

type Props = {
  mode: "prompt" | "gate";
  busy?: boolean;
  /** Show Stop when run is generating / waiting. */
  canStop?: boolean;
  gateReason?: string;
  onIntent: (payload: IntentPayload) => void;
};

/** Thin composer: prompt input or gate takeover (DSH-aligned). */
export function IntentBar({ mode, busy = false, canStop = false, gateReason, onIntent }: Props) {
  const [prompt, setPrompt] = useState("");

  if (mode === "gate") {
    return (
      <div className="agent-intent-bar" data-testid="agent-intent-bar" data-mode="gate">
        <p className="muted-line" data-testid="agent-intent-gate-reason">
          {gateReason || "Run 等待审批"}
        </p>
        <div className="row-actions">
          <button
            type="button"
            className="btn mini ok"
            disabled={busy}
            data-testid="agent-intent-approve"
            onClick={() => onIntent({ action: "approve", reason: "approved from IntentBar" })}
          >
            批准
          </button>
          <button
            type="button"
            className="btn mini err"
            disabled={busy}
            data-testid="agent-intent-reject"
            onClick={() => onIntent({ action: "reject", reason: "rejected from IntentBar" })}
          >
            拒绝
          </button>
          <button
            type="button"
            className="btn mini"
            disabled={busy}
            data-testid="agent-intent-stop"
            onClick={() => onIntent({ action: "stop" })}
          >
            停止
          </button>
          <button
            type="button"
            className="btn mini"
            disabled={busy}
            data-testid="agent-intent-cancel"
            onClick={() => onIntent({ action: "cancel" })}
          >
            取消
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="agent-intent-bar" data-testid="agent-intent-bar" data-mode="prompt">
      <label className="wide-field">
        意图
        <textarea
          rows={2}
          value={prompt}
          disabled={busy}
          data-testid="agent-intent-prompt"
          placeholder="只发意图，不拼系统 prompt"
          onChange={(e) => setPrompt(e.target.value)}
        />
      </label>
      <div className="row-actions">
        {canStop ? (
          <button
            type="button"
            className="btn mini err"
            disabled={busy}
            data-testid="agent-intent-stop"
            onClick={() => onIntent({ action: "stop" })}
          >
            停止
          </button>
        ) : null}
        <button
          type="button"
          className="btn mini ok"
          disabled={busy || !prompt.trim()}
          data-testid="agent-intent-send"
          onClick={() => {
            const text = prompt.trim();
            if (!text) return;
            onIntent({ action: "prompt", prompt: text });
            setPrompt("");
          }}
        >
          发送
        </button>
      </div>
    </div>
  );
}
