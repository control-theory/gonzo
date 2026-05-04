import { upsellURL } from '../../lib/constants'

export function MobiusChat() {
  return (
    <div className="chat-widget" style={{ flex: 1, minHeight: 0 }}>
      <div className="chat-header">
        <img src="/Mobius_Circle.svg" alt="Möbius" style={{ width: 32, height: 32 }} />
        <div>
          <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)' }}>
            Möbius
          </div>
          <div className="chat-status">
            <div className="chat-status-dot" />
            <span>Online</span>
          </div>
        </div>
      </div>

      <div className="chat-messages">
        <div className="chat-message assistant">
          <img src="/Mobius_Circle.svg" alt="Möbius" className="chat-message-avatar" style={{ width: 24, height: 24 }} />
          <div className="chat-message-content">
            <div className="chat-message-bubble">
              Welcome to <strong>Dstl8 Lite</strong>! I&apos;m Möbius, your AI log analyst.
              I can help you understand patterns, anomalies, and issues in your log data.
            </div>
            <div className="chat-message-time">Just now</div>
          </div>
        </div>

        <div className="chat-message assistant">
          <img src="/Mobius_Circle.svg" alt="Möbius" className="chat-message-avatar" style={{ width: 24, height: 24 }} />
          <div className="chat-message-content">
            <div className="chat-message-bubble">
              For continuous AI analysis and real-time insights,{' '}
              <a
                href={upsellURL('mobius-chat')}
                target="_blank"
                rel="noopener noreferrer"
                style={{ color: 'var(--accent-blue)', fontWeight: 500 }}
              >
                upgrade to Dstl8 Pro
              </a>{' '}
              to unlock the full Möbius experience.
            </div>
            <div className="chat-message-time">Just now</div>
          </div>
        </div>
      </div>

      <div className="chat-input-container">
        <div className="chat-input-wrapper">
          <input
            className="chat-input"
            placeholder="Ask about your logs..."
            disabled
            style={{ cursor: 'not-allowed', opacity: 0.6 }}
          />
          <button className="chat-send-btn" disabled>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="m22 2-7 20-4-9-9-4Z"/><path d="m22 2-11 11"/>
            </svg>
          </button>
        </div>
      </div>
    </div>
  )
}
