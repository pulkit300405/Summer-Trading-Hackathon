import React, { useState, useEffect } from 'react';
import './App.css';
import LeaderboardTable from './components/LeaderboardTable';
import MetricsGraph from './components/MetricsGraph';

function App() {
  const [leaderboard, setLeaderboard] = useState([]);
  const [selectedSubmission, setSelectedSubmission] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Fetch leaderboard data
  useEffect(() => {
    const fetchLeaderboard = async () => {
      try {
        const response = await fetch('http://localhost:8082/leaderboard');
        if (!response.ok) throw new Error('Failed to fetch leaderboard');
        const data = await response.json();
        setLeaderboard(data || []);
        setLoading(false);
      } catch (err) {
        setError(err.message);
        setLoading(false);
      }
    };

    // Fetch immediately and then every 5 seconds
    fetchLeaderboard();
    const interval = setInterval(fetchLeaderboard, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="app">
      <header className="app-header">
        <h1>🏆 IICPC Summer Hackathon 2026</h1>
        <p>Distributed Benchmarking & Hosting Platform</p>
        <p className="subtitle">Real-Time Leaderboard</p>
      </header>

      <main className="app-main">
        {loading && <div className="loading">Loading leaderboard...</div>}
        {error && <div className="error">Error: {error}</div>}

        {!loading && !error && (
          <>
            <section className="leaderboard-section">
              <h2>Rankings</h2>
              <LeaderboardTable
                data={leaderboard}
                onSelectSubmission={setSelectedSubmission}
                selectedSubmission={selectedSubmission}
              />
            </section>

            {selectedSubmission && (
              <section className="metrics-section">
                <h2>Detailed Metrics: {selectedSubmission.team_name}</h2>
                <MetricsGraph submission={selectedSubmission} />
              </section>
            )}
          </>
        )}
      </main>

      <footer className="app-footer">
        <p>Last updated: {new Date().toLocaleTimeString()}</p>
      </footer>
    </div>
  );
}

export default App;
