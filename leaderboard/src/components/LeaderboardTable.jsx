import React from 'react';

const LeaderboardTable = ({ data, onSelectSubmission, selectedSubmission }) => {
  if (!data || data.length === 0) {
    return <div style={{ padding: '2rem', textAlign: 'center' }}>No submissions yet</div>;
  }

  const getRankBadgeClass = (rank) => {
    if (rank === 1) return 'rank-1';
    if (rank === 2) return 'rank-2';
    if (rank === 3) return 'rank-3';
    return 'rank-other';
  };

  const formatMetric = (value, decimal Places = 2) => {
    if (value === null || value === undefined) return 'N/A';
    return typeof value === 'number' ? value.toFixed(decimalPlaces) : value;
  };

  const getMetricClass = (metric, value) => {
    if (metric === 'latency' && value < 50) return 'good';
    if (metric === 'latency' && value < 100) return 'warning';
    if (metric === 'latency') return 'critical';
    if (metric === 'throughput' && value > 5000) return 'good';
    if (metric === 'correctness' && value > 99) return 'good';
    return '';
  };

  return (
    <table className="leaderboard-table">
      <thead>
        <tr>
          <th style={{ width: '5%' }}>Rank</th>
          <th style={{ width: '20%' }}>Team Name</th>
          <th style={{ width: '12%' }}>p99 Latency (ms)</th>
          <th style={{ width: '12%' }}>Throughput (TPS)</th>
          <th style={{ width: '12%' }}>Correctness (%)</th>
          <th style={{ width: '12%' }}>Orders</th>
          <th style={{ width: '15%' }}>Score</th>
        </tr>
      </thead>
      <tbody>
        {data.map((entry, index) => (
          <tr
            key={entry.submission_id}
            className={selectedSubmission?.submission_id === entry.submission_id ? 'selected' : ''}
            onClick={() => onSelectSubmission(entry)}
            style={{ cursor: 'pointer' }}
          >
            <td>
              <div className={`rank-badge ${getRankBadgeClass(entry.rank || index + 1)}`}>
                {entry.rank || index + 1}
              </div>
            </td>
            <td>{entry.team_name || 'Unknown'}</td>
            <td>
              <span className={`metric ${getMetricClass('latency', entry.p99_latency_ms)}`}>
                {formatMetric(entry.p99_latency_ms, 1)}
              </span>
            </td>
            <td>
              <span className={`metric ${getMetricClass('throughput', entry.throughput_tps)}`}>
                {formatMetric(entry.throughput_tps, 0)}
              </span>
            </td>
            <td>
              <span className={`metric ${getMetricClass('correctness', entry.correctness_rate)}`}>
                {formatMetric(entry.correctness_rate, 1)}
              </span>
            </td>
            <td>
              <span className="metric">
                {formatMetric(entry.orders_processed, 0)}
              </span>
            </td>
            <td>
              <strong>
                {formatMetric(entry.composite_score, 0)}
              </strong>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
};

export default LeaderboardTable;
