import React, { useState, useEffect } from 'react';
import './App.css';
import SystemStatus from './components/SystemStatus';
import SubmitCode from './components/SubmitCode';
import Leaderboard from './components/Leaderboard';
import LiveMonitor from './components/LiveMonitor';
import BotTelemetry from './components/BotTelemetry';

const TABS = [
  { id: 'system', label: 'System Status', icon: '📡' },
  { id: 'submit', label: 'Submit Code', icon: '⬆️' },
  { id: 'leaderboard', label: 'Leaderboard', icon: '🏆' },
  { id: 'monitor', label: 'Live Monitor', icon: '⚡' },
  { id: 'bots', label: 'Bot Fleet', icon: '🤖' },
];

function App() {
  const [activeTab, setActiveTab] = useState('system');
  const [leaderboard, setLeaderboard] = useState([]);
  const [botStats, setBotStats] = useState(null);
  const [health, setHealth] = useState({ submission: false, telemetry: false, bot: false });
  const [lastUpdate, setLastUpdate] = useState(null);
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    const start = Date.now();
    const timer = setInterval(() => setElapsed(Math.floor((Date.now() - start) / 1000)), 1000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    const fetchAll = async () => {
      try {
        const lb = await fetch('http://localhost:8082/leaderboard').then(r => r.json()).catch(() => ({ leaderboard: [] }));
        setLeaderboard(lb.leaderboard || []);
        const bots = await fetch('http://localhost:8081/stats').then(r => r.json()).catch(() => null);
        setBotStats(bots);
        const sh = await fetch('http://localhost:8080/health').then(r => r.ok).catch(() => false);
        const th = await fetch('http://localhost:8082/health').then(r => r.ok).catch(() => false);
        const bh = await fetch('http://localhost:8081/health').then(r => r.ok).catch(() => false);
        setHealth({ submission: sh, telemetry: th, bot: bh });
        setLastUpdate(new Date());
      } catch (e) {}
    };
    fetchAll();
    const interval = setInterval(fetchAll, 2000);
    return () => clearInterval(interval);
  }, []);

  const formatTime = (s) => {
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const sec = s % 60;
    return `${String(h).padStart(2,'0')}:${String(m).padStart(2,'0')}:${String(sec).padStart(2,'0')}`;
  };

  return (
    <div className="app">
      <header className="header">
        <div className="header-left">
          <div className="logo">⚡</div>
          <div className="header-title">
            <span className="title-main">IICPC Dashboard</span>
          </div>
        </div>
        <nav className="tabs">
          {TABS.map(tab => (
            <button
              key={tab.id}
              className={`tab ${activeTab === tab.id ? 'active' : ''}`}
              onClick={() => setActiveTab(tab.id)}
            >
              <span className="tab-icon">{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </nav>
        <div className="header-right">
          <div className="timer">
            <span className="timer-icon">⏱</span>
            {formatTime(elapsed)}
            <span className="timer-label">UPTIME</span>
          </div>
          <div className="live-dot">
            <span className="pulse" />
            LIVE
          </div>
        </div>
      </header>

      <main className="main">
        {activeTab === 'system' && <SystemStatus health={health} botStats={botStats} lastUpdate={lastUpdate} />}
        {activeTab === 'submit' && <SubmitCode />}
        {activeTab === 'leaderboard' && <Leaderboard data={leaderboard} />}
        {activeTab === 'monitor' && <LiveMonitor data={leaderboard} botStats={botStats} />}
        {activeTab === 'bots' && <BotTelemetry stats={botStats} />}
      </main>
    </div>
  );
}

export default App;
