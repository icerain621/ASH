import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { listSkills, type SkillItem } from "@/modules/skills/api/skills.api";

type Props = {
  open: boolean;
  onClose: () => void;
  /** Run skill as session slash command when a session is active. */
  onRunSkill?: (skillSlash: string) => void;
  canRun?: boolean;
};

/** Agent Chat first-class Skills catalog (browse / run via slash). */
export function SkillsCatalogPanel({ open, onClose, onRunSkill, canRun = false }: Props) {
  const [repoRoot, setRepoRoot] = useState(".");
  const skillsQuery = useQuery({
    queryKey: ["agent-skills", repoRoot],
    queryFn: () => listSkills(repoRoot),
    enabled: open,
  });

  if (!open) return null;

  const items: SkillItem[] = skillsQuery.data?.items ?? [];

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
          扫描 <code>.ash/skills/*/SKILL.md</code>。在 Chat 用 <code>/skillId</code>，或点「运行」写入当前会话。
        </p>
        <label className="wide-field">
          repoRoot
          <input
            value={repoRoot}
            data-testid="agent-skills-repo-root"
            onChange={(e) => setRepoRoot(e.target.value)}
          />
        </label>
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
  );
}
