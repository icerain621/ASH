import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import {
  postRagLspDefinition,
  postRagLspHover,
  postRagLspReferences,
} from "@/modules/observability/api/observability.api";
import { getCurrentSpaceId } from "@/services/http/client";

type Props = {
  defaultRepoRoot?: string;
  testIdPrefix?: string;
};

export function RagLspProbePanel({ defaultRepoRoot = ".", testIdPrefix = "rag-lsp" }: Props) {
  const spaceId = getCurrentSpaceId();
  const [repoRoot, setRepoRoot] = useState(defaultRepoRoot);
  const [path, setPath] = useState("main.go");
  const [line, setLine] = useState(1);
  const [character, setCharacter] = useState(0);
  const [result, setResult] = useState<string>("");

  const body = () => ({
    repoRoot: repoRoot.trim() || ".",
    path: path.trim(),
    line: Number(line) || 1,
    character: Number(character) || 0,
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
            data-testid={`${testIdPrefix}-path`}
          />
        </label>
        <label className="scenario-picker">
          line
          <input
            type="number"
            min={1}
            value={line}
            onChange={(e) => setLine(Number(e.target.value))}
            data-testid={`${testIdPrefix}-line`}
          />
        </label>
        <label className="scenario-picker">
          char
          <input
            type="number"
            min={0}
            value={character}
            onChange={(e) => setCharacter(Number(e.target.value))}
            data-testid={`${testIdPrefix}-char`}
          />
        </label>
      </div>
      <div className="toolbar metrics-toolbar" style={{ gap: "0.5rem", marginTop: "0.5rem" }}>
        <button
          type="button"
          className="btn"
          disabled={busy}
          data-testid={`${testIdPrefix}-hover`}
          onClick={() => hoverMut.mutate()}
        >
          Hover
        </button>
        <button
          type="button"
          className="btn"
          disabled={busy}
          data-testid={`${testIdPrefix}-definition`}
          onClick={() => defMut.mutate()}
        >
          Definition
        </button>
        <button
          type="button"
          className="btn"
          disabled={busy}
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
