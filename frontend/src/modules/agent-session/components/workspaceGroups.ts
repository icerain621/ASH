import type { AgentSessionView } from "../api/session.api";
import type { AgentWorkspaceView } from "../api/workspace.api";

export type WorkspaceGroup = {
  workspace: AgentWorkspaceView | null;
  sessions: AgentSessionView[];
};

/** Group sessions under workspaces; unassigned bucket last. */
export function groupSessionsByWorkspace(
  sessions: AgentSessionView[],
  workspaces: AgentWorkspaceView[],
): WorkspaceGroup[] {
  const byId = new Map(sessions.map((s) => [s.id, s]));
  const claimed = new Set<string>();
  const groups: WorkspaceGroup[] = [];

  for (const ws of workspaces) {
    const ordered: AgentSessionView[] = [];
    for (const sid of ws.sessionIds || []) {
      const sess = byId.get(sid);
      if (sess) {
        ordered.push(sess);
        claimed.add(sid);
      }
    }
    // Also pick up sessions that declare workspaceId but are missing from sessionIds.
    for (const sess of sessions) {
      if (sess.workspaceId === ws.id && !claimed.has(sess.id)) {
        ordered.push(sess);
        claimed.add(sess.id);
      }
    }
    groups.push({ workspace: ws, sessions: ordered });
  }

  const unassigned = sessions.filter((s) => !claimed.has(s.id));
  if (unassigned.length > 0 || workspaces.length === 0) {
    groups.push({ workspace: null, sessions: unassigned });
  }
  return groups;
}
