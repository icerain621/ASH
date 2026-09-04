-- RAG symbol source tag: regex | ctags (Sprint DX16). No new RLS table.
ALTER TABLE rag_symbols ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'regex';
