const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = 3333;
const DIR = __dirname;
const STATUS_FILE = path.join(DIR, 'status.json');
const DASHBOARD_FILE = path.join(DIR, 'dashboard.html');

const server = http.createServer((req, res) => {
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
});
