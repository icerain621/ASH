import { useMutation } from "@tanstack/react-query";
import { HeartPulse, Play } from "lucide-react";
import { useState } from "react";
import { getDoctorReport, runDoctor } from "@/modules/doctor/api/doctor.api";

export function DoctorPage() {
  const [report, setReport] = useState<string>("点击「运行 TR0」");

  const doctorMut = useMutation({
    mutationFn: async () => {
      const { reportId } = await runDoctor("TR0");
      return getDoctorReport(reportId);
    },
    onSuccess: (data) => setReport(JSON.stringify(data, null, 2)),
    onError: (err) => setReport("错误：" + (err as Error).message),
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
          <p>运行 TR 检查，并查看结构化诊断报告。</p>
        </div>
        <div className="toolbar">
          <button className="btn primary icon-btn" onClick={() => doctorMut.mutate()} disabled={doctorMut.isPending}>
            <Play size={16} strokeWidth={1.8} />
            {doctorMut.isPending ? "正在运行 TR0…" : "运行 TR0"}
          </button>
        </div>
      </div>
      <div className="pane">
        <div className="pane-title">
          <h2>报告</h2>
          <span>{doctorMut.isPending ? "运行中" : "就绪"}</span>
        </div>
        <pre className="code-block tall">{report}</pre>
      </div>
    </section>
  );
}
