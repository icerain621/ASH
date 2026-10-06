/** Map common Interaction API English errors to console-facing Chinese copy. */
export function formatInteractionError(message: string): string {
  const raw = message.trim();
  if (!raw) return raw;
  if (/^run not found$/i.test(raw)) return "未找到该 Run，请检查 Run ID";
  if (/^thread not found$/i.test(raw)) return "未找到该 Thread";
  return raw;
}
