import { useMutation } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import {
  postRagLspDefinition,
  postRagLspHover,
  postRagLspReferences,
} from "@/modules/observability/api/observability.api";
import { getCurrentSpaceId } from "@/services/http/client";

type Props = {
  defaultRepoRoot?: string;
  /** Sample file for probes; default matches ASH Go entrypoint (not bare main.go). */
  defaultPath?: string;
  testIdPrefix?: string;
};

export function RagLspProbePanel({
  defaultRepoRoot = ".",
  defaultPath = "cmd/worker/main.go",
  testIdPrefix = "rag-lsp",
}: Props) {
  const spaceId = getCurrentSpaceId();
  const [repoRoot, setRepoRoot] = useState(defaultRepoRoot);
  const [path, setPath] = useState(defaultPath);
  const [line, setLine] = useState(1);
  const [character, setCharacter] = useState(0);
  const [result, setResult] = useState<string>("");

  useEffect(() => {
    setRepoRoot(defaultRepoRoot);
  }, [defaultRepoRoot]);

  const pathReady = Boolean(path.trim());
  const repoReady = Boolean(repoRoot.trim());
  const lineIssue =
    !Number.isFinite(line) || !Number.isInteger(line) || line < 1
      ? "line 须为 ≥1 的整数"
      : "";
  const charIssue =
    !Number.isFinite(character) || !Number.isInteger(character) || character < 0
      ? "char 须为 ≥0 的整数"
      : "";

  const body = () => ({
    repoRoot: repoRoot.trim(),
    path: path.trim(),
    line,
    character,
    spaceId,
  });

  const hoverMut = useMutation({
    mutationFn: () => postRagLspHover(body()),
    onSuccess: (data) => {
      setResult(`hover (${data.server ?? "?"}): ${data.contents || "(empty)"}`);
    },
    onError: (err: Error) => setResult(`hover error: ${err.message}`),
  });
  const defMut = useMutation({
    mutationFn: () => postRagLspDefinition(body()),
    onSuccess: (data) => {
      const locs = (data.locations ?? [])
        .map((l) => `${l.path}:${l.line}`)
        .join(", ");
      setResult(`definition (${data.server ?? "?"}): ${locs || "(none)"}`);
    },
    onError: (err: Error) => setResult(`definition error: ${err.message}`),
  });
  const refsMut = useMutation({
    mutationFn: () => postRagLspReferences({ ...body(), limit: 20 }),
    onSuccess: (data) => {
      const locs = (data.locations ?? [])
        .map((l) => `${l.path}:${l.line}`)
        .join(", ");
      setResult(
        `references [${data.source}${data.truncated ? ", truncated" : ""}]: ${locs || "(none)"}`,
      );
    },
    onError: (err: Error) => setResult(`references error: ${err.message}`),
  });

  const busy = hoverMut.isPending || defMut.isPending || refsMut.isPending;
  const actionsDisabled = busy || !pathReady || !repoReady || Boolean(lineIssue) || Boolean(charIssue);
  const actionTitle = (idle: string) =>
    !repoReady
      ? "需要填写 repoRoot"
      : !pathReady
        ? "需要填写 path"
        : lineIssue
          ? lineIssue
          : charIssue
            ? charIssue
            : busy
              ? "查询中…"
              : idle;

  return (
    <div className="card-like" data-testid={`${testIdPrefix}-probe`}>
      <p className="muted">样本查询（RAG 内部 LSP：hover / definition / references）</p>
      <div className="toolbar metrics-toolbar" style={{ flexWrap: "wrap", gap: "0.5rem" }}>
        <label className="scenario-picker">
          repoRoot
          <input
            value={repoRoot}
            onChange={(e) => setRepoRoot(e.target.value)}
            data-testid={`${testIdPrefix}-repo-root`}
          />
        </label>
        <label className="scenario-picker">
          path
          <input
            value={path}
            onChange={(e) => setPath(e.target.value)}
            placeholder="cmd/worker/main.go"
            data-testid={`${testIdPrefix}-path`}
          />
        </label>
        <label className="scenario-picker">
          line
          <input
            type="number"
            min={1}
            value={Number.isFinite(line) ? line : ""}
            onChange={(e) => {
              const raw = e.target.value;
              setLine(raw === "" ? Number.NaN : Number(raw));
            }}
            data-testid={`${testIdPrefix}-line`}
          />
        </label>
        <label className="scenario-picker">
          char
          <input
            type="number"
            min={0}
            value={Number.isFinite(character) ? character : ""}
            onChange={(e) => {
              const raw = e.target.value;
              setCharacter(raw === "" ? Number.NaN : Number(raw));
            }}
            data-testid={`${testIdPrefix}-char`}
          />
        </label>
      </div>
      <div className="toolbar metrics-toolbar" style={{ gap: "0.5rem", marginTop: "0.5rem" }}>
        <button
          type="button"
          className="btn"
          disabled={actionsDisabled}
          title={actionTitle("LSP Hover（当前 path/line/char）")}
          data-testid={`${testIdPrefix}-hover`}
          onClick={() => hoverMut.mutate()}
        >
          Hover
        </button>
        <button
          type="button"
          className="btn"
          disabled={actionsDisabled}
          title={actionTitle("LSP Definition（跳转定义）")}
          data-testid={`${testIdPrefix}-definition`}
          onClick={() => defMut.mutate()}
        >
          Definition
        </button>
        <button
          type="button"
          className="btn"
          disabled={actionsDisabled}
          title={actionTitle("LSP References（查找引用）")}
          data-testid={`${testIdPrefix}-references`}
          onClick={() => refsMut.mutate()}
        >
          References
        </button>
      </div>
      {result ? (
        <pre
          className="muted"
          style={{ whiteSpace: "pre-wrap", marginTop: "0.75rem" }}
          data-testid={`${testIdPrefix}-result`}
        >
          {result}
        </pre>
      ) : null}
    </div>
  );
}
