import { useQuery } from "@tanstack/react-query";
import {
  useEffect,
  useRef,
  useState,
  type FocusEvent as ReactFocusEvent,
  type ReactNode,
} from "react";
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
  /** Bump to force-close the + menu (Composer mutual exclusion). */
  closePlusSignal?: number;
  /** Fired when the + menu transitions closed → open. */
  onPlusOpened?: () => void;
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

/** APG menu typeahead: next item whose label starts with the typed character. */
function menuItemMatchingTypeahead(
  items: HTMLButtonElement[],
  key: string,
): HTMLButtonElement | undefined {
  if (key.length !== 1 || key === " ") return undefined;
  const ch = key.toLowerCase();
  const active = items.findIndex((el) => el === document.activeElement);
  const start = active < 0 ? 0 : active + 1;
  for (let i = 0; i < items.length; i++) {
    const idx = (start + i) % items.length;
    const label = (items[idx]!.textContent ?? "").replace(/\s+/g, " ").trim().toLowerCase();
    if (label.startsWith(ch)) return items[idx];
  }
  return undefined;
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
  closePlusSignal = 0,
  onPlusOpened,
  onIntent,
}: Props) {
  const [prompt, setPrompt] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const [plusOpen, setPlusOpen] = useState(false);
  const [suppressCommandMenu, setSuppressCommandMenu] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const plusMenuRef = useRef<HTMLDivElement>(null);
  const plusTriggerRef = useRef<HTMLButtonElement>(null);
  const plusFocusPreferLastRef = useRef(false);
  const commandMenuRef = useRef<HTMLDivElement>(null);
  const gateActionsRef = useRef<HTMLDivElement>(null);
  const plusMenuDomId = "agent-composer-plus-menu";
  const commandMenuDomId = "agent-command-menu";

  const commandOptionDomId = (name: string) =>
    `agent-command-option-${name.replace(/^\//, "").replace(/[^a-zA-Z0-9_-]/g, "-") || "item"}`;

  const commandsQuery = useQuery({
    queryKey: ["agent-commands"],
    queryFn: () => listAgentCommands(),
    enabled: menuOpen || prompt.startsWith("/"),
    staleTime: 60_000,
  });

  useEffect(() => {
    setSuppressCommandMenu(false);
  }, [prompt]);

  useEffect(() => {
    if (mode !== "gate") return;
    queueMicrotask(() => {
      gateActionsRef.current
        ?.querySelector<HTMLButtonElement>("button:not([disabled])")
        ?.focus();
    });
  }, [mode, gateReason, gateTool]);

  useEffect(() => {
    if (mode !== "gate") return;
    const onKey = (e: KeyboardEvent) => {
      if (
        e.key !== "ArrowLeft" &&
        e.key !== "ArrowRight" &&
        e.key !== "Home" &&
        e.key !== "End"
      ) {
        return;
      }
      const root = gateActionsRef.current;
      if (!root || !root.contains(document.activeElement)) return;
      const items = Array.from(root.querySelectorAll<HTMLButtonElement>("button")).filter(
        (el) => !el.disabled,
      );
      if (items.length === 0) return;
      e.preventDefault();
      if (e.key === "Home") {
        items[0]?.focus();
        return;
      }
      if (e.key === "End") {
        items[items.length - 1]?.focus();
        return;
      }
      const active = items.findIndex((el) => el === document.activeElement);
      const delta = e.key === "ArrowRight" ? 1 : -1;
      const next = active < 0 ? 0 : (active + delta + items.length) % items.length;
      items[next]?.focus();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [mode]);

  useEffect(() => {
    setPlusOpen(false);
  }, [closePlusSignal]);

  useEffect(() => {
    if (!plusOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        setPlusOpen(false);
        queueMicrotask(() => plusTriggerRef.current?.focus());
        return;
      }
      const root = plusMenuRef.current;
      if (!root) return;
      const items = Array.from(
        root.querySelectorAll<HTMLButtonElement>('[role="menuitem"]'),
      ).filter((el) => !el.disabled);
      if (items.length === 0) return;
      const focusItem = (el: HTMLButtonElement | undefined) => {
        el?.focus();
        el?.scrollIntoView?.({ block: "nearest" });
      };
      if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
        const hit = menuItemMatchingTypeahead(items, e.key);
        if (hit) {
          e.preventDefault();
          focusItem(hit);
        }
        return;
      }
      if (
        e.key !== "ArrowDown" &&
        e.key !== "ArrowUp" &&
        e.key !== "Home" &&
        e.key !== "End"
      ) {
        return;
      }
      e.preventDefault();
      if (e.key === "Home") {
        focusItem(items[0]);
        return;
      }
      if (e.key === "End") {
        focusItem(items[items.length - 1]);
        return;
      }
      const active = items.findIndex((el) => el === document.activeElement);
      const delta = e.key === "ArrowDown" ? 1 : -1;
      const next = active < 0 ? 0 : (active + delta + items.length) % items.length;
      focusItem(items[next]);
    };
    const onPointer = (e: PointerEvent) => {
      if (!plusMenuRef.current?.contains(e.target as Node)) {
        setPlusOpen(false);
      }
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    queueMicrotask(() => {
      const items = Array.from(
        plusMenuRef.current?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [],
      ).filter((el) => !el.disabled);
      const preferLast = plusFocusPreferLastRef.current;
      plusFocusPreferLastRef.current = false;
      const el = preferLast ? items[items.length - 1] : items[0];
      el?.focus();
      el?.scrollIntoView?.({ block: "nearest" });
    });
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [plusOpen]);

  const showMenu = !suppressCommandMenu && (menuOpen || (prompt.startsWith("/") && !busy));

  const dismissCommandMenu = (restoreFocus = false) => {
    setMenuOpen(false);
    setSuppressCommandMenu(true);
    if (restoreFocus) queueMicrotask(() => textareaRef.current?.focus());
  };

  /** DSH: leave the + seat → close (focus may already have moved). */
  const onPlusRootBlur = (e: ReactFocusEvent<HTMLDivElement>) => {
    if (!plusOpen) return;
    const next = e.relatedTarget;
    if (next instanceof Node && plusMenuRef.current?.contains(next)) return;
    setPlusOpen(false);
  };

  /** Slash listbox dismisses when focus leaves the prompt (not when clicking an option). */
  const onPromptBlur = (e: ReactFocusEvent<HTMLTextAreaElement>) => {
    if (!showMenu) return;
    const next = e.relatedTarget;
    const menuEl = document.getElementById(commandMenuDomId);
    if (next instanceof Node && menuEl?.contains(next)) return;
    dismissCommandMenu(false);
  };

  useEffect(() => {
    if (showMenu) setPlusOpen(false);
  }, [showMenu]);

  useEffect(() => {
    if (!showMenu) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        dismissCommandMenu(true);
      }
    };
    const onPointer = (e: PointerEvent) => {
      if (!commandMenuRef.current?.contains(e.target as Node)) {
        dismissCommandMenu(false);
      }
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [showMenu]);
  const filter = prompt.startsWith("/") ? prompt.toLowerCase() : "";
  const items: AgentCommandItem[] = (commandsQuery.data?.items ?? []).filter((it) => {
    if (!filter || filter === "/") return true;
    return it.name.toLowerCase().startsWith(filter) || it.name.toLowerCase().includes(filter.slice(1));
  });
  const safeActiveIndex =
    items.length === 0 ? 0 : Math.min(Math.max(activeIndex, 0), items.length - 1);
  const activeCommandOptionId =
    showMenu && items.length > 0 ? commandOptionDomId(items[safeActiveIndex]!.name) : undefined;

  useEffect(() => {
    if (!showMenu || !activeCommandOptionId) return;
    document.getElementById(activeCommandOptionId)?.scrollIntoView?.({ block: "nearest" });
  }, [showMenu, activeCommandOptionId]);

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
        <p
          className="muted-line"
          data-testid="agent-intent-gate-reason"
          role="status"
          aria-live="polite"
        >
          {gateReason || "Run 等待审批"}
        </p>
        <div
          className="row-actions"
          role="group"
          aria-label="审批操作"
          ref={gateActionsRef}
        >
          <button
            type="button"
            className="btn mini ok"
            disabled={busy}
            data-testid="agent-intent-allow-once"
            title={tool ? `允许一次执行 ${tool}` : "允许一次"}
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
            title="拒绝本次工具调用"
            onClick={() => onIntent({ action: "reject", reason: "rejected from IntentBar" })}
          >
            拒绝
          </button>
          <button
            type="button"
            className="btn mini"
            disabled={busy}
            data-testid="agent-intent-cancel"
            title="取消当前 Run"
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
      <div className="agent-intent-prompt-wrap" ref={commandMenuRef}>
        {queueItems.length > 0 ? (
          <p
            className="agent-intent-queue-chip muted-line"
            data-testid="agent-intent-queue-chip"
            role="status"
            aria-live="polite"
          >
            队列 {queueItems.length}
            {queueItems[0] ? ` · ${queueItems[0].trim().slice(0, 24)}` : ""}
            {" · 停止只结束当前"}
          </p>
        ) : null}
        {canStop ? (
          <p
            className="muted-line"
            data-testid="agent-intent-steer-hint"
            role="status"
            aria-live="polite"
          >
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
            aria-haspopup="listbox"
            aria-controls={showMenu ? commandMenuDomId : undefined}
            aria-expanded={showMenu}
            aria-autocomplete="list"
            aria-activedescendant={activeCommandOptionId}
            placeholder={canStop ? COMPOSER_PLACEHOLDER_STEER : COMPOSER_PLACEHOLDER_DEFAULT}
            onChange={(e) => {
              const v = e.target.value;
              setPrompt(v);
              if (v.startsWith("/")) setMenuOpen(true);
            }}
            onBlur={onPromptBlur}
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
                if (e.key === "Home") {
                  e.preventDefault();
                  setActiveIndex(0);
                  return;
                }
                if (e.key === "End") {
                  e.preventDefault();
                  setActiveIndex(items.length - 1);
                  return;
                }
                if (e.key === "Escape") {
                  e.preventDefault();
                  dismissCommandMenu(true);
                  return;
                }
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  const it = items[safeActiveIndex];
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
              <div
                className="agent-composer-plus-wrap"
                ref={plusMenuRef}
                onBlur={onPlusRootBlur}
              >
                <button
                  ref={plusTriggerRef}
                  type="button"
                  className="agent-composer-plus"
                  disabled={busy}
                  data-testid="agent-composer-plus"
                  aria-label="添加附件或调用指令"
                  aria-haspopup="menu"
                  aria-expanded={plusOpen}
                  aria-controls={plusOpen ? plusMenuDomId : undefined}
                  title="添加附件或调用指令"
                  onClick={() => {
                    if (plusOpen) {
                      setPlusOpen(false);
                      return;
                    }
                    dismissCommandMenu();
                    setPlusOpen(true);
                    onPlusOpened?.();
                  }}
                  onKeyDown={(e) => {
                    if (busy || plusOpen) return;
                    if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
                    e.preventDefault();
                    plusFocusPreferLastRef.current = e.key === "ArrowUp";
                    dismissCommandMenu();
                    setPlusOpen(true);
                    onPlusOpened?.();
                  }}
                >
                  +
                </button>
                {plusOpen ? (
                  <ul
                    id={plusMenuDomId}
                    className="agent-composer-plus-menu"
                    data-testid="agent-composer-plus-menu"
                    role="menu"
                    aria-orientation="vertical"
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
                          queueMicrotask(() => textareaRef.current?.focus());
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
          <ul
            id={commandMenuDomId}
            className="agent-command-menu"
            data-testid="agent-command-menu"
            role="listbox"
            aria-orientation="vertical"
            aria-label="斜杠命令"
            aria-busy={commandsQuery.isLoading || undefined}
          >
            {commandsQuery.isLoading ? (
              <li className="muted-line" role="status">
                加载命令…
              </li>
            ) : items.length === 0 ? (
              <li className="muted-line" role="status">
                无匹配命令
              </li>
            ) : (
              items.map((it, idx) => (
                <li key={`${it.source}:${it.name}`}>
                  <button
                    id={commandOptionDomId(it.name)}
                    type="button"
                    className={`agent-command-menu-item${idx === safeActiveIndex ? " active" : ""}`}
                    disabled={busy}
                    role="option"
                    aria-selected={idx === safeActiveIndex}
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
