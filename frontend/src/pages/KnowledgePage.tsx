import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BookOpen, RefreshCcw } from "lucide-react";
import { useState } from "react";
import { getRepoProfile, getWikiPage, listWikiPages } from "@/modules/knowledge/api/knowledge.api";
import { getRagProfile, rebuildRAGSymbols } from "@/modules/observability/api/observability.api";
import { getCurrentSpaceId } from "@/services/http/client";

export function KnowledgePage() {
  const [repoRoot, setRepoRoot] = useState(".");
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const activeSpaceId = getCurrentSpaceId();
  const qc = useQueryClient();

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
  const ragQuery = useQuery({
    queryKey: ["rag-profile", activeSpaceId],
    queryFn: getRagProfile,
  });
  const rebuildMut = useMutation({
    mutationFn: () => rebuildRAGSymbols({ repoRoot, spaceId: activeSpaceId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["rag-profile", activeSpaceId] });
    },
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
          <p>即时 Repo Profile 与 Wiki 投影（不落库）；Run 准备阶段可注入 profile: / wiki: contextRefs，并触发 Hybrid 符号重建。</p>
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
              ragQuery.refetch();
            }}
            disabled={profileQuery.isFetching || wikiQuery.isFetching || ragQuery.isFetching}
          >
            <RefreshCcw size={16} />
            刷新
          </button>
        </div>
      </div>

      <div className="split-pane" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1rem" }}>
        <div>
          <h2>RAG Hybrid</h2>
          {ragQuery.isError && <p className="error">加载 RAG Profile 失败</p>}
          {ragQuery.data && (
            <div className="card-like" data-testid="knowledge-rag-hybrid">
              <p>
                模式：<code>{ragQuery.data.defaultRetrievalMode}</code>
                {ragQuery.data.hybridAvailable ? " · Hybrid 可用" : " · Hybrid 未建索引"}
              </p>
              <p>
                文档 {ragQuery.data.documentCount} · 分块 {ragQuery.data.chunkCount} · 路径{" "}
                {ragQuery.data.pathEntryCount ?? 0} · 符号 {ragQuery.data.symbolCount ?? 0}
              </p>
              <p className="muted">
                FTS {ragQuery.data.ftsAvailable ? "可用" : "不可用"}
                {ragQuery.data.ftsEngine ? ` (${ragQuery.data.ftsEngine})` : ""}
              </p>
              <button
                type="button"
                className="btn"
                data-testid="knowledge-rag-rebuild"
                disabled={rebuildMut.isPending}
                onClick={() => rebuildMut.mutate()}
              >
                {rebuildMut.isPending ? "重建中…" : "重建符号/路径索引"}
              </button>
              {rebuildMut.isSuccess && rebuildMut.data && (
                <p className="muted" data-testid="knowledge-rag-rebuild-result">
                  已写入 paths={rebuildMut.data.paths} symbols={rebuildMut.data.symbols} files=
                  {rebuildMut.data.files}
                </p>
              )}
              {rebuildMut.isError && <p className="error">重建失败</p>}
            </div>
          )}

          <h2 style={{ marginTop: "1.25rem" }}>Repo Profile</h2>
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
