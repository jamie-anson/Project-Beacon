#!/usr/bin/env node
/**
 * serve-dist-with-proxy.js
 *
 * Simple static file server for dist/ with minimal proxy for /api and /health to a target backend.
 * No external deps.
 *
 * Usage:
 *   node scripts/serve-dist-with-proxy.js --port 5050 --target http://localhost:8787
 */

const http = require('http');
const fs = require('fs');
const path = require('path');
const { URL } = require('url');

const args = process.argv.slice(2);
function getArg(name, def) {
  const i = args.indexOf(`--${name}`);
  if (i !== -1 && args[i + 1]) return args[i + 1];
  return def;
}

const ROOT = path.join(__dirname, '..', 'dist');
const PORT = Number(getArg('port', '5050'));
const TARGET = getArg('target', 'http://localhost:8787');

function contentType(fp) {
  const ext = path.extname(fp).toLowerCase();
  switch (ext) {
    case '.html': return 'text/html; charset=utf-8';
    case '.js': return 'application/javascript; charset=utf-8';
    case '.css': return 'text/css; charset=utf-8';
    case '.json': return 'application/json; charset=utf-8';
    case '.svg': return 'image/svg+xml';
    case '.png': return 'image/png';
    case '.jpg':
    case '.jpeg': return 'image/jpeg';
    case '.webp': return 'image/webp';
    default: return 'application/octet-stream';
  }
}

function serveFile(res, fp) {
  try {
    const data = fs.readFileSync(fp);
    res.writeHead(200, { 'Content-Type': contentType(fp) });
    res.end(data);
  } catch (e) {
    res.writeHead(404, { 'Content-Type': 'text/plain' });
    res.end('Not found');
  }
}

function serveSpa(res) {
  const index = path.join(ROOT, 'index.html');
  serveFile(res, index);
}

function proxyRequest(req, res, targetBase) {
  try {
    const targetUrl = new URL(req.url, targetBase);
    const opts = {
      method: req.method,
      headers: req.headers,
    };
    const proxReq = http.request(targetUrl, (proxRes) => {
      res.writeHead(proxRes.statusCode || 502, proxRes.headers);
      proxRes.pipe(res);
    });
    proxReq.on('error', (err) => {
      res.writeHead(502, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Bad gateway', message: err.message }));
    });
    if (req.method !== 'GET' && req.method !== 'HEAD') {
      req.pipe(proxReq);
    } else {
      proxReq.end();
    }
  } catch (e) {
    res.writeHead(500, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Proxy error', message: e.message }));
  }
}

const server = http.createServer((req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);

  // Minimal proxy for API endpoints expected by the portal build
  if (url.pathname === '/health' || url.pathname.startsWith('/api/')) {
    return proxyRequest(req, res, TARGET);
  }

  // Static files from dist/
  let fp = path.join(ROOT, url.pathname);
  try {
    const st = fs.statSync(fp);
    if (st.isDirectory()) fp = path.join(fp, 'index.html');
    return serveFile(res, fp);
  } catch {
    // SPA fallback for /portal/* and other client routes
    return serveSpa(res);
  }
});

server.listen(PORT, () => {
  console.log(`[serve-dist-with-proxy] Serving ${ROOT} on http://localhost:${PORT} with proxy to ${TARGET}`);
  console.log(`[serve-dist-with-proxy] Proxying /health and /api/* -> ${TARGET}`);
});
