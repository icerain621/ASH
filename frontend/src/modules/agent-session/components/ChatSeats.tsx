import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type FocusEvent as ReactFocusEvent,
  type KeyboardEvent as ReactKeyboardEvent,
} from "react";
import {
  listAgentModels,
  updateSession,
  type AgentSessionView,
  type PermissionMode,
} from "../api/session.api";
import {
  FULL_ACCESS_CONFIRM_MESSAGE,
  PERMISSION_OPTIONS,
  permissionOptionLabel,
  permissionStripLabel,
  resolvePermissionMode,
} from "../permissionModeLabels";
import { listModelProviders } from "@/modules/platform/api/platform.api";
import {
  REASONING_EFFORTS,
  isReasoningEffortId,
  readReasoningEffort,
  reasoningEffortLabel,
  writeReasoningEffort,
  type ReasoningEffortId,
} from "../reasoningEffort";

type Props = {
  session: AgentSessionView | null;
  disabled?: boolean;
  /** Bump to force-close the model menu (Composer mutual exclusion). */
  closeMenuSignal?: number;
  /** Fired when the model menu transitions closed → open. */
  onMenuOpened?: () => void;
};

function providerStatusLabel(status: string | null | undefined) {
  const s = (status ?? "").trim().toLowerCase();
  if (s === "available" || s === "configured") return "可用";
  if (s === "not_configured" || s === "") return "未配置";
  return status ?? "";
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

/** Model / permission / plan chips for the composer toolbar (DSH InputBar). */
export function ChatSeats({
  session,
  disabled = false,
  closeMenuSignal = 0,
  onMenuOpened,
}: Props) {
  const qc = useQueryClient();
  const [planDraft, setPlanDraft] = useState(session?.planId ?? "");
  const [effort, setEffort] = useState<ReasoningEffortId>(() => {
    const fromSession = session?.reasoningEffort?.trim();
    if (fromSession && isReasoningEffortId(fromSession)) return fromSession;
    return readReasoningEffort(session?.id);
  });
  const [modelMenu, setModelMenu] = useState<"closed" | "root" | "model" | "effort">("closed");
  const [permMenuOpen, setPermMenuOpen] = useState(false);
  const modelMenuRef = useRef<HTMLDivElement>(null);
  const modelTriggerRef = useRef<HTMLButtonElement>(null);
  const permMenuRef = useRef<HTMLDivElement>(null);
  const permTriggerRef = useRef<HTMLButtonElement>(null);
  const skipPlanBlurRef = useRef(false);
  const permFocusPreferLastRef = useRef(false);
  const prevModelMenuRef = useRef(modelMenu);
  const modelMenuDomId = "agent-chat-seat-model-menu";
  const permMenuDomId = "agent-chat-seat-permission-menu";

  const closeModelMenu = (restoreFocus = false) => {
    setModelMenu("closed");
    if (restoreFocus) queueMicrotask(() => modelTriggerRef.current?.focus());
  };

  const closePermMenu = (restoreFocus = false) => {
    setPermMenuOpen(false);
    if (restoreFocus) queueMicrotask(() => permTriggerRef.current?.focus());
  };

  /** DSH ModelSelect: dismiss when focus leaves the seat root. */
  const onPermRootBlur = (e: ReactFocusEvent<HTMLDivElement>) => {
    if (!permMenuOpen) return;
    const next = e.relatedTarget;
    if (next instanceof Node && permMenuRef.current?.contains(next)) return;
    closePermMenu(false);
  };
  const onModelRootBlur = (e: ReactFocusEvent<HTMLDivElement>) => {
    if (modelMenu === "closed") return;
    const next = e.relatedTarget;
    if (next instanceof Node && modelMenuRef.current?.contains(next)) return;
    closeModelMenu(false);
  };

  const openPermMenu = () => {
    closeModelMenu(false);
    setPermMenuOpen(true);
    onMenuOpened?.();
  };

  const openModelMenuRoot = () => {
    closePermMenu(false);
    setModelMenu("root");
    onMenuOpened?.();
  };

  useEffect(() => {
    const prev = prevModelMenuRef.current;
    prevModelMenuRef.current = modelMenu;
    if (modelMenu === "root" && (prev === "model" || prev === "effort")) {
      const testId =
        prev === "effort" ? "agent-chat-seat-effort-open" : "agent-chat-seat-model-open";
      queueMicrotask(() => {
        modelMenuRef.current
          ?.querySelector<HTMLButtonElement>(`[data-testid="${testId}"]`)
          ?.focus();
      });
      return;
    }
    if ((modelMenu === "model" || modelMenu === "effort") && prev === "root") {
      queueMicrotask(() => {
        modelMenuRef.current
          ?.querySelector<HTMLButtonElement>('[role="menuitemradio"]')
          ?.focus();
      });
    }
  }, [modelMenu]);

  useEffect(() => {
    setPlanDraft(session?.planId ?? "");
    const fromSession = session?.reasoningEffort?.trim();
    setEffort(
      fromSession && isReasoningEffortId(fromSession)
        ? fromSession
        : readReasoningEffort(session?.id),
    );
    setModelMenu("closed");
    setPermMenuOpen(false);
  }, [session?.id, session?.planId, session?.reasoningEffort]);

  useEffect(() => {
    setModelMenu("closed");
    setPermMenuOpen(false);
  }, [closeMenuSignal]);

  useEffect(() => {
    if (!permMenuOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        closePermMenu(true);
        return;
      }
      const root = permMenuRef.current;
      if (!root) return;
      const items = Array.from(
        root.querySelectorAll<HTMLButtonElement>('[role="menuitemradio"]'),
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
      if (!permMenuRef.current?.contains(e.target as Node)) {
        closePermMenu(false);
      }
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    queueMicrotask(() => {
      const items = Array.from(
        permMenuRef.current?.querySelectorAll<HTMLButtonElement>('[role="menuitemradio"]') ??
          [],
      ).filter((el) => !el.disabled);
      const preferLast = permFocusPreferLastRef.current;
      permFocusPreferLastRef.current = false;
      const el = preferLast ? items[items.length - 1] : items[0];
      el?.focus();
      el?.scrollIntoView?.({ block: "nearest" });
    });
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [permMenuOpen]);

  useEffect(() => {
    if (modelMenu === "closed") return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        if (modelMenu === "model" || modelMenu === "effort") {
          setModelMenu("root");
          return;
        }
        closeModelMenu(true);
        return;
      }
      if (e.key === "ArrowRight" && modelMenu === "root") {
        const active = document.activeElement;
        if (!(active instanceof HTMLElement) || !modelMenuRef.current?.contains(active)) return;
        const tid = active.getAttribute("data-testid");
        if (tid === "agent-chat-seat-model-open") {
          e.preventDefault();
          setModelMenu("model");
          return;
        }
        if (tid === "agent-chat-seat-effort-open") {
          e.preventDefault();
          setModelMenu("effort");
          return;
        }
        return;
      }
      if (e.key === "ArrowLeft") {
        if (modelMenu === "model" || modelMenu === "effort") {
          e.preventDefault();
          setModelMenu("root");
          return;
        }
        if (modelMenu === "root") {
          e.preventDefault();
          closeModelMenu(true);
          return;
        }
      }
      const root = modelMenuRef.current;
      if (!root) return;
      const items = Array.from(
        root.querySelectorAll<HTMLButtonElement>('[role="menuitem"], [role="menuitemradio"]'),
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
      if (!modelMenuRef.current?.contains(e.target as Node)) {
        closeModelMenu(false);
      }
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [modelMenu]);

  const modelsQuery = useQuery({
    queryKey: ["agent-models"],
    queryFn: () => listAgentModels(),
    staleTime: 60_000,
  });

  const providersQuery = useQuery({
    queryKey: ["model-router-providers"],
    queryFn: () => listModelProviders(),
    staleTime: 60_000,
  });

  const providerHealth = useMemo(() => {
    const items = providersQuery.data?.items ?? [];
    if (items.length === 0) return "目录空";
    return items
      .map((p) => `${p.role || p.id}:${providerStatusLabel(p.status)}`)
      .join(" · ");
  }, [providersQuery.data?.items]);

  const patchMut = useMutation({
    mutationFn: (body: {
      providerKind?: string;
      permissionMode?: PermissionMode;
      planId?: string;
      reasoningEffort?: ReasoningEffortId;
    }) => {
      if (!session?.id) throw new Error("session not ready");
      return updateSession(session.id, body);
    },
    onSuccess: (updated) => {
      void qc.invalidateQueries({ queryKey: ["agent-session", updated.id] });
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
    },
    onError: () => {
      setPlanDraft(session?.planId ?? "");
      const fromSession = session?.reasoningEffort?.trim();
      setEffort(
        fromSession && isReasoningEffortId(fromSession)
          ? fromSession
          : readReasoningEffort(session?.id),
      );
    },
  });

  const seatLocked = disabled || !session?.id;
  const busy = seatLocked || patchMut.isPending;
  const providerKind = session?.providerKind || modelsQuery.data?.items?.[0]?.id || "static";
  const modelItems = modelsQuery.data?.items?.length
    ? modelsQuery.data.items
    : [{ id: providerKind, label: providerKind, providerKind }];
  const modelLabel =
    modelItems.find((m) => m.id === providerKind)?.label || providerKind;
  const chatModel = (modelsQuery.data?.chatModel || "").trim();
  const effortLabel = reasoningEffortLabel(effort);
  const modelTriggerAria = chatModel
    ? `选择模型，当前 ${modelLabel}，Chat ${chatModel}，推理等级 ${effortLabel}`
    : `选择模型，当前 ${modelLabel}，推理等级 ${effortLabel}`;
  const modelTriggerTitle = seatLocked
    ? "会话未就绪或保存中"
    : chatModel
      ? `${modelLabel} · ${chatModel} · ${effortLabel}`
      : `${modelLabel} · ${effortLabel}`;
  const permissionMode = resolvePermissionMode(session?.permissionMode);
  const permissionLabel = permissionStripLabel(permissionMode);
  const commitPlanDraft = () => {
    const next = planDraft.trim();
    const current = session?.planId || "";
    if (next === current) return;
    if (!next && !current) return;
    patchMut.mutate({ planId: next });
  };
  const revertPlanDraft = () => {
    setPlanDraft(session?.planId ?? "");
  };
  const onPlanKeyDown = (e: ReactKeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      commitPlanDraft();
      skipPlanBlurRef.current = true;
      e.currentTarget.blur();
      return;
    }
    if (e.key === "Escape") {
      e.preventDefault();
      revertPlanDraft();
      skipPlanBlurRef.current = true;
      e.currentTarget.blur();
    }
  };
  const onPlanBlur = () => {
    if (skipPlanBlurRef.current) {
      skipPlanBlurRef.current = false;
      return;
    }
    commitPlanDraft();
  };
  const rawPatchError =
    patchMut.isError && patchMut.error instanceof Error
      ? patchMut.error.message
      : patchMut.isError
        ? "保存失败"
        : "";
  const patchError = /^plan not found$/i.test(rawPatchError.trim())
    ? "未找到该 Plan，请检查 planId"
    : rawPatchError;

  return (
    <div className="agent-chat-seats" data-testid="agent-chat-seats">
      {patchError ? (
        <p className="error-text" role="alert" data-testid="agent-chat-seats-error">
          {patchError}
        </p>
      ) : null}
      <div className="agent-chat-seats-modes">
        <div
          className="agent-chat-seat agent-chat-seat-perm"
          ref={permMenuRef}
          onBlur={onPermRootBlur}
        >
          <button
            ref={permTriggerRef}
            type="button"
            className="agent-chat-seat-perm-trigger"
            data-testid="agent-chat-seat-permission"
            data-permission={permissionMode}
            disabled={seatLocked}
            aria-haspopup="menu"
            aria-expanded={permMenuOpen}
            aria-controls={permMenuOpen ? permMenuDomId : undefined}
            aria-label={`审批模式，当前 ${permissionOptionLabel(permissionMode)}`}
            title={
              seatLocked
                ? "会话未就绪或保存中"
                : `${permissionOptionLabel(permissionMode)} · 本会话审批（完全访问需确认）`
            }
            onClick={() => {
              if (permMenuOpen) {
                closePermMenu(false);
                return;
              }
              openPermMenu();
            }}
            onKeyDown={(e) => {
              if (seatLocked || permMenuOpen) return;
              if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
              e.preventDefault();
              permFocusPreferLastRef.current = e.key === "ArrowUp";
              openPermMenu();
            }}
          >
            <span>{permissionLabel}</span>
            <span
              className={
                permMenuOpen
                  ? "agent-chat-seat-model-trigger-chevron agent-chat-seat-model-trigger-chevron--open"
                  : "agent-chat-seat-model-trigger-chevron"
              }
              aria-hidden="true"
            >
              ▾
            </span>
          </button>
          {permMenuOpen ? (
            <div
              id={permMenuDomId}
              className="agent-chat-seat-perm-menu"
              data-testid="agent-chat-seat-permission-menu"
              role="menu"
              aria-orientation="vertical"
              aria-label="审批模式"
            >
              {PERMISSION_OPTIONS.map((o) => {
                const selected = o.value === permissionMode;
                return (
                  <button
                    key={o.value}
                    type="button"
                    className={
                      selected
                        ? "agent-chat-seat-model-row agent-chat-seat-model-row--selected"
                        : "agent-chat-seat-model-row"
                    }
                    data-testid={`agent-chat-seat-permission-${o.value}`}
                    role="menuitemradio"
                    aria-checked={selected}
                    disabled={busy}
                    title={
                      o.value === "full"
                        ? "完全访问（需确认）；危险工具可跳过逐步批准"
                        : o.value === "workspace-write"
                          ? "自动审批工作区内写操作"
                          : "询问审批：危险工具需人工确认"
                    }
                    onClick={() => {
                      if (o.value !== permissionMode) {
                        if (o.value === "full") {
                          const ok = window.confirm(FULL_ACCESS_CONFIRM_MESSAGE);
                          if (!ok) return;
                        }
                        patchMut.mutate({ permissionMode: o.value });
                      }
                      closePermMenu(true);
                    }}
                  >
                    <span>{o.label}</span>
                    {selected ? (
                      <span className="agent-chat-seat-model-check" aria-hidden="true">
                        ✓
                      </span>
                    ) : null}
                  </button>
                );
              })}
            </div>
          ) : null}
        </div>
        <label className="agent-chat-seat agent-chat-seat-plan">
          <span className="sr-only">Plan</span>
          <span
            className={
              session?.planId
                ? "agent-chat-seat-plan-wrap"
                : "agent-chat-seat-plan-wrap muted-line"
            }
            data-testid={session?.planId ? undefined : "agent-chat-seat-plan-empty"}
          >
            <input
              data-testid="agent-chat-seat-plan"
              disabled={busy}
              aria-label="绑定 Plan"
              title={
                busy
                  ? "会话未就绪或保存中"
                  : session?.planId
                    ? "绑定 / 修改 planId（Enter 保存，Esc 还原）"
                    : "可选：绑定 planId（Enter 或失焦保存，Esc 清空草稿）"
              }
              value={planDraft}
              placeholder="planId"
              onChange={(e) => setPlanDraft(e.target.value)}
              onKeyDown={onPlanKeyDown}
              onBlur={onPlanBlur}
            />
            {session?.planId ? (
              <button
                type="button"
                className="agent-chat-seat-plan-clear"
                data-testid="agent-chat-seat-plan-clear"
                disabled={busy}
                aria-label="清除 Plan 绑定"
                title="清除 Plan 绑定"
                onClick={() => {
                  skipPlanBlurRef.current = true;
                  setPlanDraft("");
                  patchMut.mutate({ planId: "" });
                }}
              >
                ×
              </button>
            ) : null}
          </span>
        </label>
      </div>
      <div className="agent-chat-seats-model" ref={modelMenuRef} onBlur={onModelRootBlur}>
        <button
          ref={modelTriggerRef}
          type="button"
          className="agent-chat-seat-model-trigger"
          data-testid="agent-chat-seat-model-trigger"
          disabled={seatLocked}
          aria-haspopup="menu"
          aria-expanded={modelMenu !== "closed"}
          aria-controls={modelMenu !== "closed" ? modelMenuDomId : undefined}
          aria-label={modelTriggerAria}
          title={modelTriggerTitle}
          onClick={() => {
            if (modelMenu !== "closed") {
              closeModelMenu(false);
              return;
            }
            openModelMenuRoot();
          }}
          onKeyDown={(e) => {
            if (seatLocked || modelMenu !== "closed") return;
            if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
            e.preventDefault();
            openModelMenuRoot();
            const preferLast = e.key === "ArrowUp";
            queueMicrotask(() => {
              const items = Array.from(
                modelMenuRef.current?.querySelectorAll<HTMLButtonElement>(
                  '[role="menuitem"], [role="menuitemradio"]',
                ) ?? [],
              ).filter((el) => !el.disabled);
              if (items.length === 0) return;
              const el = preferLast ? items[items.length - 1] : items[0];
              el?.focus();
              el?.scrollIntoView?.({ block: "nearest" });
            });
          }}
        >
          <span data-testid="agent-chat-seat-model-name">{modelLabel}</span>
          {chatModel ? (
            <span className="agent-chat-seat-chat-model" data-testid="agent-chat-seat-chat-model">
              {chatModel}
            </span>
          ) : null}
          <span className="agent-chat-seat-effort-caption">{effortLabel}</span>
          <span
            className={
              modelMenu !== "closed"
                ? "agent-chat-seat-model-trigger-chevron agent-chat-seat-model-trigger-chevron--open"
                : "agent-chat-seat-model-trigger-chevron"
            }
            aria-hidden="true"
          >
            ▾
          </span>
        </button>
        <select
          className="sr-only"
          aria-hidden="true"
          tabIndex={-1}
          data-testid="agent-chat-seat-model"
          disabled={busy}
          title={busy ? "会话未就绪或保存中" : "选择本会话模型 / provider"}
          value={providerKind}
          onChange={(e) => {
            const next = e.target.value.trim();
            if (!next || next === providerKind) return;
            patchMut.mutate({ providerKind: next });
          }}
        >
          {modelItems.map((m) => (
            <option key={m.id} value={m.id}>
              {m.label || m.id}
            </option>
          ))}
        </select>
        <span className="sr-only" data-testid="agent-chat-seat-model-health">
          {providersQuery.isLoading ? "探针…" : providerHealth}
        </span>
        {modelMenu !== "closed" ? (
          <div
            id={modelMenuDomId}
            className="agent-chat-seat-model-menu"
            data-testid="agent-chat-seat-model-menu"
            role="menu"
            aria-orientation="vertical"
            aria-label="模型与推理等级"
            aria-busy={modelsQuery.isLoading || patchMut.isPending}
          >
            {modelMenu === "root" ? (
              <>
                <button
                  type="button"
                  className="agent-chat-seat-model-row"
                  data-testid="agent-chat-seat-model-open"
                  role="menuitem"
                  aria-haspopup="menu"
                  onClick={() => setModelMenu("model")}
                >
                  <span>模型</span>
                  <span className="agent-chat-seat-model-row-value">
                    {chatModel ? `${modelLabel} · ${chatModel}` : modelLabel}
                    <span className="agent-chat-seat-model-chevron" aria-hidden="true">
                      ›
                    </span>
                  </span>
                </button>
                <button
                  type="button"
                  className="agent-chat-seat-model-row"
                  data-testid="agent-chat-seat-effort-open"
                  role="menuitem"
                  aria-haspopup="menu"
                  onClick={() => setModelMenu("effort")}
                >
                  <span>推理等级</span>
                  <span className="agent-chat-seat-model-row-value">
                    {effortLabel}
                    <span className="agent-chat-seat-model-chevron" aria-hidden="true">
                      ›
                    </span>
                  </span>
                </button>
              </>
            ) : null}
            {modelMenu === "model" ? (
              <>
                {chatModel ? (
                  <p
                    className="agent-chat-seat-model-hint"
                    data-testid="agent-chat-seat-chat-model-hint"
                    role="note"
                  >
                    Chat · {chatModel}
                  </p>
                ) : null}
                {modelItems.map((m) => {
                  const selected = m.id === providerKind;
                  return (
                    <button
                      key={m.id}
                      type="button"
                      className={
                        selected
                          ? "agent-chat-seat-model-row agent-chat-seat-model-row--selected"
                          : "agent-chat-seat-model-row"
                      }
                      data-testid={`agent-chat-seat-model-pick-${m.id}`}
                      role="menuitemradio"
                      aria-checked={selected}
                      disabled={busy}
                      onClick={() => {
                        if (m.id !== providerKind) patchMut.mutate({ providerKind: m.id });
                        closeModelMenu(true);
                      }}
                    >
                      <span>{m.label || m.id}</span>
                      {selected ? (
                        <span className="agent-chat-seat-model-check" aria-hidden="true">
                          ✓
                        </span>
                      ) : null}
                    </button>
                  );
                })}
              </>
            ) : null}
            {modelMenu === "effort"
              ? REASONING_EFFORTS.map((item) => {
                  const selected = item.id === effort;
                  return (
                    <button
                      key={item.id}
                      type="button"
                      className={
                        selected
                          ? "agent-chat-seat-model-row agent-chat-seat-model-row--selected"
                          : "agent-chat-seat-model-row"
                      }
                      data-testid={`agent-chat-seat-effort-${item.id}`}
                      role="menuitemradio"
                      aria-checked={selected}
                      onClick={() => {
                        if (item.id !== effort) {
                          if (session?.id) writeReasoningEffort(session.id, item.id);
                          setEffort(item.id);
                          if (session?.id) patchMut.mutate({ reasoningEffort: item.id });
                        }
                        closeModelMenu(true);
                      }}
                    >
                      <span>{item.label}</span>
                      {selected ? (
                        <span className="agent-chat-seat-model-check" aria-hidden="true">
                          ✓
                        </span>
                      ) : null}
                    </button>
                  );
                })
              : null}
          </div>
        ) : null}
      </div>
    </div>
  );
}
