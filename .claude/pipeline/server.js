const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = 3333;
const DIR = __dirname;
const STATUS_FILE = path.join(DIR, 'status.json');
const DASHBOARD_FILE = path.join(DIR, 'dashboard.html');

let sseClients = [];

// Watch status.json for changes and push to all SSE clients
let debounce = null;
fs.watch(STATUS_FILE, () => {
  clearTimeout(debounce);
  debounce = setTimeout(() => {
    try {
      const data = fs.readFileSync(STATUS_FILE, 'utf-8');
      JSON.parse(data); // validate JSON
      sseClients.forEach(res => {
        res.write(`data: ${data}\n\n`);
      });
    } catch (e) { /* ignore parse errors during write */ }
  }, 100);
});

const server = http.createServer((req, res) => {
  // SSE endpoint for realtime updates
  if (req.url.startsWith('/api/stream')) {
    res.writeHead(200, {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
      'Access-Control-Allow-Origin': '*',
    });
    // Send current state immediately
    try {
      const data = fs.readFileSync(STATUS_FILE, 'utf-8');
      res.write(`data: ${data}\n\n`);
    } catch (e) {}
    sseClients.push(res);
    req.on('close', () => {
      sseClients = sseClients.filter(c => c !== res);
    });
    return;
  }

  if (req.url.startsWith('/api/status')) {
    try {
      const data = fs.readFileSync(STATUS_FILE, 'utf-8');
      res.writeHead(200, { 'Content-Type': 'application/json', 'Cache-Control': 'no-cache' });
      res.end(data);
    } catch (e) {
      res.writeHead(500, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Status file not found' }));
    }
  } else {
    try {
      const html = fs.readFileSync(DASHBOARD_FILE, 'utf-8');
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end(html);
    } catch (e) {
      res.writeHead(500, { 'Content-Type': 'text/plain' });
      res.end('Dashboard not found');
    }
  }
});

server.listen(PORT, () => {
  console.log(`Pipeline Dashboard running at http://localhost:${PORT}`);
  console.log(`  Realtime SSE at http://localhost:${PORT}/api/stream`);
});
