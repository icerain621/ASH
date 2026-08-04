import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { nextReconnectDelayMs, useRunStream } from "./runStream";

type Handler = ((ev: MessageEvent) => void) | null;

class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  onopen: (() => void) | null = null;
  onmessage: Handler = null;
  onerror: (() => void) | null = null;
  private listeners = new Map<string, Set<EventListener>>();
  readyState = 0;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: EventListener) {
    if (!this.listeners.has(type)) this.listeners.set(type, new Set());
    this.listeners.get(type)!.add(listener);
  }

  close() {
    this.readyState = 2;
  }

  emitOpen() {
    this.readyState = 1;
    this.onopen?.();
  }

  emitMessage(type: string, data: string, lastEventId = "") {
    const ev = { type, data, lastEventId } as MessageEvent;
    if (type === "message") {
      this.onmessage?.(ev);
      return;
    }
    for (const listener of this.listeners.get(type) ?? []) {
      listener(ev);
    }
  }

  emitError() {
    this.onerror?.();
  }
}

describe("nextReconnectDelayMs", () => {
  it("grows exponentially up to 30s", () => {
    expect(nextReconnectDelayMs(0)).toBe(1000);
    expect(nextReconnectDelayMs(1)).toBe(2000);
    expect(nextReconnectDelayMs(2)).toBe(4000);
    expect(nextReconnectDelayMs(10)).toBe(30000);
  });
});

describe("useRunStream", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    vi.stubGlobal("EventSource", MockEventSource as unknown as typeof EventSource);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("connects and appends typed events", () => {
    const { result } = renderHook(() => useRunStream("run_1"));
    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe("/api/v1/runs/run_1/stream");

    act(() => {
      MockEventSource.instances[0].emitOpen();
      MockEventSource.instances[0].emitMessage("run.started", '{"ok":true}', "evt-1");
    });

    expect(result.current.status).toBe("open");
    expect(result.current.lines).toHaveLength(1);
    expect(result.current.lines[0].type).toBe("run.started");
  });

  it("reconnects with Last-Event-ID after error", () => {
    const { result } = renderHook(() => useRunStream("run_1"));

    act(() => {
      MockEventSource.instances[0].emitOpen();
      MockEventSource.instances[0].emitMessage("step.finished", "{}", "evt-9");
      MockEventSource.instances[0].emitError();
    });

    expect(result.current.status).toBe("reconnecting");
    expect(result.current.lines.some((l) => l.type === "sse")).toBe(true);

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(MockEventSource.instances).toHaveLength(2);
    expect(MockEventSource.instances[1].url).toContain("Last-Event-ID=evt-9");

    act(() => {
      MockEventSource.instances[1].emitOpen();
    });
    expect(result.current.status).toBe("open");
  });

  it("stops reconnecting after unmount", () => {
    const { unmount } = renderHook(() => useRunStream("run_1"));
    act(() => {
      MockEventSource.instances[0].emitError();
    });
    unmount();
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(MockEventSource.instances).toHaveLength(1);
  });
});
