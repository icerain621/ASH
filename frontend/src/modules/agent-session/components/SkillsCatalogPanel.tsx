import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  installSkillFromCatalog,
  installSkillPack,
  listSkillCatalog,
  listSkills,
  verifySkillPack,
  type SkillItem,
} from "@/modules/skills/api/skills.api";

type Props = {
  open: boolean;
  onClose: () => void;
  /** Run skill as session slash command when a session is active. */
  onRunSkill?: (skillSlash: string) => void;
  canRun?: boolean;
};

/** Agent Chat first-class Skills catalog (browse / pack install / run via slash). */
export function SkillsCatalogPanel({ open, onClose, onRunSkill, canRun = false }: Props) {
  const qc = useQueryClient();
  const [repoRoot, setRepoRoot] = useState(".");
  const [packPath, setPackPath] = useState("");
  const [packSig, setPackSig] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const skillsQuery = useQuery({
    queryKey: ["agent-skills", repoRoot],
    queryFn: () => listSkills(repoRoot),
    enabled: open,
  });

  const catalogQuery = useQuery({
    queryKey: ["agent-skills-catalog", repoRoot],
    queryFn: () => listSkillCatalog(repoRoot),
    enabled: open,
  });

  const installedIds = new Set((skillsQuery.data?.items ?? []).map((s) => s.id));

  const refreshSkills = () => {
    void qc.invalidateQueries({ queryKey: ["agent-skills"] });
    void qc.invalidateQueries({ queryKey: ["agent-skills-catalog"] });
    void qc.invalidateQueries({ queryKey: ["agent-commands"] });
  };

  const verifyMut = useMutation({
    mutationFn: () =>
      verifySkillPack({
        repoRoot,
        packPath: packPath.trim(),
        signature: packSig.trim(),
      }),
    onSuccess: (res) => {
      setError("");
      setMessage(
        res.ok
          ? `验签通过：${res.name || "?"}@${res.version || "?"} · ${res.digest || ""}`
          : res.message || "验签未通过",
      );
    },
    onError: (e: Error) => {
      setMessage("");
      setError(e.message);
    },
  });

  const installPackMut = useMutation({
    mutationFn: () =>
      installSkillPack({
        repoRoot,
        packPath: packPath.trim(),
        signature: packSig.trim(),
      }),
    onSuccess: (res) => {
      setError("");
      setMessage(`已安装 ${res.name}@${res.version} → ${res.path}`);
      setPackPath("");
      setPackSig("");
      refreshSkills();
    },
    onError: (e: Error) => {
      setMessage("");
      setError(e.message);
    },
  });

  const installCatalogMut = useMutation({
    mutationFn: (item: { name: string; version?: string }) =>
      installSkillFromCatalog({
        repoRoot,
        name: item.name,
        version: item.version,
      }),
    onSuccess: (res) => {
      setError("");
      setMessage(`Catalog 安装 ${res.name}@${res.version}`);
      refreshSkills();
    },
    onError: (e: Error) => {
      setMessage("");
      setError(e.message);
    },
  });

  if (!open) return null;

  const items: SkillItem[] = skillsQuery.data?.items ?? [];
  const catalogItems = catalogQuery.data?.items ?? [];

  return (
    <div className="agent-mcp-overlay" data-testid="agent-skills-panel" role="dialog" aria-modal="true">
      <div className="agent-mcp-panel">
        <div className="pane-title">
          <h2>Skills</h2>
          <button type="button" className="btn mini" data-testid="agent-skills-close" onClick={onClose}>
            关闭
          </button>
        </div>
        <p className="muted-line">
          扫描 <code>.ash/skills/*/SKILL.md</code>。Chat 用 <code>/skillId</code>，或点「运行」。签名 pack
          可在此验签/安装（无公网市场）。
        </p>

        <label className="wide-field">
          repoRoot
          <input
            value={repoRoot}
            data-testid="agent-skills-repo-root"
            onChange={(e) => setRepoRoot(e.target.value)}
          />
        </label>

        <div className="agent-skills-pack" data-testid="agent-skills-pack">
          <label className="wide-field">
            packPath
            <input
              value={packPath}
              placeholder="/path/to/skill.ash-skill.zip"
              data-testid="agent-skills-pack-path"
              onChange={(e) => setPackPath(e.target.value)}
            />
          </label>
          <label className="wide-field">
            signature
            <input
              value={packSig}
              placeholder="hmac hex"
              data-testid="agent-skills-pack-sig"
              onChange={(e) => setPackSig(e.target.value)}
            />
          </label>
          <div className="agent-skills-pack-actions">
            <button
              type="button"
              className="btn mini"
              data-testid="agent-skills-pack-verify"
              disabled={!packPath.trim() || !packSig.trim() || verifyMut.isPending}
              onClick={() => verifyMut.mutate()}
            >
              验签
            </button>
            <button
              type="button"
              className="btn mini ok"
              data-testid="agent-skills-pack-install"
              disabled={!packPath.trim() || !packSig.trim() || installPackMut.isPending}
              onClick={() => installPackMut.mutate()}
            >
              安装 pack
            </button>
          </div>
        </div>

        {error ? (
          <p className="error-text" data-testid="agent-skills-error">
            {error}
          </p>
        ) : null}
        {message ? (
          <p className="muted-line" data-testid="agent-skills-message">
            {message}
          </p>
        ) : null}

        <div className="agent-skills-section" data-testid="agent-skills-org-catalog">
          <h3 className="agent-skills-section-title">组织 Catalog</h3>
          {catalogQuery.data?.source ? (
            <p className="muted-line">
              source: <code>{catalogQuery.data.source}</code>
            </p>
          ) : null}
          <ul className="agent-mcp-list">
            {catalogItems.map((it) => {
              const installed = installedIds.has(it.name);
              return (
                <li
                  key={`${it.publisher}/${it.name}@${it.version}`}
                  className="agent-mcp-row"
                  data-testid={`agent-skills-catalog-row-${it.name}`}
                >
                  <div>
                    <strong>{it.name}</strong>
                    <span className="muted-line">
                      {it.version} · {it.publisher}
                      {installed ? " · 已安装" : ""}
                    </span>
                  </div>
                  <button
                    type="button"
                    className="btn mini"
                    data-testid={`agent-skills-catalog-install-${it.name}`}
                    disabled={installCatalogMut.isPending}
                    onClick={() =>
                      installCatalogMut.mutate({ name: it.name, version: it.version })
                    }
                  >
                    {installed ? "重装" : "安装"}
                  </button>
                </li>
              );
            })}
            {!catalogQuery.isLoading && catalogItems.length === 0 ? (
              <li className="muted-line">暂无 catalog（可放 .ash/skill-catalog.json）</li>
            ) : null}
          </ul>
        </div>

        <div className="agent-skills-section">
          <h3 className="agent-skills-section-title">已安装</h3>
          <ul className="agent-mcp-list" data-testid="agent-skills-list">
            {items.map((sk) => {
              const slash = sk.id.startsWith("/") ? sk.id : `/${sk.id}`;
              return (
                <li key={sk.id} className="agent-mcp-row" data-testid={`agent-skills-row-${sk.id}`}>
                  <div>
                    <strong>{sk.name || sk.id}</strong>
                    <span className="muted-line">
                      {slash}
                      {sk.description ? ` · ${sk.description}` : ""}
                    </span>
                  </div>
                  <button
                    type="button"
                    className="btn mini ok"
                    data-testid={`agent-skills-run-${sk.id}`}
                    disabled={!canRun || !onRunSkill}
                    title={canRun ? `运行 ${slash}` : "先选择会话"}
                    onClick={() => onRunSkill?.(slash)}
                  >
                    运行
                  </button>
                </li>
              );
            })}
            {!skillsQuery.isLoading && items.length === 0 ? (
              <li className="muted-line">未发现 Skill（可放在 .ash/skills/&lt;name&gt;/）</li>
            ) : null}
          </ul>
        </div>
      </div>
    </div>
  );
}
