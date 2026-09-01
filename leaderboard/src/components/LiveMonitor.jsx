import React, { useState, useEffect, useRef } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

const CustomTooltip = ({ active, payload, label }) => {
  if (!active || !payload?.length) return null;
  return (
    <div style={{ background: '#0d1520', border: '1px solid #1e3048', borderRadius: 6, padding: '0.75rem', fontFamily: 'JetBrains Mono, monospace', fontSize: '0.75rem' }}>
      <div style={{ color: '#4a6480', marginBottom: 4 }}>{label}s</div>
      {payload.map(p => (
        <div key={p.name} style={{ color: p.color }}>{p.name}: {typeof p.value === 'number' ? p.value.toFixed(2) : p.value}</div>
      ))}
    </div>
  );
};

export default function LiveMonitor({ data, botStats }) {
  const [history, setHistory] = useState([]);
  const [teamId, setTeamId] = useState('');
  const [duration, setDuration] = useState(30);
  const [concurrency, setConcurrency] = useState(100);
  const [running, setRunning] = useState(false);
  const startRef = useRef(Date.now());

  useEffect(() => {
    if (!botStats) return;
    setHistory(prev => [...prev, {
      time: Math.floor((Date.now() - startRef.current) / 1000),
      orders: botStats.orders_sent || 0,
      bots: botStats.total_bots || 0,
    }].slice(-60));
  }, [botStats]);

  const sorted = [...(data || [])].sort((a, b) => (b.composite_score || 0) - (a.composite_score || 0));

  const colors = ['#22c55e', '#f59e0b', '#06b6d4', '#7c3aed', '#ef4444', '#4f8ef7'];

  return (
    <div>
      <div className="monitor-grid">
        {/* Left Panel */}
        <div>
          <div className="card section">
            <div className="card-header">
              <div className="card-icon">🚀</div>
              <div>
                <div className="card-title">Deployment Control</div>
                <div className="card-subtitle">Configure load test</div>
              </div>
            </div>

            <div className="deploy-input">
              <div className="form-group">
                <label className="form-label">DEPLOY_ID</label>
                <input
                  className="form-input"
                  placeholder="Team name..."
                  value={teamId}
                  onChange={e => setTeamId(e.target.value)}
                />
              </div>

              <div style={{ background: 'var(--surface2)', border: '1px solid var(--border)', borderRadius: 6, padding: '0.6rem 0.875rem', fontFamily: 'var(--mono)', fontSize: '0.78rem', color: 'var(--muted)' }}>
                📄 GO_SUBMISSION.ZIP
              </div>

              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', padding: '0.4rem 0', fontFamily: 'var(--mono)', fontSize: '0.75rem' }}>
                <span style={{ color: 'var(--yellow)' }}>STATE:</span>
                <span style={{ color: botStats ? 'var(--green)' : 'var(--muted)' }}>
                  {botStats ? '● BENCHMARKING' : '● STANDBY'}
                </span>
              </div>

              <button
                className="run-btn"
                onClick={() => setRunning(!running)}
                style={{ background: running ? 'var(--red)' : 'var(--green)' }}
              >
                {running ? '⏹ STOP TEST' : '▶ EXECUTE DEPLOYMENT'}
              </button>
            </div>
          </div>

          <div className="card section">
            <div className="card-label">Automated Load Test</div>
            <div className="form-group">
              <label className="form-label">TEAM_ID</label>
              <input
                className="form-input"
                placeholder="team_name..."
                value={teamId}
                onChange={e => setTeamId(e.target.value)}
              />
            </div>
            <div className="load-row">
              <div className="form-group">
                <label className="form-label">DURATION (s)</label>
                <input
                  className="form-input"
                  type="number"
                  value={duration}
                  onChange={e => setDuration(e.target.value)}
                />
              </div>
              <div className="form-group">
                <label className="form-label">CONCURRENCY</label>
                <input
                  className="form-input"
                  type="number"
                  value={concurrency}
                  onChange={e => setConcurrency(e.target.value)}
                />
              </div>
            </div>
            <button className="run-btn" onClick={() => setRunning(!running)}>
              ▶ RUN LOAD TEST
            </button>
          </div>
        </div>

        {/* Right Panel - Leaderboard */}
        <div className="card">
          <div className="card-header">
            <div className="card-icon">🏆</div>
            <div>
              <div className="card-title">Live Leaderboard</div>
              <div className="card-subtitle">KBS CLUSTER: {botStats ? '● ACTIVE' : '● STANDBY'}</div>
            </div>
          </div>

          {sorted.length === 0 ? (
            <div className="empty">
              <div className="empty-icon">📡</div>
              <h3>No submissions yet</h3>
              <p>Submit your trading engine to appear here</p>
            </div>
          ) : (
            <div className="table-wrap">
              <table className="lb-table">
                <thead>
                  <tr>
                    <th>RANK</th>
                    <th>TEAM</th>
                    <th>MAX TPS</th>
                    <th>P50</th>
                    <th>P90</th>
                    <th>P99</th>
                    <th>ACCURACY</th>
                    <th>SCORE</th>
                  </tr>
                </thead>
                <tbody>
                  {sorted.map((entry, i) => (
                    <tr key={entry.submission_id}>
                      <td><span style={{ fontFamily: 'var(--mono)', color: 'var(--muted)' }}>{String(i+1).padStart(2,'0')}</span></td>
                      <td className="team-name-cell">{entry.team_name || 'Unknown'}</td>
                      <td><span className="good">{(entry.tps || entry.throughput_tps || 0).toFixed(2)}</span></td>
                      <td><span style={{ color: (entry.p50_ms||entry.p50||0) < 50 ? 'var(--green)' : 'var(--yellow)' }}>{(entry.p50_ms||entry.p50||0).toFixed(0)}ms</span></td>
                      <td><span style={{ color: (entry.p90_ms||entry.p90||0) < 100 ? 'var(--green)' : 'var(--yellow)' }}>{(entry.p90_ms||entry.p90||0).toFixed(0)}ms</span></td>
                      <td><span style={{ color: (entry.p99_ms||entry.p99||0) < 200 ? 'var(--green)' : 'var(--red)' }}>{(entry.p99_ms||entry.p99||0).toFixed(2)}ms</span></td>
                      <td><span className="good">{(entry.correctness_rate||0).toFixed(1)}%</span></td>
                      <td><span className="score-val">{(entry.composite_score||0).toFixed(2)}</span></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Time-series chart */}
          <div style={{ marginTop: '1.5rem' }}>
            <div className="card-label">Active Bot Fleet Telemetry (LIVE)</div>
            <div className="telemetry-bar">
              <div className="tele-item">
                <div className="tele-label">Active Team</div>
                <div className="tele-value" style={{ fontSize: '0.85rem', color: 'var(--text)' }}>{teamId || 'N/A'}</div>
              </div>
              <div className="tele-item">
                <div className="tele-label">Requests</div>
                <div className="tele-value">{botStats?.orders_sent?.toLocaleString() || '—'}</div>
              </div>
              <div className="tele-item">
                <div className="tele-label">Errors</div>
                <div className="tele-value" style={{ color: 'var(--muted)' }}>—</div>
              </div>
              <div className="tele-item">
                <div className="tele-label">P50 Latency</div>
                <div className="tele-value">{sorted[0] ? `${(sorted[0].p50_ms||sorted[0].p50||0).toFixed(2)}ms` : '—'}</div>
              </div>
              <div className="tele-item">
                <div className="tele-label">P99 Latency</div>
                <div className="tele-value">{sorted[0] ? `${(sorted[0].p99_ms||sorted[0].p99||0).toFixed(2)}ms` : '—'}</div>
              </div>
              <div className="tele-item">
                <div className="tele-label">Concurrency</div>
                <div className="tele-value">{botStats?.total_bots || concurrency}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Time-series Line Charts */}
      <div className="grid-2 section" style={{ marginTop: '1.25rem' }}>
        <div className="card">
          <div className="card-label">Orders Over Time</div>
          <div className="chart-wrap">
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={history} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1e3048" />
                <XAxis dataKey="time" tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} tickFormatter={v => `${v}s`} />
                <YAxis tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} tickFormatter={v => v.toLocaleString()} />
                <Tooltip content={<CustomTooltip />} />
                <Line type="monotone" dataKey="orders" name="Orders" stroke="#22c55e" strokeWidth={2} dot={false} isAnimationActive={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="card">
          <div className="card-label">Composite Score — by team</div>
          <div className="chart-wrap">
            {sorted.length === 0 ? (
              <div className="empty" style={{ padding: '2rem' }}>
                <p>Submit engines to see scores</p>
              </div>
            ) : (
              <ResponsiveContainer width="100%" height={220}>
                <LineChart margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1e3048" />
                  <XAxis dataKey="time" tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} tickFormatter={v => `${v}s`} />
                  <YAxis tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} />
                  <Tooltip content={<CustomTooltip />} />
                  <Legend wrapperStyle={{ fontSize: 10, fontFamily: 'JetBrains Mono' }} />
                  {sorted.slice(0, 5).map((entry, i) => (
                    <Line key={entry.submission_id} type="monotone" data={[{ time: 0, score: 0 }, { time: 10, score: entry.composite_score || 0 }]} dataKey="score" name={entry.team_name} stroke={colors[i % colors.length]} strokeWidth={2} dot={false} isAnimationActive={false} />
                  ))}
                </LineChart>
              </ResponsiveContainer>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
