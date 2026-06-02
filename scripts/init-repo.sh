#!/usr/bin/env bash
# Initialize git repo and push to GitHub (Git Bash / Linux / macOS)
# Works without GitHub CLI (gh): commits locally, then prints manual push steps.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

REPO_NAME="${GH_REPO_NAME:-ash}"
# Optional: GH_OWNER=your-user or ash-repwiki
# Optional: GITHUB_REMOTE=git@github.com:org/ash.git (skip gh create, add remote directly)

COMMIT_MSG="$(cat <<'EOF'
Initial M0: Go worker API, rules engine, run control, memory, doctor, React UI.

Includes Gin/GORM/SQLite worker, ToolBus, artifacts bundle, SSE events,
resume/replay API, memory review, TR0 doctor, Vite React console, and docs.
EOF
)"

if [[ ! -d .git ]]; then
  git init -b main
  echo "git init -b main"
else
  echo "git repo already exists"
fi

git add -A
if [[ -n "$(git status --porcelain)" ]]; then
  git commit -m "$COMMIT_MSG"
  echo "committed: $(git rev-parse HEAD)"
else
  echo "nothing new to commit"
fi

BRANCH="$(git branch --show-current)"

setup_remote() {
  if git remote get-url origin >/dev/null 2>&1; then
    echo "remote origin: $(git remote get-url origin)"
    return 0
  fi

  if [[ -n "${GITHUB_REMOTE:-}" ]]; then
    git remote add origin "$GITHUB_REMOTE"
    echo "added origin: $GITHUB_REMOTE"
    return 0
  fi

  if command -v gh >/dev/null 2>&1; then
    echo "Creating GitHub repo with gh..."
    if [[ -n "${GH_OWNER:-}" ]]; then
      gh repo create "${GH_OWNER}/${REPO_NAME}" --source=. --remote=origin \
        --description "ASH delivery-oriented coding robot (M0)"
    else
      gh repo create "$REPO_NAME" --source=. --remote=origin \
        --description "ASH delivery-oriented coding robot (M0)"
    fi
    return 0
  fi

  echo ""
  echo "=== gh not found — create the remote repo manually ==="
  echo ""
  echo "1) Open https://github.com/new"
  echo "   Repository name: ${REPO_NAME}"
  echo "   Do NOT initialize with README / .gitignore / license"
  echo ""
  if [[ -n "${GH_OWNER:-}" ]]; then
    echo "2) Then run:"
    echo "   git remote add origin git@github.com:${GH_OWNER}/${REPO_NAME}.git"
    echo "   git push -u origin ${BRANCH}"
  else
    echo "2) Then run (replace YOUR_USER):"
    echo "   git remote add origin git@github.com:YOUR_USER/${REPO_NAME}.git"
    echo "   git push -u origin ${BRANCH}"
    echo ""
    echo "   Or set owner and re-run this script:"
    echo "   GH_OWNER=YOUR_USER bash scripts/init-repo.sh"
    echo ""
    echo "   Or set full remote URL:"
    echo "   GITHUB_REMOTE=git@github.com:YOUR_USER/${REPO_NAME}.git bash scripts/init-repo.sh"
  fi
  echo ""
  echo "Install GitHub CLI (optional): https://cli.github.com/"
  return 1
}

if setup_remote; then
  git push -u origin "$BRANCH"
  echo ""
  echo "=== done ==="
  git remote get-url origin
  git rev-parse HEAD
  git status -sb
else
  echo ""
  echo "=== local commit ready; add remote and push (see above) ==="
  git rev-parse HEAD 2>/dev/null || true
  git status -sb
  exit 0
fi
