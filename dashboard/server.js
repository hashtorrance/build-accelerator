const express = require('express');
const http = require('http');
const WebSocket = require('ws');
const fs = require('fs');
const path = require('path');
const yaml = require('yaml');

const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ server });

// Load configuration
const configPath = path.join(__dirname, '..', 'config.yml');
const config = yaml.parse(fs.readFileSync(configPath, 'utf8'));

const PORT = config.dashboard.port || 9876;

// Mock metrics for demo (in production, would read from cache service)
let metrics = {
  cacheHitRate: 0.0,
  buildTimeSaved: 0,
  activePeers: 0,
  totalBuilds: 0,
  compilerInvocations: []
};

// Serve static dashboard
app.use(express.static(path.join(__dirname, 'public')));

app.get('/api/metrics', (req, res) => {
  res.json(metrics);
});

app.get('/api/config', (req, res) => {
  res.json(config);
});

// WebSocket for real-time updates
wss.on('connection', (ws) => {
  console.log('Dashboard client connected');

  // Send initial metrics
  ws.send(JSON.stringify(metrics));

  // Simulate periodic updates
  const interval = setInterval(() => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(metrics));
    }
  }, config.dashboard.refresh_interval_ms || 2000);

  ws.on('close', () => {
    clearInterval(interval);
    console.log('Dashboard client disconnected');
  });
});

server.listen(PORT, () => {
  console.log(`Build Accelerator Dashboard running on http://localhost:${PORT}`);
  console.log(`Metrics API: http://localhost:${PORT}/api/metrics`);
  console.log(`WebSocket updates every ${config.dashboard.refresh_interval_ms}ms`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('Shutting down dashboard server...');
  server.close(() => {
    process.exit(0);
  });
});
