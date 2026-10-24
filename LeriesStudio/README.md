# LTSeries Studio

React + TypeScript frontend for [LTSeries](../README.md).

## Development

Start MongoDB and the Go API using the [root setup guide](../README.md#local-development), then:

```bash
npm ci
cp .env.example .env
npm run dev
```

Open <http://localhost:3108>. The frontend server proxies `/api` and `/files` to
`BACKEND_URL` (default: `http://localhost:18080`). AI keys belong in the project-root
`.env`, never in browser environment variables.

```bash
npm run lint
npm run build
NODE_ENV=production npm start
```

See the [API reference](../docs/frontend-api.md) and [contribution guide](../CONTRIBUTING.md).
