import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createAgentAsset,
  createMemoryAsset,
  getSpacePolicy,
  listAgentAssets,
  listMemoryAssets,
  patchAgentAssetStatus,
  patchMemoryAssetStatus,
  putSpacePolicy,
  type RegistryAsset,
} from "@/modules/registry/api/registry.api";

export function RegistryAssetsPanel({ spaceId }: { spaceId: string }) {
  const qc = useQueryClient();
  const agentsQuery = useQuery({
    queryKey: ["agent-assets", spaceId],
    queryFn: () => listAgentAssets("", 40),
  });
  const memoryQuery = useQuery({
    queryKey: ["memory-assets", spaceId],
    queryFn: () => listMemoryAssets("", 40),
  });
  const policyQuery = useQuery({
    queryKey: ["space-policy", spaceId],
    queryFn: () => getSpacePolicy(spaceId),
  });

  const patchAgent = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) => patchAgentAssetStatus(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["agent-assets", spaceId] }),
  });
  const patchMemory = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) => patchMemoryAssetStatus(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["memory-assets", spaceId] }),
  });
  const createAgent = useMutation({
    mutationFn: () => createAgentAsset({ kind: "harness_profile", name: `agent-${Date.now()}` }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["agent-assets", spaceId] }),
  });
  const createMemory = useMutation({
    mutationFn: () => createMemoryAsset({ kind: "record", name: `memory-${Date.now()}` }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["memory-assets", spaceId] }),
  });
  const putPolicy = useMutation({
    mutationFn: (body: { citationMode?: string; multiSign?: boolean; reviewSlaHours?: number }) =>
      putSpacePolicy(spaceId, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["space-policy", spaceId] }),
  });

  const eff = policyQuery.data?.effective;

  return (
    <div className="pane" data-testid="registry-assets-panel">
      <div className="pane-title">
        <h2>管控登记 / 策略</h2>
        <span>{spaceId}</span>
      </div>

      {eff ? (
        <div className="muted-line" data-testid="effective-policy-summary">
          Effective：citation={eff.citationMode} · multiSign={String(eff.multiSign)} · SLA={eff.reviewSlaHours}h
          <div>sources: {(eff.sources ?? []).join(" → ")}</div>
        </div>
      ) : null}

      <div className="row-actions" style={{ marginBottom: 8 }}>
        <button type="button" className="btn mini" onClick={() => createAgent.mutate()} data-testid="registry-create-agent">
          + Agent 资产
        </button>
        <button type="button" className="btn mini" onClick={() => createMemory.mutate()} data-testid="registry-create-memory">
          + Memory 资产
        </button>
        <button
          type="button"
          className="btn mini"
          onClick={() => putPolicy.mutate({ citationMode: "required", multiSign: true, reviewSlaHours: 48 })}
          data-testid="registry-put-policy"
        >
          写入策略包
        </button>
      </div>

      <AssetTable
        title="Agent assets"
        items={agentsQuery.data?.items ?? []}
        onToggle={(id, status) => patchAgent.mutate({ id, status: status === "active" ? "disabled" : "active" })}
        testId="agent-assets-list"
      />
      <AssetTable
        title="Memory assets"
        items={memoryQuery.data?.items ?? []}
        onToggle={(id, status) => patchMemory.mutate({ id, status: status === "active" ? "disabled" : "active" })}
        testId="memory-assets-list"
      />
    </div>
  );
}

function AssetTable({
  title,
  items,
  onToggle,
  testId,
}: {
  title: string;
  items: RegistryAsset[];
  onToggle: (id: string, status: string) => void;
  testId: string;
}) {
  return (
    <div>
      <h3>{title}</h3>
      <table className="table" data-testid={testId}>
        <thead>
          <tr>
            <th>Name</th>
            <th>Kind</th>
            <th>Status</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {items.map((it) => (
            <tr key={it.id}>
              <td>{it.name}</td>
              <td>{it.kind}</td>
              <td>{it.status}</td>
              <td>
                <button type="button" className="btn mini" onClick={() => onToggle(it.id, it.status)} data-testid={`toggle-${it.id}`}>
                  {it.status === "active" ? "停用" : "启用"}
                </button>
              </td>
            </tr>
          ))}
          {items.length === 0 ? (
            <tr className="empty-row">
              <td colSpan={4}>暂无登记</td>
            </tr>
          ) : null}
        </tbody>
      </table>
    </div>
  );
}
