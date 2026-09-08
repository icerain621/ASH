import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useQuestBoardStream } from "./questBoardStream";

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
    for (const listener of this.listeners.get(type) ?? []) {
      listener(ev);
    }
  }

  emitError() {
    this.onerror?.();
  }
}

describe("useQuestBoardStream", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    vi.stubGlobal("EventSource", MockEventSource as unknown as typeof EventSource);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("connects to quest stream and notifies on plan.created", () => {
    const onBoardEvent = vi.fn();
    const { result } = renderHook(() => useQuestBoardStream("local", { onBoardEvent }));
    expect(MockEventSource.instances[0].url).toBe("/api/v1/quest/stream");

    act(() => {
      MockEventSource.instances[0].emitOpen();
      MockEventSource.instances[0].emitMessage("plan.created", '{"planId":"gplan_1"}', "evt-1");
    });

    expect(result.current.status).toBe("open");
    expect(onBoardEvent).toHaveBeenCalledWith("plan.created");
  });

  it("falls back to board poll after reconnect exhaustion", async () => {
    const pollBoard = vi.fn(async () => ({}));
    const onBoardEvent = vi.fn();
    renderHook(() =>
      useQuestBoardStream("local", {
        onBoardEvent,
        pollBoard,
        maxReconnectAttempts: 2,
        pollIntervalMs: 1000,
      }),
    );

    act(() => {
      MockEventSource.instances[0].emitError();
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });
    act(() => {
      MockEventSource.instances[MockEventSource.instances.length - 1].emitError();
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(pollBoard).toHaveBeenCalled();
    expect(onBoardEvent).toHaveBeenCalledWith("poll.refresh");
  });
});
