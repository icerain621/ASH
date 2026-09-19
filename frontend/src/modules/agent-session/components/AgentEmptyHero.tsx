import {
  agentModeTagline,
  ASH_CHAT_BG_URL,
  ASH_ICON_URL,
  type AgentMode,
} from "../agentModeLabels";

type Props = {
  agentMode: AgentMode;
  compact?: boolean;
  onDismiss?: () => void;
};

/** Branded empty-state hero (prototype ash-hero). */
export function AgentEmptyHero({ agentMode, compact, onDismiss }: Props) {
  return (
    <div
      className={`agent-ash-hero${compact ? " compact" : ""}`}
      data-testid="agent-ash-hero"
      style={{
        backgroundImage: `linear-gradient(90deg, rgba(7,9,13,0.28) 0%, rgba(7,9,13,0.72) 48%, rgba(7,9,13,0.9) 100%), url(${ASH_CHAT_BG_URL})`,
      }}
    >
      <div className="agent-ash-hero-mark">
        <img src={ASH_ICON_URL} alt="ASH" width={72} height={72} />
      </div>
      <h2>ASH Agent</h2>
      <p data-testid="agent-ash-hero-tag">{agentModeTagline(agentMode)}</p>
      {onDismiss ? (
        <button
          type="button"
          className="btn mini ghost agent-ash-hero-dismiss"
          data-testid="agent-ash-hero-dismiss"
          onClick={onDismiss}
        >
          关闭空态
        </button>
      ) : null}
    </div>
  );
}
