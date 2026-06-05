import { useMutation } from "@tanstack/react-query";
import { HeartPulse, Play } from "lucide-react";
import { useState } from "react";
import { DoctorReportView } from "@/components/DoctorReportView";
import { getDoctorReport, runDoctor, type DoctorReport, type DoctorSuite } from "@/modules/doctor/api/doctor.api";
import { Link } from "@tanstack/react-router";

const SUITES = ["TR0", "TR1", "TR2", "TR3", "M2", "M3", "ALL"] as const;

export function DoctorPage() {
  const [suite, setSuite] = useState<DoctorSuite>("TR0");
  const [report, setReport] = useState<DoctorReport | null>(null);
  const [errorText, setErrorText] = useState("");

  const doctorMut = useMutation({
    mutationFn: async (selectedSuite: DoctorSuite) => {
      const { reportId } = await runDoctor(selectedSuite);
      return getDoctorReport(reportId);
    },
    onSuccess: (data) => {
      setReport(data);
      setErrorText("");
    },
    onError: (err) => {
      setReport(null);
      setErrorText((err as Error).message);
    },
  });

  return (
    <section className="panel active">
      <div className="page-kicker">
        <HeartPulse size={17} strokeWidth={1.8} />
        健康诊断
      </div>
      <div className="page-heading">
        <div>
          <h1>诊断</h1>
          <p>运行 TR 检查套件，并以表格查看每项用例与证据。</p>
          {suite === "TR2" && (
            <p className="muted-line">
              TR2 侧重身份/隔离/存储/插件 ABI，也可在{" "}
              <Link to="/compliance" className="inline-link">
                合规控制台
              </Link>{" "}
              查看实时配置。
            </p>
          )}
          {suite === "TR3" && (
            <p className="muted-line">
              TR3 为 GA 规模化门禁，可在{" "}
              <Link to="/scale" className="inline-link">
                规模化控制台
              </Link>{" "}
              查看就绪指标。
            </p>
          )}
          {suite === "M2" && (
            <p className="muted-line">
              M2 覆盖 RBAC 与场景工具矩阵，可在{" "}
              <Link to="/space" className="inline-link">
                空间控制台
              </Link>{" "}
              查看并编辑策略。
            </p>
          )}
          {suite === "M3" && (
            <p className="muted-line">
              M3 校验多租户隔离、Postgres 就绪与 migrate 表目录（M3-03），详见{" "}
              <code>doc/05-M3-多租户与Postgres演进.md</code>。
            </p>
          )}
        </div>
        <div className="toolbar">
          <label className="scenario-picker">
            套件
            <select
              value={suite}
              disabled={doctorMut.isPending}
              onChange={(event) => setSuite(event.target.value as DoctorSuite)}
            >
              {SUITES.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
          </label>
          <button
            className="btn primary icon-btn"
            onClick={() => doctorMut.mutate(suite)}
            disabled={doctorMut.isPending}
          >
            <Play size={16} strokeWidth={1.8} />
            {doctorMut.isPending ? `正在运行 ${suite}…` : `运行 ${suite}`}
          </button>
        </div>
      </div>
      {errorText && <p className="error-text">{errorText}</p>}
      <div className="pane">
        <div className="pane-title">
          <h2>报告</h2>
          <span>{doctorMut.isPending ? "运行中" : report ? `${report.summary.pass} 通过` : "就绪"}</span>
        </div>
        <DoctorReportView report={report} />
        {report && (
          <details className="raw-report">
            <summary>原始 JSON</summary>
            <pre className="code-block tall">{JSON.stringify(report, null, 2)}</pre>
          </details>
        )}
      </div>
    </section>
  );
}
