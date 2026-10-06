import { Link } from "@tanstack/react-router";
import { Compass } from "lucide-react";

/** Shown when no console route matches (unknown /ui/... path). */
export function NotFoundPage() {
  return (
    <section className="panel active" data-testid="console-not-found">
      <div className="page-kicker">
        <Compass size={17} strokeWidth={1.8} />
        Not found
      </div>
      <div className="page-heading">
        <div>
          <h1>页面不存在</h1>
          <p>该路径没有对应的控制台页面。请从导航进入，或返回任务板。</p>
        </div>
        <div className="toolbar">
          <Link to="/quest" className="btn primary" data-testid="console-not-found-home">
            回任务板
          </Link>
          <Link to="/runs" className="btn" data-testid="console-not-found-runs">
            运行
          </Link>
        </div>
      </div>
    </section>
  );
}
