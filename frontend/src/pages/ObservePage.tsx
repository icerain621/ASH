import { Link, useSearch } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { LensPanel } from "@/modules/observability/components/LensPanel";
import type { ObserveLens } from "@/modules/observability/api/lenses.api";

const LENSES: { id: ObserveLens; label: string }[] = [
  { id: "global", label: "全局" },
  { id: "agent", label: "Agent" },
  { id: "memory", label: "记忆" },
];

function parseLens(raw: unknown): ObserveLens {
  if (raw === "agent" || raw === "memory" || raw === "global") return raw;
  return "global";
}

export function ObservePage() {
  const search = useSearch({ strict: false }) as { lens?: string; run?: string; id?: string };
  const lens = parseLens(search.lens);
  const [runDraft, setRunDraft] = useState(search.run ?? "");
  const [memDraft, setMemDraft] = useState(search.id ?? "");
  const runId = useMemo(() => (search.run || "").trim(), [search.run]);
  const memoryId = useMemo(() => (search.id || "").trim(), [search.id]);

  return (
    <div className="page" data-testid="observe-page">
      <div className="pane-title">
        <h1>观测</h1>
        <p className="muted-line">三镜头共用事件脊柱 · `/ui/observe?lens=…`</p>
      </div>
      <div className="toolbar" role="tablist" aria-label="观测镜头" data-testid="observe-lens-tabs">
        {LENSES.map((item) => (
          <Link
            key={item.id}
            to="/observe"
            search={{
              lens: item.id,
              ...(item.id === "agent" && runId ? { run: runId } : {}),
              ...(item.id === "memory" && memoryId ? { id: memoryId } : {}),
            }}
            role="tab"
            aria-selected={lens === item.id}
            className={lens === item.id ? "btn mini primary" : "btn mini"}
            data-testid={`observe-tab-${item.id}`}
          >
            {item.label}
          </Link>
        ))}
        <Link to="/observability" className="btn mini" data-testid="observe-ops-link">
          运维面板
        </Link>
      </div>

      {lens === "agent" ? (
        <form
          className="toolbar"
          onSubmit={(e) => {
            e.preventDefault();
            window.location.assign(`/ui/observe?lens=agent&run=${encodeURIComponent(runDraft.trim())}`);
          }}
        >
          <input
            value={runDraft}
            onChange={(e) => setRunDraft(e.target.value)}
            placeholder="run id"
            data-testid="observe-run-input"
          />
          <button type="submit" className="btn mini" data-testid="observe-run-go">
            打开
          </button>
        </form>
      ) : null}

      {lens === "memory" ? (
        <form
          className="toolbar"
          onSubmit={(e) => {
            e.preventDefault();
            window.location.assign(`/ui/observe?lens=memory&id=${encodeURIComponent(memDraft.trim())}`);
          }}
        >
          <input
            value={memDraft}
            onChange={(e) => setMemDraft(e.target.value)}
            placeholder="memory id"
            data-testid="observe-memory-input"
          />
          <button type="submit" className="btn mini" data-testid="observe-memory-go">
            打开
          </button>
        </form>
      ) : null}

      <LensPanel lens={lens} runId={runId || undefined} memoryId={memoryId || undefined} />
    </div>
  );
}
