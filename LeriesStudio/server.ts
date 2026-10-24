import 'dotenv/config';
import express from 'express';
import type { Server } from 'http';
import path from 'path';
import { createServer as createViteServer } from 'vite';

const DEFAULT_BACKEND_URL = 'http://localhost:18080';
let activeServer: Server | undefined;

function backendURL() {
  return (process.env.BACKEND_URL || process.env.VITE_BACKEND_URL || DEFAULT_BACKEND_URL).replace(/\/$/, '');
}

async function proxyToBackend(req: express.Request, res: express.Response) {
  const target = `${backendURL()}${req.originalUrl}`;
  const headers = new Headers();
  for (const [key, value] of Object.entries(req.headers)) {
    if (!value || key.toLowerCase() === 'host' || key.toLowerCase() === 'content-length') continue;
    headers.set(key, Array.isArray(value) ? value.join(',') : value);
  }

  try {
    const init: RequestInit & { duplex?: 'half' } = {
      method: req.method,
      headers,
    };
    if (req.method !== 'GET' && req.method !== 'HEAD') {
      init.body = req as unknown as BodyInit;
      init.duplex = 'half';
    }

    const upstream = await fetch(target, init);

    res.status(upstream.status);
    upstream.headers.forEach((value, key) => {
      if (['content-encoding', 'content-length', 'transfer-encoding', 'connection'].includes(key.toLowerCase())) return;
      res.setHeader(key, value);
    });

    const body = Buffer.from(await upstream.arrayBuffer());
    res.send(body);
  } catch (error) {
    const message = error instanceof Error ? error.message : 'backend proxy failed';
    res.status(502).json({ error: message });
  }
}

async function startServer() {
  const app = express();
  const port = Number(process.env.PORT || 3108);

  app.use('/api', proxyToBackend);
  app.use('/files', proxyToBackend);

  if (process.env.NODE_ENV !== 'production') {
    const vite = await createViteServer({
      server: { middlewareMode: true },
      appType: 'spa',
    });
    app.use(vite.middlewares);
  } else {
    const distPath = path.join(process.cwd(), 'dist');
    app.use(express.static(distPath));
    app.get('*', (_req, res) => {
      res.sendFile(path.join(distPath, 'index.html'));
    });
  }

  activeServer = app.listen(port, '0.0.0.0', () => {
    console.log(`LeriesStudio listening on http://localhost:${port}, proxying API to ${backendURL()}`);
  });
  return activeServer;
}

startServer().catch((error) => {
  console.error(error);
  process.exit(1);
});
