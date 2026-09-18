import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
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
  scope?: "once" | "session";
  tool?: string;
};

type Props = {
  mode: "prompt" | "gate";
  busy?: boolean;
  /** Show Stop when run is generating / waiting. */
  canStop?: boolean;
  gateReason?: string;
  /** Tool name from open gate (for allow_session). */
  gateTool?: string;
  onIntent: (payload: IntentPayload) => void;
};

function ensureSlash(name: string): string {
  const n = name.trim();
  if (!n) return "/";
  return n.startsWith("/") ? n : `/${n}`;
}

function submitText(
  text: string,
  onIntent: (payload: IntentPayload) => void,
): boolean {
  const trimmed = text.trim();
  if (!trimmed) return false;
  if (trimmed.startsWith("/")) {
    const sp = trimmed.indexOf(" ");
    const command = sp < 0 ? trimmed : trimmed.slice(0, sp);
    const args = sp < 0 ? "" : trimmed.slice(sp + 1).trim();
    onIntent({ action: "command", command: ensureSlash(command), args: args || undefined });
  } else {
    onIntent({ action: "prompt", prompt: trimmed });
  }
  return true;
}

/** Thin composer: prompt input or gate takeover (DSH-aligned). */
export function IntentBar({
  mode,
  busy = false,
  canStop = false,
  gateReason,
  gateTool,
  onIntent,
}: Props) {
  const [prompt, setPrompt] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

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

  useEffect(() => {
    setActiveIndex(0);
  }, [filter, items.length]);

  /** Fill composer with command (DSH: pick does not auto-send; user may add args). */
  const pickCommand = (it: AgentCommandItem) => {
    const name = ensureSlash(it.name);
    setPrompt(`${name} `);
    setMenuOpen(false);
    requestAnimationFrame(() => {
      const el = textareaRef.current;
      if (!el) return;
      el.focus();
      const len = el.value.length;
      el.setSelectionRange(len, len);
    });
  };

  const sendCurrent = () => {
    if (busy) return;
    if (submitText(prompt, onIntent)) {
      setPrompt("");
      setMenuOpen(false);
    }
  };

  if (mode === "gate") {
    const tool = (gateTool || "").trim();
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
            data-testid="agent-intent-allow-once"
            onClick={() =>
              onIntent({
                action: "approve",
                scope: "once",
                tool: tool || undefined,
                reason: "allow once from IntentBar",
              })
            }
          >
            允许一次
          </button>
          <button
            type="button"
            className="btn mini ok"
            disabled={busy}
            data-testid="agent-intent-allow-session"
            title={tool ? `本会话允许 ${tool}` : "本会话允许此类工具"}
            onClick={() =>
              onIntent({
                action: "approve",
                scope: "session",
                tool: tool || undefined,
                reason: "allow session from IntentBar",
              })
            }
          >
            本会话允许此类
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
            data-testid="agent-intent-cancel"
            onClick={() => onIntent({ action: "cancel" })}
          >
            取消 Run
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
            ref={textareaRef}
            rows={2}
            value={prompt}
            disabled={busy}
            data-testid="agent-intent-prompt"
            placeholder="Enter 发送 · Shift+Enter 换行 · / 打开命令（点选填入，可加参数后再发）"
            onChange={(e) => {
              const v = e.target.value;
              setPrompt(v);
              if (v.startsWith("/")) setMenuOpen(true);
            }}
            onKeyDown={(e) => {
              if (showMenu && items.length > 0) {
                if (e.key === "ArrowDown") {
                  e.preventDefault();
                  setActiveIndex((i) => (i + 1) % items.length);
                  return;
                }
                if (e.key === "ArrowUp") {
                  e.preventDefault();
                  setActiveIndex((i) => (i - 1 + items.length) % items.length);
                  return;
                }
                if (e.key === "Escape") {
                  e.preventDefault();
                  setMenuOpen(false);
                  return;
                }
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  const it = items[Math.min(activeIndex, items.length - 1)];
                  if (it) pickCommand(it);
                  return;
                }
              }
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                sendCurrent();
              }
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
              items.map((it, idx) => (
                <li key={`${it.source}:${it.name}`}>
                  <button
                    type="button"
                    className={`agent-command-menu-item${idx === activeIndex ? " active" : ""}`}
                    disabled={busy}
                    role="option"
                    aria-selected={idx === activeIndex}
                    data-testid={`agent-command-${it.name.replace(/^\//, "")}`}
                    onMouseEnter={() => setActiveIndex(idx)}
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
          onClick={sendCurrent}
        >
          发送
        </button>
      </div>
    </div>
  );
}
