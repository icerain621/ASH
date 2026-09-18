import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
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

type Props = {
  session: AgentSessionView | null;
  disabled?: boolean;
};

/** Model / permission / plan seats above the composer. */
export function ChatSeats({ session, disabled = false }: Props) {
  const qc = useQueryClient();
  const [planDraft, setPlanDraft] = useState(session?.planId ?? "");

  useEffect(() => {
    setPlanDraft(session?.planId ?? "");
  }, [session?.id, session?.planId]);

  const modelsQuery = useQuery({
    queryKey: ["agent-models"],
    queryFn: () => listAgentModels(),
    staleTime: 60_000,
  });

  const patchMut = useMutation({
    mutationFn: (body: {
      providerKind?: string;
      permissionMode?: PermissionMode;
      planId?: string;
    }) => {
      if (!session?.id) throw new Error("session not ready");
      return updateSession(session.id, body);
    },
    onSuccess: (updated) => {
      void qc.invalidateQueries({ queryKey: ["agent-session", updated.id] });
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
    },
  });

  const busy = disabled || !session?.id || patchMut.isPending;
  const providerKind = session?.providerKind || modelsQuery.data?.items?.[0]?.id || "static";
  const permissionMode = resolvePermissionMode(session?.permissionMode);

  return (
    <div className="agent-chat-seats" data-testid="agent-chat-seats">
      <label className="agent-chat-seat">
        Model
        <select
          data-testid="agent-chat-seat-model"
          disabled={busy}
          value={providerKind}
          onChange={(e) => {
            const next = e.target.value.trim();
            if (!next || next === providerKind) return;
            patchMut.mutate({ providerKind: next });
          }}
        >
          {(modelsQuery.data?.items ?? [{ id: providerKind, label: providerKind, providerKind }]).map(
            (m) => (
              <option key={m.id} value={m.id}>
                {m.label || m.id}
              </option>
            ),
          )}
        </select>
      </label>
      <label className="agent-chat-seat">
        审批
        <select
          data-testid="agent-chat-seat-permission"
          disabled={busy}
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
        Plan
        {session?.planId ? (
          <input
            data-testid="agent-chat-seat-plan"
            disabled={busy}
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
              value={planDraft}
              placeholder="可选 planId"
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
  );
}
