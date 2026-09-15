import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import {
  listAgentCommands,
  type AgentCommandItem,
  type SessionIntentAction,
} from "../api/session.api";

export type IntentPayload = {
  action: SessionIntentAction;
  prompt?: string;
  reason?: string;
  command?: string;
  args?: string;
};

type Props = {
  mode: "prompt" | "gate";
  busy?: boolean;
  /** Show Stop when run is generating / waiting. */
  canStop?: boolean;
  gateReason?: string;
  onIntent: (payload: IntentPayload) => void;
};

function ensureSlash(name: string): string {
  const n = name.trim();
  if (!n) return "/";
  return n.startsWith("/") ? n : `/${n}`;
}

/** Thin composer: prompt input or gate takeover (DSH-aligned). */
export function IntentBar({ mode, busy = false, canStop = false, gateReason, onIntent }: Props) {
  const [prompt, setPrompt] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);

  const commandsQuery = useQuery({
    queryKey: ["agent-commands"],
    queryFn: () => listAgentCommands(),
    enabled: menuOpen || prompt.startsWith("/"),
    staleTime: 60_000,
  });

  const showMenu = menuOpen || (prompt.startsWith("/") && !busy);
  const filter = prompt.startsWith("/") ? prompt.toLowerCase() : "";
  const items: AgentCommandItem[] = (commandsQuery.data?.items ?? []).filter((it) => {
    if (!filter || filter === "/") return true;
    return it.name.toLowerCase().startsWith(filter) || it.name.toLowerCase().includes(filter.slice(1));
  });

  useEffect(() => {
    if (!prompt.startsWith("/")) setMenuOpen(false);
  }, [prompt]);

  const pickCommand = (it: AgentCommandItem) => {
    const name = ensureSlash(it.name);
    onIntent({ action: "command", command: name });
    setPrompt("");
    setMenuOpen(false);
  };

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
      <div className="agent-intent-prompt-wrap">
        <label className="wide-field">
          意图
          <textarea
            rows={2}
            value={prompt}
            disabled={busy}
            data-testid="agent-intent-prompt"
            placeholder="只发意图，不拼系统 prompt（/ 打开命令）"
            onChange={(e) => {
              const v = e.target.value;
              setPrompt(v);
              if (v.startsWith("/")) setMenuOpen(true);
            }}
          />
        </label>
        {showMenu ? (
          <ul className="agent-command-menu" data-testid="agent-command-menu" role="listbox">
            {commandsQuery.isLoading ? (
              <li className="muted-line">加载命令…</li>
            ) : items.length === 0 ? (
              <li className="muted-line">无匹配命令</li>
            ) : (
              items.map((it) => (
                <li key={`${it.source}:${it.name}`}>
                  <button
                    type="button"
                    className="agent-command-menu-item"
                    disabled={busy}
                    data-testid={`agent-command-${it.name.replace(/^\//, "")}`}
                    onClick={() => pickCommand(it)}
                  >
                    <strong>{it.name}</strong>
                    <span className="muted-line">
                      {it.description} · {it.source}
                    </span>
                  </button>
                </li>
              ))
            )}
          </ul>
        ) : null}
      </div>
      <div className="row-actions">
        <button
          type="button"
          className="btn mini"
          disabled={busy}
          data-testid="agent-intent-slash"
          title="命令"
          onClick={() => {
            setMenuOpen(true);
            if (!prompt.startsWith("/")) setPrompt("/");
          }}
        >
          /
        </button>
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
            if (text.startsWith("/")) {
              const sp = text.indexOf(" ");
              const command = sp < 0 ? text : text.slice(0, sp);
              const args = sp < 0 ? "" : text.slice(sp + 1).trim();
              onIntent({ action: "command", command: ensureSlash(command), args: args || undefined });
            } else {
              onIntent({ action: "prompt", prompt: text });
            }
            setPrompt("");
            setMenuOpen(false);
          }}
        >
          发送
        </button>
      </div>
    </div>
  );
}
