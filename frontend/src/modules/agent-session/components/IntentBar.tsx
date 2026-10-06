import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  listAgentCommands,
  type AgentCommandItem,
  type SessionIntentAction,
} from "../api/session.api";
import {
  COMPOSER_PLACEHOLDER_DEFAULT,
  COMPOSER_PLACEHOLDER_STEER,
} from "../reasoningEffort";

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
  /** Follow-up prompts already queued on the session (chip only; Stop does not clear them). */
  queueItems?: string[];
  /** DSH InputBar: seats / chips left of send, inside the draft card. */
  toolbarStart?: ReactNode;
  /** DSH trailing cluster (rarely used; extras go in the + menu). */
  toolbarEnd?: ReactNode;
  /** Extra rows in the left + menu (Tools / MCP / Skills). */
  plusItems?: ReactNode;
  onIntent: (payload: IntentPayload) => void;
};

function ensureSlash(name: string): string {
  const n = name.trim();
  if (!n) return "/";
  return n.startsWith("/") ? n : `/${n}`;
}

/** Bare `/` only opens the command menu — not a submittable intent. */
function isBareSlash(text: string): boolean {
  return text.trim() === "/";
}

function isDestructiveCommand(command: string): boolean {
  return ensureSlash(command).toLowerCase() === "/clear";
}

