# Deployment & Troubleshooting

## Supported Starting Points

- **Local development:** Go API + Node frontend + MongoDB in Docker.
- **Single-machine Compose:** containerized API/frontend/MongoDB, including FFmpeg.

This is a single-admin workspace. Multi-tenant authorization, per-project users,
rate limiting and a complete public SaaS security model are not implemented.
Keep a self-hosted instance on a trusted network unless you add the appropriate
access controls and HTTPS reverse proxy.

## Configuration

Copy `.env.example` to `.env` and supply your provider key, login password and
session secret. Do not commit `.env`. There is no shared default login password.

The frontend never needs an AI key. Its Node server proxies `/api` and `/files`
to `BACKEND_URL`. In Compose the target is `http://backend:18080`.

The Go API uses `CONFIG_PATH` to save settings changes. Compose mounts the root
`.env` at `/app/config/.env`, making UI configuration changes persistent on the
host. Settings updates change the worker process environment; restart/recreate
the API after changing endpoint-level or startup-only settings.

```bash
docker compose up -d --force-recreate backend
```

## Data & Backups

| Data | Local development | Docker Compose |
| --- | --- | --- |
| Project, series and task records | `ltseries_mongo_data` Docker volume | Compose `mongo_data` volume |
| Uploaded/extracted/assembled media | `STORAGE_DIR` (typically `backend/storage`) | Compose `media_data` volume |
| Credentials and editable config | Root `.env` | Host `.env`, mounted in API container |
| Provider-generated media | Often a temporary provider URL | Often a temporary provider URL |

Back up MongoDB, the media directory/volume and your configuration separately.
A Git clone restores code, not projects or generated media. `docker compose down`
keeps named volumes; `docker compose down -v` deletes them.

## Reference URLs & Consistency

`PUBLIC_BASE_URL` controls the URLs returned for files hosted by the API. It does
not upload data, configure a domain, create a tunnel, or make localhost public.

Cloud providers must be able to fetch reference images and videos without your
local browser session. In particular:

- `localhost`, `127.0.0.1` and private-network files are not reachable by DashScope.
- Expired signed URLs cannot be used as reliable references.
- The current pipeline may retry without references after certain provider
  access errors. A completed task does not prove that all references influenced it.
- Last-frame records describe extracted media. Their existence does not prove
  that the next generation used them.
- Audit scores are metadata checks, not visual similarity measurements.

For remote generation using your own uploads, configure a reachable HTTPS media
origin or supply suitable external URLs. Automatic OSS/S3 archival is not
implemented. Current `/files` routes serve media without authentication; do not
place confidential files on a public media origin without adding access controls.

The **Refresh reference assets** control generates new images and can incur
charges. It does not renew the signature of an old URL or preserve the exact
previous image.

## Diagnostics

```bash
docker compose ps
docker compose logs --tail=100 backend
docker compose logs --tail=100 frontend
curl -fsS http://localhost:18080/api/health
curl -fsS http://localhost:3108/api/health
docker compose exec -T backend ffmpeg -version
```

Logs may contain provider response details. Redact credentials, signed URLs and
private story content before including them in an issue.

| Symptom | Check |
| --- | --- |
| Login fails | `AUTH_ENABLED`, `ADMIN_USERNAME`, `ADMIN_PASSWORD`, `AUTH_SESSION_SECRET`; recreate the API after startup-config changes. |
| Frontend returns 502 | Backend health and `BACKEND_URL`; inside Compose use the service hostname, not localhost. |
| Provider returns 401 | Credential validity and endpoint/region alignment; use a key authorized for the selected service. |
| Model request fails | Actual model availability, account permissions, mode, media format and provider response. A catalog entry is only a preset. |
| Image/video was visible yesterday | The provider URL may have expired. Existing outputs are not automatically archived permanently. |
| Reference inspection fails | Public accessibility, URL expiry, content format and provider media requirements. |
| Audio task fails | Set `AUDIO_PROVIDER_ENDPOINT` to a compatible service; audio is not built into the default DashScope adapter. |
| Merge fails | Verify FFmpeg, readable input clips and compatible media; inspect the stored task error. |
| Go tests cannot connect | Start local MongoDB, or set `LTSERIES_TEST_MONGODB_URI` to a dedicated test server. |

## Verification Scope

CI tests project APIs and MongoDB task behavior, checks the frontend, builds the
containers and probes health endpoints. Provider payload tests use local HTTP
test servers. CI does not certify paid model execution, generated-media quality
or visual consistency. Revalidate those in your own provider account.
