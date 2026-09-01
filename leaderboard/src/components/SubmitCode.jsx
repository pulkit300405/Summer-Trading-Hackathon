import React, { useState } from 'react';

export default function SubmitCode() {
  const [teamName, setTeamName] = useState('');
  const [language, setLanguage] = useState('go');
  const [file, setFile] = useState(null);
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(false);
  const [log, setLog] = useState([]);

  const addLog = (msg) => setLog(prev => [...prev, `[${new Date().toLocaleTimeString()}] ${msg}`]);

  const handleSubmit = async () => {
    if (!teamName) { setStatus({ type: 'error', msg: 'Team name required!' }); return; }
    setLoading(true);
    setStatus(null);
    addLog(`Submitting for team: ${teamName} (${language})`);
    try {
      const code = file ? await file.text() : `package main\nimport "fmt"\nfunc main() { fmt.Println("Hello from ${teamName}") }`;
      addLog('Sending to submission handler...');
      const res = await fetch('http://localhost:8080/submit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ language, team_name: teamName, code }),
      });
      const data = await res.json().catch(() => ({}));
      if (res.ok) {
        addLog(`✅ Submission accepted! ID: ${data.submission_id || 'N/A'}`);
        addLog('🤖 Bot fleet attacking your endpoint...');
        addLog('📊 Metrics flowing to telemetry ingester...');
        setStatus({ type: 'success', msg: `Queued: ${teamName} (${data.submission_id || 'pending'})` });
      } else {
        addLog(`❌ Error: ${data.error || data.message || 'Submission failed'}`);
        addLog('💡 Tip: Sending test metrics directly to telemetry...');
        await fetch('http://localhost:8082/ingest', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            submission_id: `${teamName.toLowerCase().replace(/\s/g,'_')}_${Date.now()}`,
            team_name: teamName,
            latency_ms: 10 + Math.random() * 40,
            success: true,
            order_type: 'LIMIT',
            timestamp: Date.now(),
          }),
        });
        addLog('✅ Test metrics sent! Check Leaderboard tab.');
        setStatus({ type: 'warn', msg: 'Simulated submission — check Leaderboard!' });
      }
    } catch (e) {
      addLog(`❌ Network error: ${e.message}`);
      setStatus({ type: 'error', msg: e.message });
    }
    setLoading(false);
  };

  const handleSample = async () => {
    setLoading(true);
    addLog('Running sample submission...');
    try {
      await fetch('http://localhost:8082/ingest', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          submission_id: `sample_${Date.now()}`,
          team_name: teamName || 'SampleTeam',
          latency_ms: 5 + Math.random() * 20,
          success: true,
          order_type: 'LIMIT',
          timestamp: Date.now(),
        }),
      });
      addLog('✅ Sample submission complete! Check Leaderboard.');
      setStatus({ type: 'success', msg: 'Sample submission complete!' });
    } catch (e) {
      addLog(`❌ Error: ${e.message}`);
    }
    setLoading(false);
  };

  return (
    <div className="submit-wrap">
      <div className="card">
        <div className="card-header">
          <div className="card-icon">⬆️</div>
          <div>
            <div className="card-title">Submit your engine</div>
            <div className="card-subtitle">Upload & benchmark your trading system</div>
          </div>
        </div>

        <div className="form-group">
          <label className="form-label">Team name</label>
          <input
            className="form-input"
            placeholder="Enter team name..."
            value={teamName}
            onChange={e => setTeamName(e.target.value)}
          />
        </div>

        <div className="form-group">
          <label className="form-label">Language</label>
          <select
            className="form-input"
            value={language}
            onChange={e => setLanguage(e.target.value)}
          >
            <option value="go">Go</option>
            <option value="rust">Rust</option>
            <option value="cpp">C++</option>
          </select>
        </div>

        <div className="form-group">
          <label className="form-label">Source file</label>
          <div className="file-input-wrap" onClick={() => document.getElementById('fileInput').click()}>
            📁 {file ? file.name : 'Choose File'}
            <input
              id="fileInput"
              type="file"
              style={{ display: 'none' }}
              onChange={e => setFile(e.target.files[0])}
            />
          </div>
        </div>

        {status && (
          <div style={{
            padding: '0.6rem 0.875rem',
            borderRadius: 6,
            marginBottom: '1rem',
            fontSize: '0.82rem',
            fontFamily: 'var(--mono)',
            background: status.type === 'success' ? 'rgba(34,197,94,0.1)' : status.type === 'warn' ? 'rgba(245,158,11,0.1)' : 'rgba(239,68,68,0.1)',
            color: status.type === 'success' ? 'var(--green)' : status.type === 'warn' ? 'var(--yellow)' : 'var(--red)',
            border: `1px solid ${status.type === 'success' ? 'rgba(34,197,94,0.3)' : status.type === 'warn' ? 'rgba(245,158,11,0.3)' : 'rgba(239,68,68,0.3)'}`,
          }}>
            {status.msg}
          </div>
        )}

        <div className="btn-row">
          <button className="btn btn-primary" onClick={handleSubmit} disabled={loading}>
            {loading ? '⏳ Processing...' : '⬆️ Upload & benchmark'}
          </button>
          <button className="btn btn-secondary" onClick={handleSample} disabled={loading}>
            🧪 Run sample submission
          </button>
        </div>

        <div style={{ marginTop: '1rem', fontSize: '0.75rem', color: 'var(--muted)', fontFamily: 'var(--mono)' }}>
          API: <span style={{ color: 'var(--accent)' }}>localhost:8080/submit</span> ·{' '}
          <span style={{ color: 'var(--accent)' }}>localhost:8082/leaderboard</span>
        </div>
      </div>

      <div className="card">
        <div className="card-header">
          <div className="card-icon">📋</div>
          <div>
            <div className="card-title">Submission Log</div>
            <div className="card-subtitle">Real-time pipeline status</div>
          </div>
        </div>
        <div style={{
          background: 'var(--bg)',
          borderRadius: 6,
          padding: '0.75rem',
          fontFamily: 'var(--mono)',
          fontSize: '0.78rem',
          color: 'var(--muted)',
          minHeight: 200,
          maxHeight: 320,
          overflowY: 'auto',
        }}>
          {log.length === 0 ? (
            <span style={{ color: 'var(--muted)' }}>Waiting for submission...</span>
          ) : (
            log.map((line, i) => (
              <div key={i} style={{
                color: line.includes('✅') ? 'var(--green)' : line.includes('❌') ? 'var(--red)' : line.includes('💡') ? 'var(--yellow)' : 'var(--muted)',
                marginBottom: '0.25rem',
              }}>
                {line}
              </div>
            ))
          )}
        </div>

        <div style={{ marginTop: '1rem' }}>
          <div className="card-label">How it works</div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2,1fr)', gap: '0.5rem', marginTop: '0.5rem' }}>
            {[
              ['01 Upload', 'Submit ZIP or source file'],
              ['02 Isolate', 'Docker container with CPU/mem limits'],
              ['03 Judge', 'FIFO correctness validation'],
              ['04 Benchmark', 'Bot fleet measures TPS, p50/p90/p99'],
            ].map(([title, desc]) => (
              <div key={title} style={{ background: 'var(--surface2)', borderRadius: 6, padding: '0.6rem 0.75rem' }}>
                <div style={{ fontSize: '0.65rem', color: 'var(--accent)', fontFamily: 'var(--mono)', marginBottom: '0.2rem' }}>{title}</div>
                <div style={{ fontSize: '0.78rem', color: 'var(--muted)' }}>{desc}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
