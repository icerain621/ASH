import { createRootRoute, createRoute, createRouter, Navigate } from "@tanstack/react-router";
import { AppLayout } from "./layout/AppLayout";
import { AutomationPage } from "../pages/AutomationPage";
import { CIPage } from "../pages/CIPage";
import { DoctorPage } from "../pages/DoctorPage";
import { FeedbackPage } from "../pages/FeedbackPage";
import { MemoryPage } from "../pages/MemoryPage";
import { MetricsPage } from "../pages/MetricsPage";
import { ObservabilityPage } from "../pages/ObservabilityPage";
import { ReleasesPage } from "../pages/ReleasesPage";
import { RunsPage } from "../pages/RunsPage";
import { LoginPage } from "../pages/LoginPage";
import { CompliancePage } from "../pages/CompliancePage";
import { ScalePage } from "../pages/ScalePage";
import { SpacePage } from "../pages/SpacePage";

const rootRoute = createRootRoute({
  component: AppLayout,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: () => <Navigate to="/runs" replace />,
});

const runsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/runs",
  component: RunsPage,
});

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/login",
  component: LoginPage,
});

const memoryRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/memory",
  component: MemoryPage,
});

const automationRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/automation",
  component: AutomationPage,
});

const feedbackRoute = createRoute({
	getParentRoute: () => rootRoute,
	path: "/feedback",
	component: FeedbackPage,
});

const ciRoute = createRoute({
	getParentRoute: () => rootRoute,
	path: "/ci",
	component: CIPage,
});

const metricsRoute = createRoute({
	getParentRoute: () => rootRoute,
	path: "/metrics",
	component: MetricsPage,
});

const observabilityRoute = createRoute({
	getParentRoute: () => rootRoute,
	path: "/observability",
	component: ObservabilityPage,
});

const releasesRoute = createRoute({
	getParentRoute: () => rootRoute,
	path: "/releases",
	component: ReleasesPage,
});

const spaceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/space",
  component: SpacePage,
});

const doctorRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/doctor",
  component: DoctorPage,
});

const complianceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/compliance",
  component: CompliancePage,
});

const scaleRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/scale",
  component: ScalePage,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  runsRoute,
  memoryRoute,
	automationRoute,
	feedbackRoute,
	ciRoute,
	metricsRoute,
	observabilityRoute,
	releasesRoute,
	spaceRoute,
  complianceRoute,
  scaleRoute,
  doctorRoute,
]);

export const router = createRouter({
  routeTree,
  basepath: "/ui",
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
