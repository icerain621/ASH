/** Move fromId next to toId within an ordered id list (HTML5 DnD helper). */
export function reorderSessionIds(ids: string[], fromId: string, toId: string): string[] {
  if (fromId === toId) return ids.slice();
  const from = ids.indexOf(fromId);
  const to = ids.indexOf(toId);
  if (from < 0 || to < 0) return ids.slice();
  const next = ids.slice();
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item);
  return next;
}