function submitText(
  text: string,
  onIntent: (payload: IntentPayload) => void,
  opts?: { steer?: boolean; queue?: boolean },
): boolean {
  const trimmed = text.trim();
  if (!trimmed || isBareSlash(trimmed)) return false;
  if (trimmed.startsWith("/")) {
    const sp = trimmed.indexOf(" ");
    const command = sp < 0 ? trimmed : trimmed.slice(0, sp);
    const args = sp < 0 ? "" : trimmed.slice(sp + 1).trim();
    const cmd = ensureSlash(command);
    if (isDestructiveCommand(cmd)) {
      const ok = window.confirm(
        "确认执行 /clear？将清空当前会话的对话上下文（不可恢复）。",
      );
      if (!ok) return false;
    }
    onIntent({ action: "command", command: cmd, args: args || undefined });
  } else if (opts?.queue) {
    onIntent({ action: "queue", prompt: trimmed });
  } else {
    onIntent({
      action: opts?.steer ? "steer" : "prompt",
      prompt: trimmed,
    });
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
  queueItems = [],
  toolbarStart,
  toolbarEnd,
  plusItems,
  onIntent,
}: Props) {
  const [prompt, setPrompt] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const [plusOpen, setPlusOpen] = useState(false);
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
  const bareSlash = isBareSlash(prompt);
  const canSubmitPrompt = Boolean(prompt.trim()) && !bareSlash;
  const trimmedPrompt = prompt.trim();
  const destructiveClear =
    trimmedPrompt.startsWith("/") &&
    isDestructiveCommand(
      trimmedPrompt.includes(" ") ? trimmedPrompt.slice(0, trimmedPrompt.indexOf(" ")) : trimmedPrompt,
    );
  const submitBlockedTitle = !prompt.trim()
    ? "需要输入内容"
    : bareSlash
      ? "选择命令或继续输入"
      : null;
  const sendTitle = busy
    ? "处理中"
    : submitBlockedTitle
      ? submitBlockedTitle
      : destructiveClear
        ? "执行 /clear（需确认）"
        : canStop
          ? "续写"
          : "发送";

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
    if (submitText(prompt, onIntent, { steer: canStop })) {
      setPrompt("");
      setMenuOpen(false);
    }
  };

  const queueCurrent = () => {
    if (busy || !canStop) return;
    if (submitText(prompt, onIntent, { queue: true })) {
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
    <div
      className="agent-intent-bar"
      data-testid="agent-intent-bar"
      data-mode="prompt"
      data-steer={canStop ? "true" : "false"}
      data-queue-count={queueItems.length}
    >
      <div className="agent-intent-prompt-wrap">
        {queueItems.length > 0 ? (
          <p className="agent-intent-queue-chip muted-line" data-testid="agent-intent-queue-chip">
            队列 {queueItems.length}
            {queueItems[0] ? ` · ${queueItems[0].trim().slice(0, 24)}` : ""}
            {" · 停止只结束当前"}
          </p>
        ) : null}
        {canStop ? (
          <p className="muted-line" data-testid="agent-intent-steer-hint">
            运行中：发送将打断当前生成并以新提示续写（Steer）
          </p>
        ) : null}
        <div className="agent-intent-field" data-testid="agent-intent-field">
          <label className="sr-only" htmlFor="agent-intent-prompt">
            {canStop ? "Steer" : "意图"}
          </label>
          <textarea
            id="agent-intent-prompt"
            ref={textareaRef}
            className="agent-intent-input"
            rows={2}
            value={prompt}
            disabled={busy}
            data-testid="agent-intent-prompt"
            aria-label={canStop ? "Steer" : "意图"}
            placeholder={canStop ? COMPOSER_PLACEHOLDER_STEER : COMPOSER_PLACEHOLDER_DEFAULT}
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
                if (e.altKey && canStop) {
                  queueCurrent();
                  return;
                }
                sendCurrent();
              }
            }}
          />
          <div className="agent-intent-toolbar" data-testid="agent-intent-toolbar">
            <div className="agent-composer-tools">
              <div className="agent-composer-plus-wrap">
                <button
                  type="button"
                  className="agent-composer-plus"
                  disabled={busy}
                  data-testid="agent-composer-plus"
                  aria-label="添加附件或调用指令"
                  aria-haspopup="menu"
                  aria-expanded={plusOpen}
                  title="添加附件或调用指令"
                  onClick={() => setPlusOpen((open) => !open)}
                >
                  +
                </button>
                {plusOpen ? (
                  <ul
                    className="agent-composer-plus-menu"
                    data-testid="agent-composer-plus-menu"
                    role="menu"
                    onClick={(e) => {
                      if ((e.target as HTMLElement).closest("button")) setPlusOpen(false);
                    }}
                  >
                    <li>
                      <button
                        type="button"
                        className="agent-composer-plus-item"
                        disabled={busy}
                        data-testid="agent-intent-slash"
                        role="menuitem"
                        title="命令"
                        onClick={() => {
                          setPlusOpen(false);
                          setMenuOpen(true);
                          if (!prompt.startsWith("/")) setPrompt("/");
                        }}
                      >
                        命令
                      </button>
                    </li>
                    {plusItems}
                  </ul>
                ) : null}
              </div>
            </div>
            {toolbarStart}
            <div className="agent-composer-trailing">
              {toolbarEnd}
              <div className="row-actions">
              {canStop ? (
                <>
                  <button
                    type="button"
                    className="btn mini err"
                    disabled={busy}
                    data-testid="agent-intent-stop"
                    title="只停止当前生成或 Run，不清空 follow-up 队列"
                    onClick={() => onIntent({ action: "stop" })}
                  >
                    停止
                  </button>
                  <button
                    type="button"
                    className="btn mini"
                    disabled={!canSubmitPrompt || busy}
                    data-testid="agent-intent-queue"
                    title={
                      submitBlockedTitle
                        ? bareSlash
                          ? "选择命令或继续输入后再排队"
                          : "需要输入内容后再排队"
                        : "当前结束后再发送，不打断 Steer"
                    }
                    onClick={queueCurrent}
                  >
                    排队
                  </button>
                </>
              ) : null}
              <button
                type="button"
                className="btn mini ok"
                disabled={busy || !canSubmitPrompt}
                data-testid="agent-intent-send"
                title={sendTitle}
                onClick={sendCurrent}
              >
                {canStop ? "续写" : "发送"}
              </button>
              </div>
            </div>
          </div>
        </div>
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
                    title={busy ? "会话忙" : `填入 ${ensureSlash(it.name)}（不自动发送）`}
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
    </div>
  );
}
