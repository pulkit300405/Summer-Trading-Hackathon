import React from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';

const MetricsGraph = ({ submission }) => {
  if (!submission) return null;

  // Prepare data for latency chart
  const latencyData = [
    { name: 'p50', latency: submission.p50_latency_ms || 0 },
    { name: 'p90', latency: submission.p90_latency_ms || 0 },
    { name: 'p99', latency: submission.p99_latency_ms || 0 },
  ];

  // Prepare data for performance metrics
  const performanceData = [
    { name: 'Throughput (TPS)', value: submission.throughput_tps || 0, max: 15000 },
    { name: 'Correctness (%)', value: submission.correctness_rate || 0, max: 100 },
  ];

  return (
    <div className="metrics-grid" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '2rem', marginTop: '1rem' }}>
      {/* Latency Percentiles */}
      <div className="metric-chart" style={{ background: 'rgba(255,255,255,0.05)', borderRadius: '8px', padding: '1rem' }}>
        <h3 style={{ marginBottom: '1rem', color: '#667eea' }}>Latency Percentiles (ms)</h3>
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={latencyData}>
            <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
            <XAxis dataKey="name" stroke="#999" />
            <YAxis stroke="#999" />
            <Tooltip
              contentStyle={{ background: 'rgba(0,0,0,0.8)', border: 'none', borderRadius: '4px' }}
              labelStyle={{ color: '#fff' }}
            />
            <Bar dataKey="latency" fill="#667eea" name="Latency (ms)" />
          </BarChart>
        </ResponsiveContainer>
      </div>

      {/* Key Metrics */}
      <div className="metric-summary" style={{ background: 'rgba(255,255,255,0.05)', borderRadius: '8px', padding: '1rem' }}>
        <h3 style={{ marginBottom: '1rem', color: '#667eea' }}>Summary</h3>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
          <MetricCard
            label="Composite Score"
            value={submission.composite_score}
            unit=""
            color="#667eea"
          />
          <MetricCard
            label="Orders Processed"
            value={submission.orders_processed}
            unit=""
            color="#51cf66"
          />
          <MetricCard
            label="Correctness Rate"
            value={submission.correctness_rate}
            unit="%"
            color="#ffd43b"
          />
          <MetricCard
            label="Throughput"
            value={submission.throughput_tps}
            unit="TPS"
            color="#667eea"
          />
        </div>
      </div>
    </div>
  );
};

const MetricCard = ({ label, value, unit, color }) => {
  const formatted = typeof value === 'number' ? value.toFixed(2) : value || '0';

  return (
    <div style={{
      padding: '1rem',
      background: `rgba(${color === '#667eea' ? '102,126,234' : color === '#51cf66' ? '81,207,102' : '255,212,59'}, 0.1)`,
      borderLeft: `4px solid ${color}`,
      borderRadius: '4px',
    }}>
      <div style={{ fontSize: '0.9rem', color: '#999', marginBottom: '0.25rem' }}>
        {label}
      </div>
      <div style={{ fontSize: '1.5rem', fontWeight: 'bold', color }}>
        {formatted} {unit}
      </div>
    </div>
  );
};

export default MetricsGraph;
