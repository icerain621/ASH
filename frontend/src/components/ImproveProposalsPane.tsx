import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FlaskConical, Play, RotateCcw, Upload } from "lucide-react";
import { useState } from "react";
import {
  createImproveProposal,
  listImproveProposals,
  promoteImproveProposal,
  rollbackImproveProposal,
  startImproveCanary,
  startImproveExperiment,
} from "@/modules/improve/api/improve.api";
import { shortId } from "@/shared/utils/format";

export function ImproveProposalsPane() {
  const qc = useQueryClient();
  const [title, setTitle] = useState("M1 自我迭代提案");
  const [baselineRunId, setBaselineRunId] = useState("");
  const [canaryPercent, setCanaryPercent] = useState("10");
  const [message, setMessage] = useState("");

  const proposalsQuery = useQuery({
    queryKey: ["improve", "proposals"],
    queryFn: () => listImproveProposals(20),
  });

  const refresh = () => qc.invalidateQueries({ queryKey: ["improve", "proposals"] });

  const createMut = useMutation({
    mutationFn: () =>
      createImproveProposal({
        title,
        baselineRunId: baselineRunId.trim(),
        changeSummary: "M1 replay compare",
      }),
    onSuccess: (res) => {
      setMessage(`已创建提案 ${shortId(res.id)}`);
      refresh();
    },
    onError: (err) => setMessage((err as Error).message),
  });

  const experimentMut = useMutation({
    mutationFn: (id: string) => startImproveExperiment(id),
    onSuccess: (res) => {
      setMessage(`实验运行 ${shortId(res.experimentRunId)}，匹配 ${res.compare?.matched ?? 0}`);
      refresh();
    },
    onError: (err) => setMessage((err as Error).message),
  });

  const canaryMut = useMutation({
    mutationFn: ({ id, percent }: { id: string; percent: number }) => startImproveCanary(id, percent),
    onSuccess: () => {
      setMessage("灰度已启动");
      refresh();
    },
    onError: (err) => setMessage((err as Error).message),
  });

  const promoteMut = useMutation({
    mutationFn: promoteImproveProposal,
    onSuccess: () => {
      setMessage("已晋升");
      refresh();
    },
    onError: (err) => setMessage((err as Error).message),
  });

  const rollbackMut = useMutation({
    mutationFn: rollbackImproveProposal,
    onSuccess: () => {
      setMessage("已回滚");
      refresh();
    },
    onError: (err) => setMessage((err as Error).message),
  });

  const items = proposalsQuery.data?.items ?? [];

  return (
    <div className="pane">
      <div className="pane-title">
        <h2>
          <FlaskConical size={15} strokeWidth={1.8} />
          自我迭代 (M1)
        </h2>
        <span>{message || `${items.length} 个提案`}</span>
      </div>
      <div className="secret-form">
        <label>
          标题
          <input value={title} onChange={(e) => setTitle(e.target.value)} />
        </label>
        <label>
          基线 Run ID
          <input placeholder="run_..." value={baselineRunId} onChange={(e) => setBaselineRunId(e.target.value)} />
        </label>
        <label>
          灰度 %
          <input type="number" min={1} max={100} value={canaryPercent} onChange={(e) => setCanaryPercent(e.target.value)} />
        </label>
        <button
          className="btn mini icon-only"
          type="button"
          disabled={createMut.isPending || !baselineRunId.trim()}
          title="创建提案"
          onClick={() => createMut.mutate()}
        >
          <Play size={13} strokeWidth={1.8} />
        </button>
      </div>
      <table className="table">
        <thead>
          <tr>
            <th>提案</th>
            <th>状态</th>
            <th>基线</th>
            <th>实验</th>
            <th>对照</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id}>
              <td title={item.id}>{item.title}</td>
              <td>{item.status}</td>
              <td title={item.baselineRunId}>{shortId(item.baselineRunId)}</td>
              <td title={item.experimentRunId}>{item.experimentRunId ? shortId(item.experimentRunId) : "-"}</td>
              <td>
                {item.compare
                  ? `M${item.compare.matched}/C${item.compare.changed}`
                  : "-"}
              </td>
              <td>
                <div className="row-actions">
                  <button
                    className="btn mini"
                    type="button"
                    disabled={experimentMut.isPending}
                    onClick={() => experimentMut.mutate(item.id)}
                  >
                    实验
                  </button>
                  <button
                    className="btn mini"
                    type="button"
                    disabled={canaryMut.isPending}
                    onClick={() => canaryMut.mutate({ id: item.id, percent: Number(canaryPercent || 10) })}
                  >
                    灰度
                  </button>
                  <button className="btn mini ok icon-only" type="button" title="晋升" onClick={() => promoteMut.mutate(item.id)}>
                    <Upload size={13} strokeWidth={1.8} />
                  </button>
                  <button className="btn mini err icon-only" type="button" title="回滚" onClick={() => rollbackMut.mutate(item.id)}>
                    <RotateCcw size={13} strokeWidth={1.8} />
                  </button>
                </div>
              </td>
            </tr>
          ))}
          {!items.length && (
            <tr className="empty-row">
              <td colSpan={6}>暂无自我迭代提案。先在 Runs 完成一次运行，再填写基线 Run ID。</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
