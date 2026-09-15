import { useCallback, useEffect, useState, type CSSProperties } from "react";

const SIDEBAR_KEY = "ash.agentChat.sidebarWidth";
const DETAILS_KEY = "ash.agentChat.detailsWidth";

const SIDEBAR_DEFAULT = 280;
const DETAILS_DEFAULT = 300;
const SIDEBAR_MIN = 200;
const SIDEBAR_MAX = 420;
const DETAILS_MIN = 220;
const DETAILS_MAX = 440;

function clamp(n: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, n));
}

function readStored(key: string, fallback: number): number {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return fallback;
    const n = Number(raw);
    return Number.isFinite(n) ? n : fallback;
  } catch {
    return fallback;
  }
}

function writeStored(key: string, value: number) {
  try {
    localStorage.setItem(key, String(Math.round(value)));
  } catch {
    /* ignore quota / private mode */
  }
}

export type ChatColumnWidths = {
  sidebarWidth: number;
  detailsWidth: number;
  shellStyle: CSSProperties;
  startResize: (side: "sidebar" | "details", clientX: number) => void;
};

/** Persistable left/right column widths for AgentChatShell (DSH-like drag). */
export function useChatColumnWidths(): ChatColumnWidths {
  const [sidebarWidth, setSidebarWidth] = useState(() =>
    clamp(readStored(SIDEBAR_KEY, SIDEBAR_DEFAULT), SIDEBAR_MIN, SIDEBAR_MAX),
  );
  const [detailsWidth, setDetailsWidth] = useState(() =>
    clamp(readStored(DETAILS_KEY, DETAILS_DEFAULT), DETAILS_MIN, DETAILS_MAX),
  );

  useEffect(() => writeStored(SIDEBAR_KEY, sidebarWidth), [sidebarWidth]);
  useEffect(() => writeStored(DETAILS_KEY, detailsWidth), [detailsWidth]);

  const startResize = useCallback((side: "sidebar" | "details", clientX: number) => {
    const startX = clientX;
    const startSidebar = sidebarWidth;
    const startDetails = detailsWidth;

    const onMove = (ev: MouseEvent) => {
      const dx = ev.clientX - startX;
      if (side === "sidebar") {
        setSidebarWidth(clamp(startSidebar + dx, SIDEBAR_MIN, SIDEBAR_MAX));
      } else {
        setDetailsWidth(clamp(startDetails - dx, DETAILS_MIN, DETAILS_MAX));
      }
    };
    const onUp = () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }, [sidebarWidth, detailsWidth]);

  return {
    sidebarWidth,
    detailsWidth,
    shellStyle: {
      ["--ash-sidebar-width" as string]: `${sidebarWidth}px`,
      ["--ash-details-width" as string]: `${detailsWidth}px`,
    },
    startResize,
  };
}
