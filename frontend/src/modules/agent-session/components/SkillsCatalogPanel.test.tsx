import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SkillsCatalogPanel } from "./SkillsCatalogPanel";
import {
  installSkillFromCatalog,
  installSkillPack,
  listSkillCatalog,
  listSkills,
  verifySkillPack,
} from "@/modules/skills/api/skills.api";

vi.mock("@/modules/skills/api/skills.api", () => ({
  listSkills: vi.fn(async () => ({
    items: [
      {
        id: "ash-test",
        name: "ash-test",
        description: "test skill",
        path: "x",
        relPath: "x",
        contextRef: "",
      },
    ],
    repoRoot: ".",
  })),
  listSkillCatalog: vi.fn(async () => ({
    ok: true,
    source: ".ash/skill-catalog.json",
    items: [
      {
        name: "pack-demo",
        version: "1.0.0",
        publisher: "ash",
        url: "file://pack-demo",
      },
    ],
  })),
  verifySkillPack: vi.fn(async () => ({
    ok: true,
    name: "pack-demo",
    version: "1.0.0",
    digest: "sha256:abc",
  })),
  installSkillPack: vi.fn(async () => ({
    ok: true,
    name: "pack-demo",
    version: "1.0.0",
    publisher: "ash",
    path: ".ash/skills/pack-demo",
    repoRoot: ".",
  })),
  installSkillFromCatalog: vi.fn(async () => ({
    ok: true,
    name: "pack-demo",
    version: "1.0.0",
    publisher: "ash",
    path: ".ash/skills/pack-demo",
    repoRoot: ".",
  })),
  getSkill: vi.fn(),
}));

function renderPanel(
  props: {
    onRunSkill?: (skillSlash: string) => void;
    canRun?: boolean;
  } = {},
) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <SkillsCatalogPanel open onClose={() => undefined} {...props} />
    </QueryClientProvider>,
  );
}

describe("SkillsCatalogPanel", () => {
  beforeEach(() => {
    vi.mocked(listSkills).mockClear();
    vi.mocked(listSkillCatalog).mockClear();
    vi.mocked(verifySkillPack).mockClear();
    vi.mocked(installSkillPack).mockClear();
    vi.mocked(installSkillFromCatalog).mockClear();
  });

  it("lists installed skills and catalog", async () => {
    renderPanel({ canRun: true, onRunSkill: vi.fn() });
    expect(await screen.findByTestId("agent-skills-panel")).toBeTruthy();
    expect(await screen.findByTestId("agent-skills-row-ash-test")).toBeTruthy();
    expect(screen.getByTestId("agent-skills-catalog-row-pack-demo")).toBeTruthy();
    expect(screen.getByTestId("catalog-private-marker")).toHaveTextContent("私有 · 不计费");
    expect(screen.getByTestId("agent-skills-pack")).toBeTruthy();
  });

  it("verifies and installs signed pack", async () => {
    renderPanel();
    fireEvent.change(await screen.findByTestId("agent-skills-pack-path"), {
      target: { value: "/tmp/demo.ash-skill.zip" },
    });
    fireEvent.change(screen.getByTestId("agent-skills-pack-sig"), {
      target: { value: "deadbeef" },
    });
    fireEvent.click(screen.getByTestId("agent-skills-pack-verify"));
    await waitFor(() => {
      expect(verifySkillPack).toHaveBeenCalledWith(
        expect.objectContaining({
          packPath: "/tmp/demo.ash-skill.zip",
          signature: "deadbeef",
        }),
      );
    });
    expect(await screen.findByTestId("agent-skills-message")).toHaveTextContent(/验签通过/);

    fireEvent.click(screen.getByTestId("agent-skills-pack-install"));
    await waitFor(() => {
      expect(installSkillPack).toHaveBeenCalled();
    });
    expect(await screen.findByTestId("agent-skills-message")).toHaveTextContent(/已安装/);
  });

  it("installs from org catalog", async () => {
    renderPanel();
    fireEvent.click(await screen.findByTestId("agent-skills-catalog-install-pack-demo"));
    await waitFor(() => {
      expect(installSkillFromCatalog).toHaveBeenCalledWith(
        expect.objectContaining({ name: "pack-demo", version: "1.0.0" }),
      );
    });
  });

  it("runs skill when session active", async () => {
    const onRunSkill = vi.fn();
    renderPanel({ canRun: true, onRunSkill });
    fireEvent.click(await screen.findByTestId("agent-skills-run-ash-test"));
    expect(onRunSkill).toHaveBeenCalledWith("/ash-test");
  });
});
