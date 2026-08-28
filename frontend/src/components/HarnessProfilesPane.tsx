import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Shield, Upload } from "lucide-react";
import { useState } from "react";
import {
  createHarnessProfile,
  listHarnessProfiles,
  loadActiveHarnessProfile,
  promoteHarnessProfile,
  rollbackHarnessProfile,
  submitHarnessReview,
} from "@/modules/reviews/api/reviews.api";
import { shortId } from "@/shared/utils/format";

const defaultSpec = {
  provider: { kind: "static" },
  sandbox: { defaultMode: "workspace-write", network: "deny" },
  tools: { allowlist: [] as string[] },
  policyProfile: "default",
};

export function HarnessProfilesPane() {
  const qc = useQueryClient();
  const [name, setName] = useState("default");
  const [sandboxMode, setSandboxMode] = useState("workspace-write");
  const [message, setMessage] = useState("");

  const activeQuery = useQuery({
    queryKey: ["harness", "active", name],
    queryFn: () => loadActiveHarnessProfile(name),
  });
  const draftsQuery = useQuery({
    queryKey: ["harness", "drafts", name],
    queryFn: () => listHarnessProfiles("draft", name),
  });
  const reviewQuery = useQuery({
    queryKey: ["harness", "in_review", name],
    queryFn: () => listHarnessProfiles("in_review", name),
  });

  const refresh = () => {
    qc.invalidateQueries({ queryKey: ["harness"] });
    qc.invalidateQueries({ queryKey: ["reviews-queue"] });
  };

  const createMut = useMutation({
    mutationFn: () =>
      createHarnessProfile({
        name: name.trim() || "default",
        spec: {
          ...defaultSpec,
          sandbox: { defaultMode: sandboxMode, network: "deny" },
        },
      }),
    onSuccess: (p) => {
      setMessage(`草稿 ${shortId(p.id)} v${p.version}`);
      refresh();
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const submitMut = useMutation({
    mutationFn: submitHarnessReview,
    onSuccess: () => {
      setMessage("已提交编排评审");
      refresh();
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const promoteMut = useMutation({
    mutationFn: promoteHarnessProfile,
    onSuccess: () => {
      setMessage("已升格为 active（须先评审通过）");
      refresh();
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const rollbackMut = useMutation({
    mutationFn: rollbackHarnessProfile,
    onSuccess: (p) => {
      setMessage(`已回滚到 v${p.version}`);
      refresh();
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const active = activeQuery.data?.profile;
  const drafts = draftsQuery.data?.items ?? [];
  const inReview = reviewQuery.data?.items ?? [];

  return (
    <div className="pane" data-testid="harness-profiles-pane">
      <div className="pane-title">
        <h2>
          <Shield size={15} strokeWidth={1.8} />
          Harness Profile
        </h2>
        <span>{active ? `active v${active.version}` : "no active"}</span>
      </div>
      <p className="muted-line">draft → submit-review → Reviews 批准 → promote。draft 不可直通升格。</p>
      <div className="secret-form">
        <label>
          Name
          <input value={name} onChange={(e) => setName(e.target.value)} data-testid="harness-name" />
        </label>
        <label>
          Sandbox mode
          <select value={sandboxMode} onChange={(e) => setSandboxMode(e.target.value)} data-testid="harness-sandbox-mode">
            <option value="off">off</option>
            <option value="workspace-write">workspace-write</option>
            <option value="isolated">isolated</option>
          </select>
        </label>
        <button className="btn mini" type="button" disabled={createMut.isPending} onClick={() => createMut.mutate()} data-testid="harness-create-draft">
          <Upload size={13} strokeWidth={1.8} />
          新建草稿
        </button>
        <span>{message || "—"}</span>
      </div>
      {active ? (
        <div className="abi-strip">
          <span>
            {active.name}@v{active.version}
          </span>
          <span>{active.spec?.sandbox?.defaultMode ?? "-"}</span>
          <span>{active.spec?.provider?.kind ?? "-"}</span>
          <button
            className="btn mini"
            type="button"
            disabled={rollbackMut.isPending}
            onClick={() => rollbackMut.mutate(active.id)}
            data-testid="harness-rollback"
          >
            Rollback
          </button>
        </div>
      ) : null}
      <table className="table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Ver</th>
            <th>Status</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {[...drafts, ...inReview].map((p) => (
            <tr key={p.id}>
              <td title={p.id}>{shortId(p.id)}</td>
              <td>v{p.version}</td>
              <td>{p.status}</td>
              <td>
                <div className="row-actions">
                  {p.status === "draft" ? (
                    <button className="btn mini" type="button" disabled={submitMut.isPending} onClick={() => submitMut.mutate(p.id)}>
                      提交评审
                    </button>
                  ) : null}
                  {p.status === "in_review" ? (
                    <button className="btn mini ok" type="button" disabled={promoteMut.isPending} onClick={() => promoteMut.mutate(p.id)} title="须先在 Reviews 批准">
                      Promote
                    </button>
                  ) : null}
                </div>
              </td>
            </tr>
          ))}
          {!drafts.length && !inReview.length && (
            <tr className="empty-row">
              <td colSpan={4}>暂无 draft / in_review Profile。</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
