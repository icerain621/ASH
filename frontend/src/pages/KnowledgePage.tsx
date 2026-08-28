import { useQuery } from "@tanstack/react-query";
import { BookOpen, RefreshCcw } from "lucide-react";
import { useState } from "react";
import { getRepoProfile, getWikiPage, listWikiPages } from "@/modules/knowledge/api/knowledge.api";

export function KnowledgePage() {
  const [repoRoot, setRepoRoot] = useState(".");
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const profileQuery = useQuery({
    queryKey: ["repo-profile", repoRoot],
    queryFn: () => getRepoProfile(repoRoot),
  });
  const wikiQuery = useQuery({
    queryKey: ["wiki-pages", repoRoot, query],
    queryFn: () => listWikiPages({ repoRoot, q: query || undefined, limit: 20 }),
  });
  const pageQuery = useQuery({
    queryKey: ["wiki-page", selectedId, repoRoot],
    queryFn: () => getWikiPage(selectedId!, repoRoot),
    enabled: Boolean(selectedId),
  });

  return (
    <section className="panel active">
      <div className="page-kicker">
        <BookOpen size={17} strokeWidth={1.8} />
        Knowledge
      </div>
      <div className="page-heading">
        <div>
          <h1>知识中心</h1>
          <p>即时 Repo Profile 与 Wiki 投影（不落库）；Run 准备阶段可注入 profile: / wiki: contextRefs。</p>
        </div>
        <div className="toolbar metrics-toolbar">
          <label className="scenario-picker">
            repoRoot
            <input
              value={repoRoot}
              onChange={(e) => setRepoRoot(e.target.value)}
              data-testid="knowledge-repo-root"
            />
          </label>
          <label className="scenario-picker">
            检索
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="architecture / testing…"
              data-testid="knowledge-query"
            />
          </label>
          <button
            className="btn icon-btn"
            onClick={() => {
              profileQuery.refetch();
              wikiQuery.refetch();
            }}
            disabled={profileQuery.isFetching || wikiQuery.isFetching}
          >
            <RefreshCcw size={16} />
            刷新
          </button>
        </div>
      </div>

      <div className="split-pane" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1rem" }}>
        <div>
          <h2>Repo Profile</h2>
          {profileQuery.isError && <p className="error">加载 Profile 失败</p>}
          {profileQuery.data && (
            <div className="card-like" data-testid="knowledge-profile">
              <p>
                <code>{profileQuery.data.contextRef}</code>
              </p>
              <p>{profileQuery.data.summary}</p>
              <p>
                语言：{(profileQuery.data.languages || []).join(", ") || "—"}
              </p>
              <p>
                测试：{(profileQuery.data.testCommands || []).join(" · ") || "—"}
              </p>
              <p>
                模块：{(profileQuery.data.modules || []).join(", ") || "—"}
              </p>
            </div>
          )}

          <h2 style={{ marginTop: "1.25rem" }}>Wiki 页</h2>
          {wikiQuery.isError && <p className="error">加载 Wiki 失败</p>}
          <ul className="list" data-testid="knowledge-wiki-list">
            {(wikiQuery.data?.items || []).map((item) => (
              <li key={item.id}>
                <button
                  type="button"
                  className="btn linkish"
                  onClick={() => setSelectedId(item.id)}
                  data-testid={`wiki-item-${item.id}`}
                >
                  {item.title}
                  <span className="muted"> · {item.source}</span>
                </button>
              </li>
            ))}
            {!wikiQuery.isLoading && (wikiQuery.data?.items || []).length === 0 && (
              <li className="muted">暂无投影页</li>
            )}
          </ul>
        </div>

        <div>
          <h2>详情</h2>
          {!selectedId && <p className="muted">选择左侧 Wiki 页查看正文。</p>}
          {pageQuery.isError && <p className="error">加载详情失败</p>}
          {pageQuery.data && (
            <article data-testid="knowledge-wiki-detail">
              <h3>{pageQuery.data.title}</h3>
              <p>
                <code>{pageQuery.data.contextRef}</code>
                {pageQuery.data.layer ? ` · ${pageQuery.data.layer}` : ""}
              </p>
              <pre style={{ whiteSpace: "pre-wrap", fontSize: "0.9rem" }}>{pageQuery.data.body}</pre>
            </article>
          )}
        </div>
      </div>
    </section>
  );
}
