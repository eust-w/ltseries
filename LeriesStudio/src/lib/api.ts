/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { AuthStatus, ProductionAuditReport, Project, RuntimeConfig, Series, Task } from '../types';

const importMetaEnv = (import.meta as unknown as { env?: Record<string, string> }).env || {};
const API_BASE = (importMetaEnv.VITE_API_BASE_URL || '/api').replace(/\/$/, '');
const MEDIA_BASE = API_BASE.replace(/\/api$/, '');

export function mediaUrl(url?: string) {
  if (!url) return '';
  if (/^https?:\/\//i.test(url)) return url;
  return `${MEDIA_BASE}${url.startsWith('/') ? url : `/${url}`}`;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('token');
  const headers = new Headers(options.headers || {});

  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  if (!(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  const text = await response.text();
  const payload = text ? JSON.parse(text) : null;

  if (response.status === 401) {
    localStorage.removeItem('token');
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }

  if (!response.ok) {
    throw new Error(payload?.error || `HTTP error! status: ${response.status}`);
  }

  return payload as T;
}

async function requestArray<T>(path: string, options: RequestInit = {}): Promise<T[]> {
  const payload = await request<T[] | null>(path, options);
  return Array.isArray(payload) ? payload : [];
}

export const api = {
  // Auth
  getAuthStatus: () => request<AuthStatus>('/auth/status'),
  login: (body: { username: string; password: string }) =>
    request<{ token: string; expires_at: number; auth_disabled?: boolean }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  // System
  getConfig: () => request<RuntimeConfig>('/config'),
  updateConfig: (body: Partial<RuntimeConfig>) =>
    request<RuntimeConfig>('/config', {
      method: 'PUT',
      body: JSON.stringify(body),
    }),

  // Projects
  getProjects: () => requestArray<Project>('/projects'),
  getProject: (id: string) => request<Project>(`/projects/${id}`),
  getProductionAudit: (id: string) => request<ProductionAuditReport>(`/projects/${id}/production-audit`),
  createProject: (body: { title: string; script: string; workflow: string }) =>
    request<Project>('/projects', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  updateProject: (id: string, body: Partial<Project> | Record<string, unknown>) =>
    request<Project>(`/projects/${id}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  deleteProject: (id: string) => request<{ status: string; id: string }>(`/projects/${id}`, { method: 'DELETE' }),

  // Analysis
  analyzeProject: (id: string) => request<{ task_id: string; task: Task }>(`/projects/${id}/analyze`, { method: 'POST' }),
  reparseProject: (id: string) => request<{ task_id: string; task: Task }>(`/projects/${id}/reparse`, { method: 'POST' }),

  // Art Direction
  getStylePresets: () => request<{ presets: any[] }>('/art-direction/presets'),
  analyzeArtDirection: (id: string) => request<{ task_id: string; task: Task }>(`/projects/${id}/art-direction/analyze`, { method: 'POST' }),
  saveArtDirection: (id: string, body: Record<string, unknown>) =>
    request<Project>(`/projects/${id}/art-direction`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  // Assets
  addAsset: (projectId: string, type: string, body: Record<string, unknown>) =>
    request<Project>(`/projects/${projectId}/assets/${type}`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  updateAsset: (projectId: string, type: string, assetId: string, body: Record<string, unknown>) =>
    request<Project>(`/projects/${projectId}/assets/${type}/${assetId}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  deleteAsset: (projectId: string, type: string, assetId: string) =>
    request<Project>(`/projects/${projectId}/assets/${type}/${assetId}`, { method: 'DELETE' }),
  toggleAssetLock: (projectId: string, type: string, assetId: string) =>
    request<Project>(`/projects/${projectId}/assets/${type}/${assetId}/toggle-lock`, { method: 'POST' }),
  generateAssets: (projectId: string, body: Record<string, unknown> & { force?: boolean }) =>
    request<{ task_ids: string[]; tasks: Task[] }>(`/projects/${projectId}/generate_assets`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  generateAssetImage: (projectId: string, type: string, assetId: string, body: Record<string, unknown>) =>
    request<{ task_id: string; task: Task }>(`/projects/${projectId}/assets/${type}/${assetId}/generate`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  generateAssetVideo: (projectId: string, type: string, assetId: string, body: Record<string, unknown>) =>
    request<{ task_id: string; task: Task }>(`/projects/${projectId}/assets/${type}/${assetId}/generate_video`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  deleteAssetVideo: (projectId: string, type: string, assetId: string, uploadType: string, videoId: string) =>
    request<Project>(`/projects/${projectId}/assets/${type}/${assetId}/videos/${videoId}?upload_type=${encodeURIComponent(uploadType)}`, {
      method: 'DELETE',
    }),
  selectAssetVariant: (projectId: string, type: string, assetId: string, body: { upload_type: string; variant_id: string }) =>
    request<Project>(`/projects/${projectId}/assets/${type}/${assetId}/variant/select`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  favoriteAssetVariant: (projectId: string, type: string, assetId: string, uploadType: string, variantId: string, isFavorited: boolean) =>
    request<Project>(`/projects/${projectId}/assets/variant/favorite`, {
      method: 'POST',
      body: JSON.stringify({ asset_id: assetId, asset_type: type, upload_type: uploadType, variant_id: variantId, is_favorited: isFavorited }),
    }),
  deleteAssetVariant: (projectId: string, type: string, assetId: string, uploadType: string, variantId: string) =>
    request<Project>(`/projects/${projectId}/assets/variant/delete`, {
      method: 'POST',
      body: JSON.stringify({ asset_id: assetId, asset_type: type, upload_type: uploadType, variant_id: variantId }),
    }),
  uploadAsset: (projectId: string, type: string, assetId: string, uploadType: string, file: File, prompt = '') => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('prompt', prompt);
    return request<{ url: string; project: Project }>(`/projects/${projectId}/assets/${type}/${assetId}/upload?upload_type=${encodeURIComponent(uploadType)}`, {
      method: 'POST',
      body: formData,
    });
  },

  // Storyboard
  analyzeStoryboard: (projectId: string, text = '') =>
    request<{ task_id: string; task: Task }>(`/projects/${projectId}/storyboard/analyze`, {
      method: 'POST',
      body: JSON.stringify({ action: text }),
    }),
  renderStoryboard: (projectId: string, body: { frame_id: string; prompt: string; references?: string[] }) =>
    request<{ task_id: string; task: Task }>(`/projects/${projectId}/storyboard/render`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  completeStoryboard: (projectId: string, body: { model?: string; duration?: number; regenerate_images?: boolean; regenerate_videos?: boolean; parameters?: Record<string, unknown> } = {}) =>
    request<{ task_id: string; task: Task }>(`/projects/${projectId}/storyboard/complete`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  addFrame: (projectId: string, body: Record<string, unknown>) =>
    request<Project>(`/projects/${projectId}/frames`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  updateFrame: (projectId: string, frameId: string, body: Record<string, unknown>) =>
    request<Project>(`/projects/${projectId}/frames/${frameId}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  deleteFrame: (projectId: string, frameId: string) => request<Project>(`/projects/${projectId}/frames/${frameId}`, { method: 'DELETE' }),
  copyFrame: (projectId: string, frameId: string) => request<Project>(`/projects/${projectId}/frames/${frameId}/copy`, { method: 'POST' }),
  toggleFrameLock: (projectId: string, frameId: string) =>
    request<Project>(`/projects/${projectId}/frames/${frameId}/toggle-lock`, { method: 'POST' }),
  uploadFrameImage: (projectId: string, frameId: string, file: File, prompt = '') => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('prompt', prompt);
    return request<{ url: string; project: Project }>(`/projects/${projectId}/frames/${frameId}/upload_image`, {
      method: 'POST',
      body: formData,
    });
  },

  // Video / Audio
  createVideoTask: (projectId: string, body: Record<string, unknown>) =>
    request<{ task_id: string; task: Task }>(`/projects/${projectId}/video-tasks`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  mergeProject: (projectId: string) => request<{ task_id?: string; task?: Task; url?: string }>(`/projects/${projectId}/merge`, { method: 'POST' }),
  exportProject: (projectId: string) => request<{ task_id?: string; task?: Task; url?: string }>(`/projects/${projectId}/export`, { method: 'POST' }),

  // Project settings
  updateModelSettings: (projectId: string, body: Record<string, string>) =>
    request<Project>(`/projects/${projectId}/model-settings`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  updatePromptConfig: (projectId: string, body: Record<string, string>) =>
    request<Project>(`/projects/${projectId}/prompt-config`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),

  // Series
  listSeries: () => requestArray<Series>('/series'),
  createSeries: (body: { title: string; description: string; workflow: string }) =>
    request<Series>('/series', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  addSeriesEpisode: (seriesId: string, projectId: string) =>
    request<Series>(`/series/${seriesId}/episodes`, {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId }),
    }),
  removeSeriesEpisode: (seriesId: string, projectId: string) =>
    request<Series>(`/series/${seriesId}/episodes/${projectId}`, { method: 'DELETE' }),

  // Tasks
  getTasks: (projectId?: string) => requestArray<Task>(`/tasks${projectId ? `?project_id=${encodeURIComponent(projectId)}` : ''}`),
  deleteTask: (id: string) => request<{ status: string; id: string }>(`/tasks/${id}`, { method: 'DELETE' }),
};
