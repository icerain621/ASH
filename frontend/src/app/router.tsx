import { createRootRoute, createRoute, createRouter, Navigate } from "@tanstack/react-router";
import { AppLayout } from "./layout/AppLayout";
import { AutomationPage } from "../pages/AutomationPage";
import { DoctorPage } from "../pages/DoctorPage";
import { FeedbackPage } from "../pages/FeedbackPage";
import { MemoryPage } from "../pages/MemoryPage";
import { RunsPage } from "../pages/RunsPage";
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

const routeTree = rootRoute.addChildren([
  indexRoute,
  runsRoute,
  memoryRoute,
  automationRoute,
  feedbackRoute,
  spaceRoute,
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
