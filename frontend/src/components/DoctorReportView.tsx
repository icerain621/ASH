import type { DoctorReport } from "@/modules/doctor/api/doctor.api";

const CASE_LABELS: Record<string, string> = {
  "TR0-01": "交付闭环",
  "TR0-02": "事件流可观测",
  "TR0-03": "回放基础",
  "TR0-04": "Agent 任务",
  "TR0-05": "产物索引",
  "TR0-06": "证据绑定",
  "TR0-07": "Checkpoint 恢复",
  "TR0-08": "三场景模板",
  "TR1-01": "模型路由降级",
  "TR1-02": "瀑布质量指标",
  "TR1-03": "记忆冲突治理",
  "TR1-04": "MCP 隔离",
  "TR1-05": "DSL Schema 校验",
  "TR2-01": "身份与作用域",
  "TR2-02": "Run 空间隔离",
  "TR2-03": "产物存储配置",
  "TR2-04": "插件 ABI",
  "TR2-05": "Secret 泄漏扫描",
  "TR3-01": "记忆迁移兼容",
  "TR3-02": "灾备降级",
  "TR3-03": "成本/延迟 SLO",
  "TR3-04": "审计可追责",
  "TR3-05": "指标回放一致",
  "TR3-06": "Postgres RAG FTS",
  "TR3-07": "插件导出健康",
  "M2-01": "权限矩阵",
  "M2-02": "场景策略更新",
  "M2-03": "运行期策略拒绝",
  "M3-01": "多租户隔离",
  "M3-02": "Postgres 就绪",
  "M3-03": "迁移目录",
  "M3-04": "迁移校验",
  "M3-05": "ExecGo E2E",
  "M3-06": "Postgres RLS",
  "M3-07": "ash_app 连接",
  "M3-08": "SQL 修订版本",
  "M3-09": "运维快照契约",
};

function caseLabel(id: string) {
  return CASE_LABELS[id] || id;
}

export function DoctorReportView({ report }: { report: DoctorReport | null }) {
  if (!report) {
    return <p className="muted-line">尚无报告。</p>;
  }
  const total = report.summary.pass + report.summary.fail;
  return (
    <div className="doctor-report">
      <div className="doctor-summary">
        <span className={"status-pill " + (report.summary.fail === 0 ? "ok" : "err")}>
          <span className="status-dot" />
          {report.suite} · {report.summary.pass}/{total} 通过
        </span>
        <span className="muted-line">
          {new Date(report.startedAt).toLocaleString()} → {new Date(report.finishedAt).toLocaleString()}
        </span>
      </div>
      <table className="table doctor-case-table">
        <thead>
          <tr>
            <th>用例</th>
            <th>说明</th>
            <th>状态</th>
            <th>证据</th>
          </tr>
        </thead>
        <tbody>
          {report.results.map((item) => (
            <tr key={item.id}>
              <td>
                <code>{item.id}</code>
              </td>
              <td title={item.message}>{caseLabel(item.id)}</td>
              <td>
                <span className={"status-pill " + (item.status === "pass" ? "ok" : "err")}>
                  <span className="status-dot" />
                  {item.status === "pass" ? "通过" : "失败"}
                </span>
              </td>
              <td>
                <code className="evidence-snippet" title={item.message}>
                  {item.evidence?.length
                    ? item.evidence.map((ev) => `${ev.kind}:${ev.ref}`).join(", ")
                    : item.message || "-"}
                </code>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
