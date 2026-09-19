import {
  AGENT_MODE_OPTIONS,
  type AgentMode,
} from "../agentModeLabels";

type Props = {
  value: AgentMode;
  disabled?: boolean;
  onChange: (mode: AgentMode) => void;
};

/** Sidebar coding / general mode toggle (prototype-aligned). */
export function AgentModeSwitch({ value, disabled, onChange }: Props) {
  return (
    <div
      className="agent-mode-switch"
      data-testid="agent-mode-switch"
      role="group"
      aria-label="Agent 模式"
    >
      {AGENT_MODE_OPTIONS.map((opt) => (
        <button
          key={opt.value}
          type="button"
          className={value === opt.value ? "on" : undefined}
          data-agent-mode={opt.value}
          data-testid={`agent-mode-${opt.value}`}
          disabled={disabled}
          aria-pressed={value === opt.value}
          onClick={() => onChange(opt.value)}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}
