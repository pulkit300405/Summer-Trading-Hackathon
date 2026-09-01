import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

const CustomTooltip = ({ active, payload, label }) => {
  if (!active || !payload?.length) return null;
  return (
    <div style={{ background: '#0d1420', border: '1px solid #1e2d42', borderRadius: 6, padding: '0.75rem', fontFamily: 'JetBrains Mono, monospace', fontSize: '0.78rem' }}>
      <div style={{ color: '#4a6480', marginBottom: 4 }}>{label}</div>
      {payload.map(p => (
        <div key={p.name} style={{ color: p.color }}>{p.name}: {typeof p.value === 'number' ? p.value.toFixed(2) : p.value}</div>
      ))}
    </div>
  );
};

export default function MetricsDashboard({ data }) {
  if (!data || data.length === 0) {
    return (
      <div className="empty">
        <div className="empty-icon">📊</div>
        <h3>No metrics yet</h3>
        <p>Submit your trading engine to see metrics</p>
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

  const tpsData = sorted.map(e => ({
    name: e.team_name || 'Unknown',
    TPS: parseFloat((e.tps || e.throughput_tps || 0).toFixed(0)),
  }));

  const scoreData = sorted.map(e => ({
    name: e.team_name || 'Unknown',
    Score: parseFloat((e.composite_score || 0).toFixed(2)),
  }));

  return (
    <div>
      <div className="card section">
        <div className="card-title">Latency Breakdown — p50 / p90 / p99 (ms)</div>
        <div className="chart-wrap">
          <ResponsiveContainer width="100%" height={280}>
            <BarChart data={latencyData} margin={{ top: 10, right: 20, left: 0, bottom: 20 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#1e2d42" />
              <XAxis dataKey="name" tick={{ fill: '#4a6480', fontSize: 11, fontFamily: 'JetBrains Mono' }} />
              <YAxis tick={{ fill: '#4a6480', fontSize: 11, fontFamily: 'JetBrains Mono' }} unit="ms" />
              <Tooltip content={<CustomTooltip />} />
              <Legend wrapperStyle={{ fontSize: 11, fontFamily: 'JetBrains Mono', color: '#4a6480' }} />
              <Bar dataKey="p50" fill="#00d4ff" radius={[3,3,0,0]} />
              <Bar dataKey="p90" fill="#7c3aed" radius={[3,3,0,0]} />
              <Bar dataKey="p99" fill="#ff1744" radius={[3,3,0,0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="grid-2 section">
        <div className="card">
          <div className="card-title">Throughput (TPS)</div>
          <div className="chart-wrap">
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={tpsData} margin={{ top: 10, right: 10, left: 0, bottom: 20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1e2d42" />
                <XAxis dataKey="name" tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} />
                <YAxis tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} />
                <Tooltip content={<CustomTooltip />} />
                <Bar dataKey="TPS" fill="#00e676" radius={[3,3,0,0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="card">
          <div className="card-title">Composite Score</div>
          <div className="chart-wrap">
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={scoreData} margin={{ top: 10, right: 10, left: 0, bottom: 20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1e2d42" />
                <XAxis dataKey="name" tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} />
                <YAxis tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} />
                <Tooltip content={<CustomTooltip />} />
                <Bar dataKey="Score" fill="#7c3aed" radius={[3,3,0,0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      <div className="grid-4 section">
        {sorted.slice(0, 4).map((e) => (
          <div key={e.submission_id} className="stat-box">
            <div className="stat-label">{e.team_name || 'Unknown'}</div>
            <div className="stat-value">{(e.composite_score || 0).toFixed(1)}</div>
            <div className="stat-unit">composite score</div>
            <div style={{ marginTop: '0.75rem', display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
              <span style={{ fontSize: '0.7rem', color: '#00d4ff', fontFamily: 'JetBrains Mono' }}>p99: {(e.p99_ms || e.p99 || 0).toFixed(1)}ms</span>
              <span style={{ fontSize: '0.7rem', color: '#00e676', fontFamily: 'JetBrains Mono' }}>{(e.correctness_rate || 0).toFixed(1)}% correct</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
