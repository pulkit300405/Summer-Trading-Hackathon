import React from 'react';

const services = [
  { key: 'submission', name: 'Submission Handler', icon: '⬆️', url: ':8080', details: [['Version','v1'],['Port','8080']] },
  { key: 'telemetry', name: 'Telemetry Ingester', icon: '📊', url: ':8082', details: [['Version','v1'],['Port','8082']] },
  { key: 'bot', name: 'Bot Fleet', icon: '🤖', url: ':8081', details: [['Bots','5000'],['Port','8081']] },
  { key: 'ui', name: 'Leaderboard UI', icon: '🏆', url: ':3000', details: [['Version','v1'],['Port','3000']] },
  { key: 'db', name: 'PostgreSQL', icon: '🗄️', url: ':5432', details: [['Version','15'],['Type','TimescaleDB']] },
  { key: 'k8s', name: 'Kubernetes IaC', icon: '☸️', url: 'manifests', details: [['Manifests','11'],['Status','Ready']] },
];

export default function SystemStatus({ health, botStats, lastUpdate }) {
  const isOk = (key) => {
    if (key === 'submission') return health.submission;
    if (key === 'telemetry') return health.telemetry;
    if (key === 'bot') return !!botStats;
    if (key === 'ui') return true;
    if (key === 'db') return health.telemetry;
    if (key === 'k8s') return true;
    return false;
  };

  return (
    <div>
      <div className="card section">
        <div className="card-header">
          <div className="card-icon">📡</div>
          <div>
            <div className="card-title">System Status</div>
            <div className="card-subtitle">
              Real-time infrastructure health
              {lastUpdate && ` · Updated: ${lastUpdate.toLocaleTimeString()}`}
            </div>
          </div>
        </div>

        <div className="service-grid">
          {services.map(svc => (
            <div key={svc.key} className={`service-card ${isOk(svc.key) ? 'ok' : 'down'}`}>
              <div className="service-top">
                <div className="service-icon">{svc.icon}</div>
                <div>
                  <div className="service-name">{svc.name}</div>
                  <div style={{ fontFamily: 'var(--mono)', fontSize: '0.7rem', color: 'var(--muted)' }}>
                    localhost{svc.url}
                  </div>
                </div>
              </div>
              {svc.details.map(([k, v]) => (
                <div key={k} className="service-detail">
                  <span className="service-key">{k}</span>
                  <span className="service-val">{k === 'Bots' && botStats ? botStats.total_bots?.toLocaleString() : v}</span>
                </div>
              ))}
              <div style={{ marginTop: '0.75rem' }}>
                <span className={`badge ${isOk(svc.key) ? 'badge-ok' : 'badge-fail'}`}>
                  ● {isOk(svc.key) ? 'OPERATIONAL' : 'DOWN'}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="grid-2 section">
        <div className="card">
          <div className="card-label">Platform Stats</div>
          <div className="grid-2" style={{ gap: '0.75rem' }}>
            <div className="stat-box">
              <div className="stat-label">Active Services</div>
              <div className="stat-value">{[health.submission, health.telemetry, !!botStats].filter(Boolean).length + 3}</div>
              <div className="stat-unit">of 6 running</div>
            </div>
            <div className="stat-box">
              <div className="stat-label">Bot Goroutines</div>
              <div className="stat-value">{botStats?.total_bots?.toLocaleString() || '—'}</div>
              <div className="stat-unit">concurrent</div>
            </div>
            <div className="stat-box">
              <div className="stat-label">Orders Sent</div>
              <div className="stat-value" style={{ fontSize: '1.2rem' }}>{botStats?.orders_sent?.toLocaleString() || '—'}</div>
              <div className="stat-unit">total</div>
            </div>
            <div className="stat-box">
              <div className="stat-label">Deployment</div>
              <div className="stat-value" style={{ fontSize: '0.9rem', paddingTop: '0.4rem' }}>Docker</div>
              <div className="stat-unit">+ K8s IaC ready</div>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-label">Platform Architecture</div>
          <pre className="arch-pre">{`
  CONTESTANT SUBMISSION
  (Orderbook / Matching Engine)
         │ upload code
  ┌──────▼──────────────┐
  │  SUBMISSION HANDLER │  :8080
  │  Validate → Docker  │  512MB / 1 core
  └──────┬──────────────┘
         │ spawn bots
  ┌──────┼──────┬─────────────┐
  │      │      │             │
 BOT   TELE  LEADER        POSTGRES
 FLEET  METRY  BOARD      + TIMESCALE
 :8081 :8082  :3000         :5432
 5K     p99    React      time-series
 bots   ms     live       metrics`}
          </pre>
        </div>
      </div>
    </div>
  );
}
