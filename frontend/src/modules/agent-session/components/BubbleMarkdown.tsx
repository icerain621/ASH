import type { ReactNode } from "react";

function renderInline(text: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  const re = /(`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*)/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let key = 0;
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) {
      nodes.push(text.slice(last, m.index));
    }
    const tok = m[0];
    if (tok.startsWith("`")) {
      nodes.push(
        <code key={key++} className="ash-md-code">
          {tok.slice(1, -1)}
        </code>,
      );
    } else if (tok.startsWith("**")) {
      nodes.push(<strong key={key++}>{tok.slice(2, -2)}</strong>);
    } else {
      nodes.push(<em key={key++}>{tok.slice(1, -1)}</em>);
    }
    last = m.index + tok.length;
  }
  if (last < text.length) {
    nodes.push(text.slice(last));
  }
  return nodes;
}

function splitCells(line: string): string[] {
  const trimmed = line.trim().replace(/^\|/, "").replace(/\|$/, "");
  return trimmed.split("|").map((c) => c.trim());
}

function isTableSep(line: string): boolean {
  const cells = splitCells(line);
  if (cells.length < 2) return false;
  return cells.every((c) => /^:?-{3,}:?$/.test(c));
}

function isTableRow(line: string): boolean {
  const t = line.trim();
  return t.includes("|") && t.length > 1;
}

function renderTable(rows: string[], key: number): ReactNode {
  if (rows.length < 2) return null;
  const header = splitCells(rows[0]);
  const body = rows.slice(2).map(splitCells);
  return (
    <div key={key} className="ash-md-table-wrap">
      <table className="ash-md-table" data-testid="agent-chat-md-table">
        <thead>
          <tr>
            {header.map((cell, i) => (
              <th key={i}>{renderInline(cell)}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {body.map((cells, ri) => (
            <tr key={ri}>
              {header.map((_, ci) => (
                <td key={ci}>{renderInline(cells[ci] ?? "")}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

const taskRe = /^(\s*)([-*])\s+\[([ xX])\]\s+(.*)$/;
const bulletRe = /^(\s*)([-*])\s+(.*)$/;

function renderProseBlock(text: string, keyBase: number): ReactNode[] {
  const lines = text.replace(/\r\n/g, "\n").split("\n");
  const out: ReactNode[] = [];
  let i = 0;
  let key = keyBase;

  while (i < lines.length) {
    const line = lines[i];
    if (!line.trim()) {
      i++;
      continue;
    }

    // GFM table: header + separator + body rows
    if (
      isTableRow(line) &&
      i + 1 < lines.length &&
      isTableSep(lines[i + 1])
    ) {
      const tableLines = [line, lines[i + 1]];
      i += 2;
      while (i < lines.length && isTableRow(lines[i]) && !isTableSep(lines[i])) {
        tableLines.push(lines[i]);
        i++;
      }
      const table = renderTable(tableLines, key++);
      if (table) out.push(table);
      continue;
    }

    // Task list
    if (taskRe.test(line)) {
      const items: ReactNode[] = [];
      while (i < lines.length) {
        const m = lines[i].match(taskRe);
        if (!m) break;
        const checked = m[3].toLowerCase() === "x";
        items.push(
          <li key={items.length} className={checked ? "checked" : undefined} data-checked={checked ? "1" : "0"}>
            <input type="checkbox" checked={checked} readOnly disabled />
            <span>{renderInline(m[4])}</span>
          </li>,
        );
        i++;
      }
      out.push(
        <ul key={key++} className="ash-md-task-list" data-testid="agent-chat-md-tasks">
          {items}
        </ul>,
      );
      continue;
    }

    // Bullet list
    if (bulletRe.test(line) && !taskRe.test(line)) {
      const items: ReactNode[] = [];
      while (i < lines.length) {
        const m = lines[i].match(bulletRe);
        if (!m || taskRe.test(lines[i])) break;
        items.push(<li key={items.length}>{renderInline(m[3])}</li>);
        i++;
      }
      out.push(
        <ul key={key++} className="ash-md-list">
          {items}
        </ul>,
      );
      continue;
    }

    // Paragraph: gather until blank / table / list
    const para: string[] = [];
    while (i < lines.length) {
      const cur = lines[i];
      if (!cur.trim()) break;
      if (isTableRow(cur) && i + 1 < lines.length && isTableSep(lines[i + 1])) break;
      if (taskRe.test(cur) || bulletRe.test(cur)) break;
      para.push(cur);
      i++;
    }
    if (para.length) {
      out.push(<p key={key++}>{renderInline(para.join(" "))}</p>);
    }
  }

  return out;
}

/**
 * Lightweight GFM-ish renderer: fences, tables, task lists, bullets, inline marks.
 * No npm deps; React text nodes escape HTML automatically.
 */
export function BubbleMarkdown({ text }: { text: string }) {
  const raw = text ?? "";
  if (!raw.trim()) {
    return null;
  }
  const parts: ReactNode[] = [];
  const fenceRe = /```([a-zA-Z0-9_-]*)\n?([\s\S]*?)```/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let key = 0;
  while ((m = fenceRe.exec(raw)) !== null) {
    if (m.index > last) {
      parts.push(
        <div key={key++} className="ash-md-block">
          {renderProseBlock(raw.slice(last, m.index), key * 100)}
        </div>,
      );
    }
    const lang = m[1] || undefined;
    parts.push(
      <pre key={key++} className="ash-md-pre" data-lang={lang || undefined}>
        <code>{m[2].replace(/\n$/, "")}</code>
      </pre>,
    );
    last = m.index + m[0].length;
  }
  if (last < raw.length) {
    parts.push(
      <div key={key++} className="ash-md-block">
        {renderProseBlock(raw.slice(last), key * 100)}
      </div>,
    );
  }
  return (
    <div className="ash-md" data-testid="agent-chat-markdown">
      {parts}
    </div>
  );
}
