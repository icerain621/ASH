import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import {
  listAgentModels,
  updateSession,
  type AgentSessionView,
  type PermissionMode,
} from "../api/session.api";
import {
  FULL_ACCESS_CONFIRM_MESSAGE,
  PERMISSION_OPTIONS,
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
};

function providerStatusLabel(status: string | null | undefined) {
  const s = (status ?? "").trim().toLowerCase();
  if (s === "available" || s === "configured") return "可用";
  if (s === "not_configured" || s === "") return "未配置";
  return status ?? "";
}

/** Model / permission / plan chips for the composer toolbar (DSH InputBar). */
export function ChatSeats({ session, disabled = false }: Props) {
  const qc = useQueryClient();
  const [planDraft, setPlanDraft] = useState(session?.planId ?? "");
  const [effort, setEffort] = useState<ReasoningEffortId>(() => {
    const fromSession = session?.reasoningEffort?.trim();
    if (fromSession && isReasoningEffortId(fromSession)) return fromSession;
    return readReasoningEffort(session?.id);
  });
  const [modelMenu, setModelMenu] = useState<"closed" | "root" | "model" | "effort">("closed");

  useEffect(() => {
    setPlanDraft(session?.planId ?? "");
    const fromSession = session?.reasoningEffort?.trim();
    setEffort(
      fromSession && isReasoningEffortId(fromSession)
        ? fromSession
        : readReasoningEffort(session?.id),
    );
    setModelMenu("closed");
  }, [session?.id, session?.planId, session?.reasoningEffort]);

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

  const busy = disabled || !session?.id || patchMut.isPending;
  const providerKind = session?.providerKind || modelsQuery.data?.items?.[0]?.id || "static";
  const modelItems = modelsQuery.data?.items?.length
    ? modelsQuery.data.items
    : [{ id: providerKind, label: providerKind, providerKind }];
  const modelLabel =
    modelItems.find((m) => m.id === providerKind)?.label || providerKind;
  const permissionMode = resolvePermissionMode(session?.permissionMode);
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
        <label className="agent-chat-seat">
          <span className="sr-only">审批</span>
          <select
            data-testid="agent-chat-seat-permission"
            disabled={busy}
            title={busy ? "会话未就绪或保存中" : "本会话审批模式（完全访问需确认）"}
            value={permissionMode}
            onChange={(e) => {
              const next = e.target.value.trim() as PermissionMode;
              if (!next || next === permissionMode) return;
              if (next === "full") {
                const ok = window.confirm(FULL_ACCESS_CONFIRM_MESSAGE);
                if (!ok) return;
              }
              patchMut.mutate({ permissionMode: next });
            }}
          >
            {PERMISSION_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </label>
        <label className="agent-chat-seat agent-chat-seat-plan">
          <span className="sr-only">Plan</span>
          {session?.planId ? (
            <input
              data-testid="agent-chat-seat-plan"
              disabled={busy}
              title={busy ? "会话未就绪或保存中" : "绑定 / 修改 planId"}
              value={planDraft}
              placeholder="planId"
              onChange={(e) => setPlanDraft(e.target.value)}
              onBlur={() => {
                const next = planDraft.trim();
                if (next !== (session.planId || "")) {
                  patchMut.mutate({ planId: next });
                }
              }}
            />
          ) : (
            <span className="muted-line" data-testid="agent-chat-seat-plan-empty">
              <input
                data-testid="agent-chat-seat-plan"
                disabled={busy}
                title={busy ? "会话未就绪或保存中" : "可选：绑定 planId（失焦保存）"}
                value={planDraft}
                placeholder="planId"
                onChange={(e) => setPlanDraft(e.target.value)}
                onBlur={() => {
                  const next = planDraft.trim();
                  if (next) patchMut.mutate({ planId: next });
                }}
              />
            </span>
          )}
        </label>
      </div>
      <div className="agent-chat-seats-model">
        <button
          type="button"
          className="agent-chat-seat-model-trigger"
          data-testid="agent-chat-seat-model-trigger"
          disabled={busy}
          aria-haspopup="menu"
          aria-expanded={modelMenu !== "closed"}
          title={
            busy
              ? "会话未就绪或保存中"
              : `选择本会话模型 / 推理等级 · ${providersQuery.isLoading ? "正在探测模型路由" : providerHealth}`
          }
          onClick={() => setModelMenu((cur) => (cur === "closed" ? "root" : "closed"))}
        >
          <span data-testid="agent-chat-seat-model-name">{modelLabel}</span>
          <span className="agent-chat-seat-effort-caption">{reasoningEffortLabel(effort)}</span>
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
          <div className="agent-chat-seat-model-menu" data-testid="agent-chat-seat-model-menu" role="menu">
            {modelMenu === "root" ? (
              <>
                <button
                  type="button"
                  className="agent-chat-seat-model-row"
                  data-testid="agent-chat-seat-model-open"
                  role="menuitem"
                  onClick={() => setModelMenu("model")}
                >
                  <span>模型</span>
                  <span>{modelLabel}</span>
                </button>
                <button
                  type="button"
                  className="agent-chat-seat-model-row"
                  data-testid="agent-chat-seat-effort-open"
                  role="menuitem"
                  onClick={() => setModelMenu("effort")}
                >
                  <span>推理等级</span>
                  <span>{reasoningEffortLabel(effort)}</span>
                </button>
              </>
            ) : null}
            {modelMenu === "model"
              ? modelItems.map((m) => (
                  <button
                    key={m.id}
                    type="button"
                    className="agent-chat-seat-model-row"
                    data-testid={`agent-chat-seat-model-pick-${m.id}`}
                    role="menuitem"
                    disabled={busy}
                    onClick={() => {
                      if (m.id !== providerKind) patchMut.mutate({ providerKind: m.id });
                      setModelMenu("closed");
                    }}
                  >
                    {m.label || m.id}
                  </button>
                ))
              : null}
            {modelMenu === "effort"
              ? REASONING_EFFORTS.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    className="agent-chat-seat-model-row"
                    data-testid={`agent-chat-seat-effort-${item.id}`}
                    role="menuitem"
                    onClick={() => {
                      if (session?.id) writeReasoningEffort(session.id, item.id);
                      setEffort(item.id);
                      setModelMenu("closed");
                      if (session?.id) patchMut.mutate({ reasoningEffort: item.id });
                    }}
                  >
                    {item.label}
                  </button>
                ))
              : null}
          </div>
        ) : null}
      </div>
    </div>
  );
}
