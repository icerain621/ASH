import { useMutation } from "@tanstack/react-query";
import { MessageSquarePlus, Send } from "lucide-react";
import { createFeedback } from "@/modules/platform/api/platform.api";

export function FeedbackPage() {
  const feedbackMut = useMutation({ mutationFn: createFeedback });

  function submit(formData: FormData) {
    feedbackMut.mutate({
      targetType: String(formData.get("targetType") || "run"),
      targetId: String(formData.get("targetId") || ""),
      rating: Number(formData.get("rating") || 0),
      comment: String(formData.get("comment") || ""),
    });
  }

  return (
    <section className="panel active">
      <div className="page-kicker">
        <MessageSquarePlus size={17} strokeWidth={1.8} />
        Feedback
      </div>
      <div className="page-heading">
        <div>
          <h1>反馈</h1>
          <p>记录运行、产物或记忆命中的采纳和质量反馈。</p>
        </div>
      </div>
      <div className="split">
        <div className="pane">
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              submit(new FormData(event.currentTarget));
            }}
          >
            <label>
              Target Type
              <select name="targetType" defaultValue="run">
                <option value="run">Run</option>
                <option value="artifact">Artifact</option>
                <option value="memory_hit">Memory Hit</option>
              </select>
            </label>
            <label>
              Target ID
              <input name="targetId" placeholder="run_..." required />
            </label>
            <label>
              Rating
              <input name="rating" type="number" min="-1" max="5" defaultValue="1" />
            </label>
            <label>
              Comment
              <textarea name="comment" rows={4} placeholder="记录采纳、失败原因或改进建议。" />
            </label>
            <button className="btn primary icon-btn" type="submit" disabled={feedbackMut.isPending}>
              <Send size={16} strokeWidth={1.8} />
              提交反馈
            </button>
          </form>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>Result</h2>
            <span>{feedbackMut.isSuccess ? "created" : "idle"}</span>
          </div>
          <pre className="code-block">
            {feedbackMut.data
              ? JSON.stringify(feedbackMut.data, null, 2)
              : feedbackMut.error
                ? feedbackMut.error.message
                : "提交后会显示反馈记录。"}
          </pre>
        </div>
      </div>
    </section>
  );
}
