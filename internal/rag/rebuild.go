package rag

import (
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

type RebuildSymbolsRequest struct {
	RepoRoot string `json:"repoRoot" binding:"required"`
	SpaceID  string `json:"spaceId,omitempty"`
}

type RebuildSymbolsResponse struct {
	Paths        int    `json:"paths"`
	Symbols      int    `json:"symbols"`
	Files        int    `json:"files"`
	SymbolSource string `json:"symbolSource"`
}

func (s *Service) RebuildSymbols(req RebuildSymbolsRequest) (*RebuildSymbolsResponse, error) {
	abs, err := AbsRepoRoot(req.RepoRoot)
	if err != nil {
		return nil, err
	}
	space := firstNonEmpty(req.SpaceID, "local")

	seenPaths := make(map[string]bool)
	indexer := ResolveSymbolIndexer()
	regexFallback := RegexIndexer{}
	resp := &RebuildSymbolsResponse{SymbolSource: indexer.Name()}
	now := time.Now().UTC()

	err = filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isIndexable(path) {
			return nil
		}

		rel, _ := filepath.Rel(abs, path)
		rel = filepath.ToSlash(rel)

		info, err := os.Stat(path)
		if err != nil || info.Size() > 512*1024 {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil || looksBinary(b) {
			return nil
		}
		resp.Files++
		seenPaths[rel] = true
		digest := digestBytes(b)
		basename := filepath.Base(rel)

		var entry store.RAGPathEntry
		err = s.gdb().Where("space_id = ? AND repo_root = ? AND path = ?", space, abs, rel).First(&entry).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			entry = store.RAGPathEntry{
				ID: "ragpath_" + uuid.NewString(), SpaceID: space, RepoRoot: abs, Path: rel,
				Basename: basename, Digest: digest, CreatedAt: now, UpdatedAt: now,
			}
			if err := s.gdb().Create(&entry).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			entry.Basename = basename
			entry.Digest = digest
			entry.UpdatedAt = now
			if err := s.gdb().Save(&entry).Error; err != nil {
				return err
			}
		}
		resp.Paths++

		if err := s.gdb().Where("space_id = ? AND repo_root = ? AND path = ?", space, abs, rel).
			Delete(&store.RAGSymbol{}).Error; err != nil {
			return err
		}

		source := indexer.Name()
		hits, err := indexer.IndexFile(path, b)
		if err != nil {
			source = regexFallback.Name()
			hits, err = regexFallback.IndexFile(path, b)
			if err != nil {
				return err
			}
			resp.SymbolSource = source
		}
		for _, hit := range hits {
			sym := store.RAGSymbol{
				ID: "ragsym_" + uuid.NewString(), SpaceID: space, RepoRoot: abs, Path: rel,
				Name: hit.Name, Kind: hit.Kind, Line: hit.Line, Source: source, Digest: digest,
				CreatedAt: now, UpdatedAt: now,
			}
			if err := s.gdb().Create(&sym).Error; err != nil {
				return err
			}
			resp.Symbols++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := deleteStaleRAGPaths(s.gdb(), space, abs, seenPaths); err != nil {
		return nil, err
	}
	if err := deleteStaleRAGSymbols(s.gdb(), space, abs, seenPaths); err != nil {
		return nil, err
	}
	return resp, nil
}

func deleteStaleRAGPaths(db *gorm.DB, space, root string, seen map[string]bool) error {
	q := db.Where("space_id = ? AND repo_root = ?", space, root)
	if len(seen) > 0 {
		paths := make([]string, 0, len(seen))
		for p := range seen {
			paths = append(paths, p)
		}
		q = q.Where("path NOT IN ?", paths)
	}
	return q.Delete(&store.RAGPathEntry{}).Error
}

func deleteStaleRAGSymbols(db *gorm.DB, space, root string, seen map[string]bool) error {
	q := db.Where("space_id = ? AND repo_root = ?", space, root)
	if len(seen) > 0 {
		paths := make([]string, 0, len(seen))
		for p := range seen {
			paths = append(paths, p)
		}
		q = q.Where("path NOT IN ?", paths)
	}
	return q.Delete(&store.RAGSymbol{}).Error
}
