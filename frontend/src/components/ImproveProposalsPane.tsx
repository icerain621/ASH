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
  type ImproveProposal,
} from "@/modules/improve/api/improve.api";
import { shortId } from "@/shared/utils/format";

/** True when proposal was auto-drafted from a low review score. */
export function isLowScoreTriggered(item: Pick<ImproveProposal, "source" | "scoreEventId" | "changeSummary">): boolean {
  if (item.source === "low_score") return true;
  if (item.scoreEventId) return true;
  if (item.changeSummary?.includes("low_score")) return true;
  return false;
}

export function lowScoreSourceLabel(item: Pick<ImproveProposal, "scoreEventId">): string {
  const id = item.scoreEventId?.trim();
  if (id) return `由低分触发 · scoreEvent ${shortId(id)}`;
  return "由低分触发";
}

/** Parse canary percent; returns issue message or a valid integer in 1..100. */
export function parseCanaryPercent(raw: string): { percent: number; issue: string } {
  const trimmed = raw.trim();
  if (!trimmed) return { percent: 0, issue: "需要填写灰度 %（1–100）" };
  const n = Number(trimmed);
  if (!Number.isInteger(n) || n < 1 || n > 100) {
    return { percent: 0, issue: "灰度 % 须为 1–100 的整数" };
  }
  return { percent: n, issue: "" };
}

export function ImproveProposalsPane() {
  const qc = useQueryClient();
  const [title, setTitle] = useState("M1 自我迭代提案");
  const [baselineRunId, setBaselineRunId] = useState("");
  const [canaryPercent, setCanaryPercent] = useState("10");
  const [message, setMessage] = useState("");
  const canaryParsed = parseCanaryPercent(canaryPercent);
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
    <div className="pane" data-testid="improve-proposals-pane">
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
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            data-testid="improve-title"
          />
        </label>
        <label>
          基线 Run ID
          <input
            placeholder="run_..."
            value={baselineRunId}
            onChange={(e) => setBaselineRunId(e.target.value)}
            data-testid="improve-baseline"
          />
        </label>
        <label>
          灰度 %
          <input
            type="number"
            min={1}
            max={100}
            value={canaryPercent}
            onChange={(e) => setCanaryPercent(e.target.value)}
            data-testid="improve-canary-percent"
          />
        </label>
        <button
          className="btn mini icon-only"
          type="button"
          disabled={createMut.isPending || !title.trim() || !baselineRunId.trim()}
          title={
            !title.trim()
              ? "需要填写标题"
              : !baselineRunId.trim()
                ? "需要填写基线 Run ID"
                : createMut.isPending
                  ? "创建中…"
                  : "创建提案（需确认）"
          }
          aria-label={
            !title.trim()
              ? "需要填写标题"
              : !baselineRunId.trim()
                ? "需要填写基线 Run ID"
                : "创建提案"
          }
          data-testid="improve-create"
          onClick={() => {
            const ok = window.confirm(
              `确认创建自我迭代提案「${title.trim()}」？基线 Run=${baselineRunId.trim()}`,
            );
            if (!ok) return;
            createMut.mutate();
          }}
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
            <tr key={item.id} data-testid={`improve-proposal-${item.id}`}>
              <td title={item.id}>
                {item.title}
                {isLowScoreTriggered(item) ? (
                  <div className="muted-line" data-testid="improve-low-score-source">
                    {lowScoreSourceLabel(item)}
                  </div>
                ) : null}
              </td>
              <td>{item.status}</td>
              <td title={item.baselineRunId || undefined} data-testid={`improve-baseline-${item.id}`}>
                {item.baselineRunId?.trim() ? shortId(item.baselineRunId) : "—"}
              </td>
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
                    data-testid={`improve-experiment-${item.id}`}
                    disabled={experimentMut.isPending || !item.baselineRunId?.trim()}
                    title={
                      !item.baselineRunId?.trim()
                        ? "需要基线 Run"
                        : "启动实验（需确认）"
                    }
                    onClick={() => {
                      const ok = window.confirm(`确认对提案「${item.title || item.id}」启动实验？`);
                      if (!ok) return;
                      experimentMut.mutate(item.id);
                    }}
                  >
                    实验
                  </button>
                  <button
                    className="btn mini"
                    type="button"
                    data-testid={`improve-canary-${item.id}`}
                    disabled={
                      canaryMut.isPending || !item.baselineRunId?.trim() || Boolean(canaryParsed.issue)
                    }
                    title={
                      !item.baselineRunId?.trim()
                        ? "需要基线 Run"
                        : canaryParsed.issue
                          ? canaryParsed.issue
                          : "启动灰度（需确认）"
                    }
                    onClick={() => {
                      if (!item.baselineRunId?.trim() || canaryParsed.issue) return;
                      const ok = window.confirm(
                        `确认对提案「${item.title || item.id}」启动灰度（${canaryParsed.percent}%）？`,
                      );
                      if (!ok) return;
                      canaryMut.mutate({ id: item.id, percent: canaryParsed.percent });
                    }}
                  >
                    灰度
                  </button>
                  <button
                    className="btn mini ok icon-only"
                    type="button"
                    data-testid={`improve-promote-${item.id}`}
                    disabled={
                      promoteMut.isPending ||
                      (item.status !== "canary" && item.status !== "experimenting")
                    }
                    title={
                      item.status !== "canary" && item.status !== "experimenting"
                        ? "需先实验或灰度后再晋升"
                        : "晋升（需确认）"
                    }
                    aria-label={
                      item.status !== "canary" && item.status !== "experimenting"
                        ? "需先实验或灰度后再晋升"
                        : "晋升"
                    }
                    onClick={() => {
                      const ok = window.confirm(`确认晋升提案「${item.title || item.id}」？`);
                      if (!ok) return;
                      promoteMut.mutate(item.id);
                    }}
                  >
                    <Upload size={13} strokeWidth={1.8} />
                  </button>
                  <button
                    className="btn mini err icon-only"
                    type="button"
                    data-testid={`improve-rollback-${item.id}`}
                    disabled={
                      rollbackMut.isPending ||
                      item.status === "rolled_back" ||
                      item.status === "promoted"
                    }
                    title={
                      item.status === "rolled_back" || item.status === "promoted"
                        ? "已终态，不可再回滚"
                        : "回滚（需确认）"
                    }
                    aria-label={
                      item.status === "rolled_back" || item.status === "promoted"
                        ? "已终态，不可再回滚"
                        : "回滚"
                    }
                    onClick={() => {
                      const ok = window.confirm(`确认回滚提案「${item.title || item.id}」？`);
                      if (!ok) return;
                      rollbackMut.mutate(item.id);
                    }}
                  >
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
