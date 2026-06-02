# Initialize git repo and push to GitHub (run from repo root)
# Works without GitHub CLI (gh): commits locally, then prints manual push steps.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
if ((Split-Path -Leaf $root) -eq "scripts") { $root = Split-Path -Parent $root }
Set-Location $root

$repoName = if ($env:GH_REPO_NAME) { $env:GH_REPO_NAME } else { "ash" }
$owner = $env:GH_OWNER
$remoteUrl = $env:GITHUB_REMOTE

if (-not (Test-Path ".git")) {
    git init -b main
    Write-Host "git init -b main"
} else {
    Write-Host "git repo already exists"
}

git add -A
$status = git status --porcelain
if ($status) {
    $msg = @"
Initial M0: Go worker API, rules engine, run control, memory, doctor, React UI.

Includes Gin/GORM/SQLite worker, ToolBus, artifacts bundle, SSE events,
resume/replay API, memory review, TR0 doctor, Vite React console, and docs.
"@
    git commit -m $msg
    Write-Host "committed: $(git rev-parse HEAD)"
} else {
    Write-Host "nothing new to commit"
}

$branch = git branch --show-current
$hasOrigin = $false
try {
    git remote get-url origin 2>$null | Out-Null
    $hasOrigin = $true
    Write-Host "remote origin: $(git remote get-url origin)"
} catch {
    $hasOrigin = $false
}

if (-not $hasOrigin) {
    if ($remoteUrl) {
        git remote add origin $remoteUrl
        Write-Host "added origin: $remoteUrl"
        $hasOrigin = $true
    } elseif (Get-Command gh -ErrorAction SilentlyContinue) {
        Write-Host "Creating GitHub repo with gh..."
        if ($owner) {
            gh repo create "${owner}/${repoName}" --source=. --remote=origin --description "ASH delivery-oriented coding robot (M0)"
        } else {
            gh repo create $repoName --source=. --remote=origin --description "ASH delivery-oriented coding robot (M0)"
        }
        $hasOrigin = $true
    } else {
        Write-Host ""
        Write-Host "=== gh not found — create the remote repo manually ==="
        Write-Host ""
        Write-Host "1) Open https://github.com/new"
        Write-Host "   Repository name: $repoName"
        Write-Host "   Do NOT initialize with README / .gitignore / license"
        Write-Host ""
        if ($owner) {
            Write-Host "2) Then run:"
            Write-Host "   git remote add origin git@github.com:${owner}/${repoName}.git"
            Write-Host "   git push -u origin $branch"
        } else {
            Write-Host "2) Then run (replace YOUR_USER):"
            Write-Host "   git remote add origin git@github.com:YOUR_USER/${repoName}.git"
            Write-Host "   git push -u origin $branch"
            Write-Host ""
            Write-Host "   Or: `$env:GH_OWNER='YOUR_USER'; .\scripts\init-repo.ps1"
        }
        Write-Host ""
        Write-Host "Install GitHub CLI (optional): https://cli.github.com/"
        exit 0
    }
}

if ($hasOrigin) {
    git push -u origin $branch
    Write-Host ""
    Write-Host "=== done ==="
    git remote get-url origin
    git rev-parse HEAD
    git status -sb
}
