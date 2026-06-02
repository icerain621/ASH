# One-time layout normalization for ash monorepo (idempotent)
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
if ((Split-Path -Leaf $root) -eq "scripts") {
    $root = Split-Path -Parent $root
}

Set-Location $root

function Ensure-Dir($path) {
    if (-not (Test-Path $path)) { New-Item -ItemType Directory -Path $path -Force | Out-Null }
}

function Move-IfExists($src, $destDir) {
    if (-not (Test-Path $src)) { return }
    $name = Split-Path -Leaf $src
    $dest = Join-Path $destDir $name
    if (Test-Path $dest) {
        Write-Host "skip $name (already in $destDir)"
        return
    }
    Move-Item -Path $src -Destination $destDir -Force
    Write-Host "moved $name -> $destDir/"
}

Ensure-Dir "$root\backend"
Ensure-Dir "$root\doc\product"
Ensure-Dir "$root\doc\design"
Ensure-Dir "$root\reports"
Ensure-Dir "$root\frontend"
Ensure-Dir "$root\scripts"

foreach ($item in @("cmd", "internal", "scenarios", "go.sum")) {
    Move-IfExists (Join-Path $root $item) "$root\backend"
}

foreach ($item in @("go.mod", "Makefile")) {
    $src = Join-Path $root $item
    $dest = Join-Path "$root\backend" $item
    if (Test-Path $src) {
        if (-not (Test-Path $dest)) {
            Move-Item $src $dest -Force
            Write-Host "moved $item -> backend/"
        } elseif $item -eq "Makefile" {
            Remove-Item $src -Force
            Write-Host "removed duplicate root Makefile (backend/ has it)"
        }
    }
}

$product = @(
    "01-ai-product-research.md", "02-formal-prd-template.md", "03-ash-product-requirements-design.md",
    "04-api-event-state.md", "05-db-design-and-api-examples.md", "06-iteration-plan-10-weeks.md",
    "07-tech-stack-decision.md", "08-frontend-architecture.md", "09-backend-architecture-go.md",
    "10-engineering-setup.md", "11-mvp-release-checklist.md", "12-risk-register.md",
    "13-milestone-review-template.md", "14-kpi-dashboard-definition.md", "15-jira-epic-story-breakdown.md",
    "16-jira-import.csv", "17-jira-import-tasks.csv"
)
foreach ($f in $product) {
    $src = Join-Path "$root\doc" $f
    if (Test-Path $src) { Move-Item $src "$root\doc\product\" -Force; Write-Host "moved doc/$f" }
}

foreach ($f in @("01-PRD-需求文档.md", "02-HLD-总体设计.md", "03-ARCH-架构与技术选型.md", "04-PLAN-进度与里程碑.md")) {
    $src = Join-Path "$root\doc" $f
    if (Test-Path $src) { Move-Item $src "$root\doc\design\" -Force; Write-Host "moved doc/$f" }
}

if ((Test-Path "$root\doc\appendices") -and -not (Test-Path "$root\doc\design\appendices")) {
    Move-Item "$root\doc\appendices" "$root\doc\design\" -Force
    Write-Host "moved doc/appendices -> doc/design/"
}

if (Test-Path "$root\doc\AI商业分析报告-ash_repwiki.md") {
    Move-Item "$root\doc\AI商业分析报告-ash_repwiki.md" "$root\reports\" -Force
}

Write-Host ""
Write-Host "Reorganized $root"
Get-ChildItem $root -Name
