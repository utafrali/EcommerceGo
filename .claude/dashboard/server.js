#!/usr/bin/env node

const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = 3333;
const STATUS_FILE = path.join(__dirname, '..', 'pipeline', 'status.json');
const HTML_FILE = path.join(__dirname, 'index.html');

// SSE clients
const clients = new Set();

// Watch status.json for changes
let lastMtime = null;

function watchStatusFile() {
  setInterval(() => {
    try {
      const stats = fs.statSync(STATUS_FILE);
      if (!lastMtime || stats.mtimeMs !== lastMtime) {
        lastMtime = stats.mtimeMs;
        const data = JSON.parse(fs.readFileSync(STATUS_FILE, 'utf8'));
        broadcast(data);
      }
    } catch (err) {
      console.error('Error reading status file:', err.message);
    }
  }, 500); // Check every 500ms
}

function broadcast(data) {
  const message = `data: ${JSON.stringify(data)}\n\n`;
  clients.forEach(client => {
    try {
      client.write(`event: update\n${message}`);
    } catch (err) {
      clients.delete(client);
    }
  });
}

// HTTP Server
const server = http.createServer((req, res) => {
  const url = req.url;

  // CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  // Routes
  if (url === '/' || url === '/index.html') {
    // Serve HTML
    try {
      const html = fs.readFileSync(HTML_FILE, 'utf8');
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end(html);
    } catch (err) {
      res.writeHead(500);
      res.end('Error loading dashboard');
    }
  } else if (url === '/api/status') {
    // Serve current status as JSON
    try {
      const data = JSON.parse(fs.readFileSync(STATUS_FILE, 'utf8'));
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(data));
    } catch (err) {
      res.writeHead(500, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Failed to read status' }));
    }
  } else if (url === '/events') {
    // SSE endpoint
    res.writeHead(200, {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
    });

    // Send initial data
    try {
      const data = JSON.parse(fs.readFileSync(STATUS_FILE, 'utf8'));
      res.write(`event: update\ndata: ${JSON.stringify(data)}\n\n`);
    } catch (err) {
      console.error('Error sending initial data:', err.message);
    }

    // Add to clients
    clients.add(res);

    // Remove on close
    req.on('close', () => {
      clients.delete(res);
    });
  } else {
    res.writeHead(404);
    res.end('Not found');
  }
});

server.listen(PORT, () => {
  console.log(`\n🚀 EcommerceGo Pipeline Dashboard`);
  console.log(`📊 Server running at http://localhost:${PORT}/`);
  console.log(`📡 SSE endpoint: http://localhost:${PORT}/events`);
  console.log(`📄 Status file: ${STATUS_FILE}\n`);
  watchStatusFile();
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('Shutting down dashboard server...');
  clients.forEach(client => client.end());
  server.close(() => process.exit(0));
});

process.on('SIGINT', () => {
  console.log('\nShutting down dashboard server...');
  clients.forEach(client => client.end());
  server.close(() => process.exit(0));
});
