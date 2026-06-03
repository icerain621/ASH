const API = "/api/v1";

async function api(path, opts = {}) {
  const res = await fetch(API + path, {
    headers: { "Content-Type": "application/json", ...(opts.headers || {}) },
    ...opts,
  });
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = text; }
  if (!res.ok) {
    const msg = data?.error?.message || data?.error || res.statusText;
    throw new Error(msg);
  }
  return data;
}

function fmtTime(ms) {
  if (!ms) return "—";
  return new Date(ms).toLocaleString();
}

function shortId(id) {
  if (!id || id.length < 16) return id || "";
  return id.slice(0, 14) + "…";
}

// --- Tabs ---
document.querySelectorAll(".tab").forEach((btn) => {
  btn.addEventListener("click", () => {
    document.querySelectorAll(".tab").forEach((b) => b.classList.remove("active"));
    document.querySelectorAll(".panel").forEach((p) => p.classList.remove("active"));
    btn.classList.add("active");
    document.getElementById("panel-" + btn.dataset.tab).classList.add("active");
  });
});

// --- Runs ---
let selectedRunId = null;
let sseSource = null;

async function loadRuns() {
  const data = await api("/runs?limit=30");
  const tbody = document.querySelector("#runs-table tbody");
  tbody.innerHTML = "";
  for (const r of data.items || []) {
    const tr = document.createElement("tr");
    if (r.runId === selectedRunId) tr.classList.add("selected");
    tr.innerHTML = `
      <td title="${r.runId}">${shortId(r.runId)}</td>
      <td>${r.status}</td>
      <td>${r.scenario?.name || ""}</td>
      <td>${fmtTime(r.startedAt)}</td>`;
    tr.onclick = () => selectRun(r.runId);
    tbody.appendChild(tr);
  }
}

async function selectRun(runId) {
  selectedRunId = runId;
  document.getElementById("run-detail-id").textContent = runId;
  document.querySelectorAll("#runs-table tbody tr").forEach((tr) => {
    tr.classList.toggle("selected", tr.textContent.includes(shortId(runId).replace("…", "")));
  });

  const sum = await api("/runs/" + runId);
  document.getElementById("run-summary").textContent = JSON.stringify(sum, null, 2);

  try {
    const art = await api("/runs/" + runId + "/artifacts");
    document.getElementById("run-artifacts").textContent = JSON.stringify(art, null, 2);
  } catch {
    document.getElementById("run-artifacts").textContent = "—";
  }

  streamRun(runId);
}

function streamRun(runId) {
  if (sseSource) {
    sseSource.close();
    sseSource = null;
  }
  const log = document.getElementById("event-log");
  log.innerHTML = "";

  const url = API + "/runs/" + runId + "/stream";
  sseSource = new EventSource(url);

  const append = (type, raw) => {
    const line = document.createElement("div");
    line.className = "event-line" + (type.startsWith("memory.") ? " memory" : "") + (type.includes("failed") ? " error" : "");
    let payload = raw;
    try { payload = JSON.stringify(JSON.parse(raw), null, 0); } catch { /* keep raw */ }
    line.innerHTML = `<span class="type">${type}</span> ${payload}`;
    log.appendChild(line);
    log.scrollTop = log.scrollHeight;
  };

  sseSource.onmessage = (ev) => append(ev.type || "message", ev.data);
  sseSource.addEventListener("run.started", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("step.started", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("step.finished", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("tool.called", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("tool.result", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("run.finished", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("run.failed", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("memory.candidate_created", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("memory.review_requested", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("memory.reviewed", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("run.checkpoint_saved", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("policy.denied", (ev) => append(ev.type, ev.data));
  sseSource.addEventListener("memory.deprecated", (ev) => append(ev.type, ev.data));
  sseSource.onerror = () => {
    append("sse", "connection closed or error");
  };
}

document.getElementById("btn-refresh-runs").onclick = () => loadRuns().catch(alert);

document.getElementById("btn-new-run").onclick = async () => {
  try {
    const body = {
      scenario: { name: "feature_delivery", scenarioVersion: "1.0.0" },
      inputs: {
        issueOrSpec: "UI demo run " + new Date().toISOString(),
        repoRoot: ".",
      },
    };
    const res = await api("/runs", { method: "POST", body: JSON.stringify(body) });
    await loadRuns();
    await selectRun(res.runId);
  } catch (e) {
    alert(e.message);
  }
};

// --- Memory ---
async function loadMemory() {
  const data = await api("/memory/candidates?limit=50");
  const tbody = document.querySelector("#memory-table tbody");
  tbody.innerHTML = "";
  for (const m of data.items || []) {
    const tr = document.createElement("tr");
    const actions = m.status === "candidate"
      ? `<button class="btn small ok" data-action="approve" data-id="${m.id}">✓</button>
         <button class="btn small err" data-action="reject" data-id="${m.id}">✗</button>`
      : "";
    tr.innerHTML = `
      <td title="${m.id}">${shortId(m.id)}</td>
      <td>${m.layer}</td>
      <td>${m.title}</td>
      <td>${m.status}</td>
      <td>${actions}</td>`;
    tbody.appendChild(tr);
  }
  tbody.querySelectorAll("[data-action]").forEach((btn) => {
    btn.onclick = async (e) => {
      e.stopPropagation();
      const id = btn.dataset.id;
      const decision = btn.dataset.action === "approve" ? "approve" : "reject";
      const runIdInput = document.querySelector('#memory-form [name="runId"]');
      const body = {
        decision,
        reason: "reviewed from UI",
        policyProfile: "default",
        reviewerId: "ui",
      };
      if (runIdInput?.value) body.runId = runIdInput.value;
      try {
        await api("/memory/candidates/" + id + "/review", {
          method: "POST",
          body: JSON.stringify(body),
        });
        await loadMemory();
      } catch (err) {
        alert(err.message);
      }
    };
  });
}

document.getElementById("memory-form").onsubmit = async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const layer = fd.get("layer");
  const body = {
    layer,
    title: fd.get("title"),
    body: fd.get("body"),
    scopeRepo: "ash",
  };
  const runId = fd.get("runId");
  if (runId) body.runId = runId;
  const ref = fd.get("evidenceRef");
  if (layer !== "L0" && ref) {
    body.evidence = [{ kind: "file", ref }];
  }
  try {
    await api("/memory/candidates", { method: "POST", body: JSON.stringify(body) });
    e.target.reset();
    document.querySelector('#memory-form [name="layer"]').value = "L1";
    await loadMemory();
    if (runId && selectedRunId === runId) streamRun(runId);
  } catch (err) {
    alert(err.message);
  }
};

document.getElementById("btn-refresh-memory").onclick = () => loadMemory().catch(alert);

// --- Doctor ---
document.getElementById("btn-run-doctor").onclick = async () => {
  const el = document.getElementById("doctor-report");
  el.textContent = "Running TR0…";
  try {
    const { reportId } = await api("/doctor/run", {
      method: "POST",
      body: JSON.stringify({ suite: "TR0", format: "json" }),
    });
    const report = await api("/doctor/reports/" + reportId);
    el.textContent = JSON.stringify(report, null, 2);
  } catch (err) {
    el.textContent = "Error: " + err.message;
  }
};

// init
loadRuns().catch(console.error);
loadMemory().catch(console.error);
