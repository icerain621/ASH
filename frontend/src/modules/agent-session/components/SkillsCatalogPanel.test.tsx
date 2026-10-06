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
      expect(screen.getByTestId("catalog-private-marker")).toHaveTextContent("私有 · 组织 Hub · 不计费");
    expect(screen.getByTestId("agent-skills-pack")).toBeTruthy();
  });

  it("verifies and installs signed pack", async () => {
    renderPanel();
    const verifyBtn = await screen.findByTestId("agent-skills-pack-verify");
    const installBtn = screen.getByTestId("agent-skills-pack-install");
    expect(verifyBtn).toBeDisabled();
    expect(verifyBtn).toHaveAttribute("title", "需要填写 packPath");
    expect(installBtn).toBeDisabled();
    expect(installBtn).toHaveAttribute("title", "需要填写 packPath");

    fireEvent.change(screen.getByTestId("agent-skills-pack-path"), {
      target: { value: "/tmp/demo.ash-skill.zip" },
    });
    expect(verifyBtn).toHaveAttribute("title", "需要填写 signature");

    fireEvent.change(screen.getByTestId("agent-skills-pack-sig"), {
      target: { value: "deadbeef" },
    });
    fireEvent.click(verifyBtn);
    await waitFor(() => {
      expect(verifySkillPack).toHaveBeenCalledWith(
        expect.objectContaining({
          packPath: "/tmp/demo.ash-skill.zip",
          signature: "deadbeef",
        }),
      );
    });
    expect(await screen.findByTestId("agent-skills-message")).toHaveTextContent(/验签通过/);

    expect(installBtn).toHaveAttribute("title", "安装签名 pack（需确认）");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    fireEvent.click(installBtn);
    expect(installSkillPack).not.toHaveBeenCalled();
    confirmSpy.mockReturnValue(true);
    fireEvent.click(installBtn);
    await waitFor(() => {
      expect(installSkillPack).toHaveBeenCalled();
    });
    expect(await screen.findByTestId("agent-skills-message")).toHaveTextContent(/已安装/);
    confirmSpy.mockRestore();
  });

  it("installs from org catalog", async () => {
    renderPanel();
    const install = await screen.findByTestId("agent-skills-catalog-install-pack-demo");
    expect(install).toHaveAttribute("title", "从 catalog 安装「pack-demo」（需确认）");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    fireEvent.click(install);
    expect(installSkillFromCatalog).not.toHaveBeenCalled();
    confirmSpy.mockReturnValue(true);
    fireEvent.click(install);
    await waitFor(() => {
      expect(installSkillFromCatalog).toHaveBeenCalledWith(
        expect.objectContaining({ name: "pack-demo", version: "1.0.0" }),
      );
    });
    confirmSpy.mockRestore();
  });

  it("runs skill when session active", async () => {
    const onRunSkill = vi.fn();
    renderPanel({ canRun: true, onRunSkill });
    fireEvent.click(await screen.findByTestId("agent-skills-run-ash-test"));
    expect(onRunSkill).toHaveBeenCalledWith("/ash-test");
  });
});
