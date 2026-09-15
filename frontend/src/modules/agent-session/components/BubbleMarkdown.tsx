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

/**
 * Lightweight GFM-ish renderer (fences / inline code / bold / italic / paragraphs).
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
          {raw
            .slice(last, m.index)
            .split(/\n{2,}/)
            .filter((p) => p.length > 0)
            .map((para, i) => (
              <p key={i}>{renderInline(para.replace(/\n/g, " "))}</p>
            ))}
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
        {raw
          .slice(last)
          .split(/\n{2,}/)
          .filter((p) => p.length > 0)
          .map((para, i) => (
            <p key={i}>{renderInline(para.replace(/\n/g, " "))}</p>
          ))}
      </div>,
    );
  }
  return (
    <div className="ash-md" data-testid="agent-chat-markdown">
      {parts}
    </div>
  );
}
