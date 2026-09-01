import React, { useState, useEffect, useRef } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

const CustomTooltip = ({ active, payload, label }) => {
  if (!active || !payload?.length) return null;
  return (
    <div style={{ background: '#0d1420', border: '1px solid #1e2d42', borderRadius: 6, padding: '0.75rem', fontFamily: 'JetBrains Mono, monospace', fontSize: '0.78rem' }}>
      <div style={{ color: '#4a6480', marginBottom: 4 }}>{label}s</div>
      {payload.map(p => (
        <div key={p.name} style={{ color: p.color }}>{p.name}: {p.value?.toLocaleString()}</div>
      ))}
    </div>
  );
};

export default function BotTelemetry({ stats }) {
  const [history, setHistory] = useState([]);
  const [elapsed, setElapsed] = useState(0);
  const startRef = useRef(Date.now());

  useEffect(() => {
    const timer = setInterval(() => setElapsed(Math.floor((Date.now() - startRef.current) / 1000)), 1000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    if (!stats) return;
    setHistory(prev => [...prev, {
      time: Math.floor((Date.now() - startRef.current) / 1000),
      orders: stats.orders_sent || 0,
      bots: stats.total_bots || 0,
    }].slice(-60));
  }, [stats]);

  const ordersPerSec = history.length > 1
    ? Math.max(0, Math.round(history[history.length-1]?.orders - history[history.length-2]?.orders))
    : 0;

  return (
    <div>
      <div className="bot-grid section">
        <div className="stat-box">
          <div className="stat-label">Total Bots</div>
          <div className="stat-value">{stats?.total_bots?.toLocaleString() || '—'}</div>
          <div className="stat-unit">goroutines</div>
        </div>
        <div className="stat-box">
          <div className="stat-label">Orders Sent</div>
          <div className="stat-value">{stats?.orders_sent?.toLocaleString() || '—'}</div>
          <div className="stat-unit">total</div>
        </div>
        <div className="stat-box">
          <div className="stat-label">Orders / sec</div>
          <div className="stat-value" style={{ color: 'var(--green)' }}>{ordersPerSec > 0 ? ordersPerSec.toLocaleString() : '—'}</div>
          <div className="stat-unit">live TPS</div>
        </div>
        <div className="stat-box">
          <div className="stat-label">Uptime</div>
          <div className="stat-value" style={{ color: 'var(--accent)' }}>{`${Math.floor(elapsed/60)}m ${elapsed%60}s`}</div>
          <div className="stat-unit">this session</div>
        </div>
      </div>

      <div className="card section">
        <div className="card-title">Orders Sent — Live</div>
        <div className="chart-wrap">
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={history} margin={{ top: 10, right: 20, left: 0, bottom: 10 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#1e2d42" />
              <XAxis dataKey="time" tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} tickFormatter={v => `${v}s`} />
              <YAxis tick={{ fill: '#4a6480', fontSize: 10, fontFamily: 'JetBrains Mono' }} tickFormatter={v => v.toLocaleString()} />
              <Tooltip content={<CustomTooltip />} />
              <Line type="monotone" dataKey="orders" name="Orders" stroke="#00d4ff" strokeWidth={2} dot={false} isAnimationActive={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="card">
        <div className="card-title">Bot Configuration</div>
        <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '0.82rem', lineHeight: 2, color: '#e8f4fd' }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.5rem' }}>
            {[
              ['Concurrency Model', 'Go Goroutines (M:N scheduling)'],
              ['Order Distribution', 'LIMIT 70% / MARKET 20% / CANCEL 10%'],
              ['HTTP Client', 'Shared pool (MaxIdleConns: 1000)'],
              ['Telemetry', 'p50 / p90 / p99 latency tracking'],
              ['Target TPS', '50,000 orders/second'],
              ['Memory Usage', '~1MB per 1000 goroutines'],
            ].map(([k, v]) => (
              <React.Fragment key={k}>
                <span style={{ color: '#4a6480' }}>{k}</span>
                <span style={{ color: '#00d4ff' }}>{v}</span>
              </React.Fragment>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
