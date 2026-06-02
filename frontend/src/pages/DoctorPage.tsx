import { useMutation } from "@tanstack/react-query";
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
    onError: (err) => setReport("Error: " + (err as Error).message),
  });

  return (
    <section className="panel active">
      <div className="toolbar">
        <button className="btn primary" onClick={() => doctorMut.mutate()} disabled={doctorMut.isPending}>
          {doctorMut.isPending ? "Running TR0…" : "运行 TR0"}
        </button>
      </div>
      <pre className="code-block tall">{report}</pre>
    </section>
  );
}
