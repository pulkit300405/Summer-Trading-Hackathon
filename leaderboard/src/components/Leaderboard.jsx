import React, { useState } from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

const fmt = (v, d = 1) => v == null ? '—' : typeof v === 'number' ? v.toFixed(d) : v;
const latClass = v => v < 20 ? 'good' : v < 50 ? 'warn' : 'bad';
const tpsClass = v => v > 5000 ? 'good' : v > 1000 ? 'warn' : 'bad';
const MEDALS = ['🥇', '🥈', '🥉'];

const CustomTooltip = ({ active, payload, label }) => {
  if (!active || !payload?.length) return null;
  return (
    <div style={{ background: '#0d1520', border: '1px solid #1e3048', borderRadius: 6, padding: '0.75rem', fontFamily: 'JetBrains Mono', fontSize: '0.75rem' }}>
      <div style={{ color: '#4a6480', marginBottom: 4 }}>{label}</div>
      {payload.map(p => <div key={p.name} style={{ color: p.color }}>{p.name}: {typeof p.value === 'number' ? p.value.toFixed(2) : p.value}</div>)}
    </div>
  );
};

export default function Leaderboard({ data }) {
  const [view, setView] = useState('table');

  if (!data || data.length === 0) {
    return (
      <div className="empty">
        <div className="empty-icon">📡</div>
        <h3>Waiting for submissions</h3>
        <p>Submit your trading engine to appear here</p>
      </div>
    );
  }

  const sorted = [...data].sort((a, b) => (b.composite_score || 0) - (a.composite_score || 0));

  const latencyData = sorted.map(e => ({
    name: e.team_name || 'Unknown',
    p50: parseFloat((e.p50_ms || e.p50 || 0).toFixed(2)),
    p90: parseFloat((e.p90_ms || e.p90 || 0).toFixed(2)),
    p99: parseFloat((e.p99_ms || e.p99 || 0).toFixed(2)),
  }));

  return (
    <div>
      {/* Top 3 Hero Cards */}
      <div className="grid-3 section">
        {sorted.slice(0, 3).map((entry, i) => (
          <div key={entry.submission_id} className="card" style={{
            borderColor: i === 0 ? '#ffd700' : i === 1 ? '#c0c0c0' : '#cd7f32',
            borderWidth: i === 0 ? '2px' : '1px',
          }}>
            <div style={{ fontSize: '1.5rem', marginBottom: '0.5rem' }}>{MEDALS[i]}</div>
            <div className="team-name-cell" style={{ fontSize: '1.05rem', marginBottom: '0.75rem' }}>{entry.team_name || 'Unknown'}</div>
            <div className="grid-2" style={{ gap: '0.6rem' }}>
              {[
                ['Score', fmt(entry.composite_score, 1), 'var(--accent)'],
                ['p99', `${fmt(entry.p99_ms || entry.p99, 1)}ms`, 'var(--green)'],
                ['TPS', fmt(entry.tps || entry.throughput_tps, 0), 'var(--cyan)'],
                ['Correct', `${fmt(entry.correctness_rate, 1)}%`, 'var(--green)'],
              ].map(([label, val, color]) => (
                <div key={label} className="stat-box" style={{ padding: '0.6rem 0.75rem' }}>
                  <div className="stat-label">{label}</div>
                  <div className="stat-value" style={{ fontSize: '1.1rem', color }}>{val}</div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      {/* View Toggle */}
      <div style={{ display: 'flex', gap: '0.5rem', marginBottom: '1rem' }}>
        {['table', 'charts'].map(v => (
          <button key={v} className={`btn ${view === v ? 'btn-primary' : 'btn-secondary'}`} onClick={() => setView(v)}>
            {v === 'table' ? '📋 Table' : '📊 Charts'}
          </button>
        ))}
      </div>

      {view === 'table' ? (
        <div className="table-wrap">
          <table className="lb-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Team</th>
                <th>Score</th>
                <th>Correct</th>
                <th>TPS</th>
                <th>p99 ms</th>
                <th>p50 ms</th>
                <th>Orders</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((entry, i) => (
                <tr key={entry.submission_id}>
                  <td><span className={`rank ${i < 3 ? `rank-${i+1}` : 'rank-other'}`}>{i < 3 ? MEDALS[i] : i+1}</span></td>
                  <td className="team-name-cell">{entry.team_name || 'Unknown'}</td>
                  <td><span className="score-val">{fmt(entry.composite_score, 1)}</span></td>
                  <td><span className={(entry.correctness_rate||0) > 99 ? 'good' : 'warn'}>{fmt(entry.correctness_rate, 1)}%</span></td>
                  <td><span className={tpsClass(entry.tps||entry.throughput_tps||0)}>{fmt(entry.tps||entry.throughput_tps, 0)}</span></td>
                  <td><span className={latClass(entry.p99_ms||entry.p99||0)}>{fmt(entry.p99_ms||entry.p99, 1)}ms</span></td>
                  <td><span className={latClass(entry.p50_ms||entry.p50||0)}>{fmt(entry.p50_ms||entry.p50, 1)}ms</span></td>
                  <td style={{ color: 'var(--muted)' }}>{fmt(entry.total_orders||entry.orders_processed, 0)}</td>
                  <td><span className="badge badge-completed">● completed</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div>
          <div className="card section">
            <div className="card-label">Latency Breakdown — p50 / p90 / p99 (ms)</div>
            <ResponsiveContainer width="100%" height={260}>
              <BarChart data={latencyData} margin={{ top: 10, right: 20, left: 0, bottom: 20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1e3048" />
                <XAxis dataKey="name" tick={{ fill: '#4a6480', fontSize: 11, fontFamily: 'JetBrains Mono' }} />
                <YAxis tick={{ fill: '#4a6480', fontSize: 11, fontFamily: 'JetBrains Mono' }} unit="ms" />
                <Tooltip content={<CustomTooltip />} />
                <Legend wrapperStyle={{ fontSize: 11, fontFamily: 'JetBrains Mono', color: '#4a6480' }} />
                <Bar dataKey="p50" fill="#06b6d4" radius={[3,3,0,0]} />
                <Bar dataKey="p90" fill="#7c3aed" radius={[3,3,0,0]} />
                <Bar dataKey="p99" fill="#ef4444" radius={[3,3,0,0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
          <div className="grid-2">
            <div className="card">
              <div className="card-label">Throughput (TPS)</div>
              <ResponsiveContainer width="100%" height={200}>
                <BarChart data={sorted.map(e => ({ name: e.team_name, TPS: e.tps||e.throughput_tps||0 }))} margin={{ top: 10, right: 10, left: 0, bottom: 20 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1e3048" />
                  <XAxis dataKey="name" tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} />
                  <YAxis tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} />
                  <Tooltip content={<CustomTooltip />} />
                  <Bar dataKey="TPS" fill="#22c55e" radius={[3,3,0,0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
            <div className="card">
              <div className="card-label">Composite Score</div>
              <ResponsiveContainer width="100%" height={200}>
                <BarChart data={sorted.map(e => ({ name: e.team_name, Score: e.composite_score||0 }))} margin={{ top: 10, right: 10, left: 0, bottom: 20 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1e3048" />
                  <XAxis dataKey="name" tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} />
                  <YAxis tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} />
                  <Tooltip content={<CustomTooltip />} />
                  <Bar dataKey="Score" fill="#4f8ef7" radius={[3,3,0,0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
