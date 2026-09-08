# v3.1 DX51 — Quest 工作台硬化（门禁 + 产物）

> Status: **implemented** (2026-09-08)  
> Program: T2 Quest · **no new tables**

## Goals

- When selected Run is `waiting_approval`, show a **gate panel** (reason from timeline `gate.waiting_approval`) with **Approve** + **Cancel**
- Show **Artifacts** pane for the selected Run (list + optional signed URL)
- Keep Diff reject actions from DX50; gate Approve remains available there too

## Non-goals

- Checkpoints UI (optional later)
- Webhook Plan unification (DX52)
- Live board / plan.* SSE (DX53)
