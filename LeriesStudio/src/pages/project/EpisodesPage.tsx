/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { FormEvent, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { ArrowRight, Film, Layers, Plus, RefreshCcw, Sparkles, Trash2 } from 'lucide-react';
import { useProject } from '../../contexts/ProjectContext';
import { api } from '../../lib/api';
import { formatDate } from '../../lib/utils';
import { Project, Series } from '../../types';

export default function EpisodesPage() {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const { currentProject, setCurrentProjectId, refreshProject, refreshTasks } = useProject();
  const [seriesList, setSeriesList] = useState<Series[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedSeriesId, setSelectedSeriesId] = useState('');
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (projectId) setCurrentProjectId(projectId);
  }, [projectId, setCurrentProjectId]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [seriesData, projectData] = await Promise.all([api.listSeries(), api.getProjects()]);
      const normalizedSeries = Array.isArray(seriesData) ? seriesData : [];
      const normalizedProjects = Array.isArray(projectData) ? projectData : [];
      setSeriesList(normalizedSeries);
      setProjects(normalizedProjects);
      const linked = normalizedProjects.find((project) => project.id === projectId)?.series_id;
      setSelectedSeriesId(linked || normalizedSeries[0]?.id || '');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [projectId]);

  const projectById = useMemo(() => new Map(projects.map((project) => [project.id, project])), [projects]);
  const selectedSeries = seriesList.find((series) => series.id === selectedSeriesId) || null;
  const selectedEpisodes = (selectedSeries?.episode_ids || [])
    .map((id) => projectById.get(id))
    .filter(Boolean) as Project[];

  const handleAnalyze = async () => {
    if (!projectId) return;
    setSubmitting(true);
    setMessage('');
    try {
      await api.analyzeProject(projectId);
      await refreshTasks();
      setMessage('当前项目的剧本解析任务已提交');
    } catch (error: any) {
      setMessage(error?.message || '任务提交失败');
    } finally {
      setSubmitting(false);
    }
  };

  const attachCurrentProject = async () => {
    if (!projectId || !selectedSeriesId) return;
    setSubmitting(true);
    setMessage('');
    try {
      await api.addSeriesEpisode(selectedSeriesId, projectId);
      await Promise.all([refreshProject(), loadData()]);
      setMessage('当前项目已加入该剧集');
    } catch (error: any) {
      setMessage(error?.message || '加入剧集失败');
    } finally {
      setSubmitting(false);
    }
  };

  const removeEpisode = async (episodeProjectId: string) => {
    if (!selectedSeriesId) return;
    if (!window.confirm('确定从该剧集中移除此项目吗？项目本身不会被删除。')) return;
    setSubmitting(true);
    setMessage('');
    try {
      await api.removeSeriesEpisode(selectedSeriesId, episodeProjectId);
      await Promise.all([refreshProject(), loadData()]);
      setMessage('已从剧集中移除');
    } catch (error: any) {
      setMessage(error?.message || '移除失败');
    } finally {
      setSubmitting(false);
    }
  };

  if (!currentProject) {
    return <div className="h-64 rounded-2xl bg-slate-100 animate-pulse" />;
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">分集管理</h1>
          <p className="mt-1 text-sm text-slate-500">分集关系直接读取后端 Series 数据，刷新后仍保留。</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {message && <span className="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-600">{message}</span>}
          <button
            onClick={handleAnalyze}
            disabled={submitting}
            className="inline-flex items-center gap-2 rounded-xl border border-indigo-200 bg-indigo-50 px-4 py-2.5 text-sm font-bold text-indigo-700 transition hover:bg-indigo-100 disabled:opacity-60"
          >
            <Sparkles size={18} />
            {submitting ? '提交中...' : '解析当前剧本'}
          </button>
          <button
            onClick={() => setShowCreate(true)}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-indigo-600/20 transition hover:bg-indigo-500"
          >
            <Plus size={18} />
            新建剧集
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[340px_minmax(0,1fr)]">
        <aside className="space-y-6">
          <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
            <h2 className="mb-5 flex items-center gap-2 text-sm font-black uppercase tracking-widest text-slate-500">
              <Layers size={17} className="text-indigo-600" />
              剧集列表
            </h2>
            {loading ? (
              <div className="h-32 rounded-2xl bg-slate-100 animate-pulse" />
            ) : seriesList.length > 0 ? (
              <div className="space-y-3">
                {seriesList.map((series) => (
                  <button
                    key={series.id}
                    onClick={() => setSelectedSeriesId(series.id)}
                    className={`w-full rounded-2xl border px-4 py-3 text-left transition ${
                      selectedSeriesId === series.id
                        ? 'border-indigo-300 bg-indigo-50 text-indigo-900'
                        : 'border-slate-100 bg-slate-50 text-slate-700 hover:border-slate-200'
                    }`}
                  >
                    <div className="font-bold">{series.title}</div>
                    <div className="mt-1 text-xs text-slate-500">{series.episode_ids?.length || 0} 集</div>
                  </button>
                ))}
              </div>
            ) : (
              <p className="text-sm leading-6 text-slate-500">暂无剧集。先新建剧集，再把当前项目加入为一集。</p>
            )}
          </section>

          <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
            <h2 className="mb-3 text-sm font-black uppercase tracking-widest text-slate-500">当前项目</h2>
            <div className="rounded-2xl bg-slate-50 p-4">
              <div className="font-bold text-slate-900">{currentProject.title}</div>
              <div className="mt-2 text-xs leading-5 text-slate-500">
                {currentProject.series_id ? `已归属剧集，集数：${currentProject.episode_number || '-'}` : '未加入任何剧集'}
              </div>
            </div>
            <button
              onClick={attachCurrentProject}
              disabled={!selectedSeriesId || submitting}
              className="mt-4 w-full rounded-xl bg-slate-900 px-4 py-2.5 text-sm font-bold text-white transition hover:bg-slate-800 disabled:opacity-50"
            >
              加入选中剧集
            </button>
          </section>
        </aside>

        <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
          {selectedSeries ? (
            <div className="space-y-6">
              <div className="flex flex-col gap-4 border-b border-slate-100 pb-6 md:flex-row md:items-start md:justify-between">
                <div>
                  <h2 className="text-xl font-bold text-slate-900">{selectedSeries.title}</h2>
                  <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-500">{selectedSeries.description || '暂无剧集描述'}</p>
                </div>
                <button
                  onClick={loadData}
                  className="inline-flex items-center gap-2 rounded-xl border border-slate-200 px-4 py-2 text-sm font-bold text-slate-600 transition hover:bg-slate-50"
                >
                  <RefreshCcw size={16} />
                  刷新
                </button>
              </div>

              {selectedEpisodes.length > 0 ? (
                <div className="space-y-4">
                  {selectedEpisodes.map((episode, index) => (
                    <div key={episode.id} className="grid grid-cols-[72px_minmax(0,1fr)_auto] overflow-hidden rounded-2xl border border-slate-200 bg-white">
                      <div className="flex flex-col items-center justify-center border-r border-slate-100 bg-slate-50 py-5">
                        <span className="text-[10px] font-black uppercase tracking-widest text-slate-400">EP</span>
                        <span className="text-3xl font-black text-indigo-600">{String(index + 1).padStart(2, '0')}</span>
                      </div>
                      <button
                        onClick={() => navigate(`/projects/${episode.id}/script`)}
                        className="min-w-0 p-5 text-left transition hover:bg-slate-50"
                      >
                        <div className="flex flex-wrap items-center gap-3">
                          <h3 className="truncate text-base font-bold text-slate-900">{episode.title}</h3>
                          <span className="rounded-lg bg-slate-100 px-2 py-1 text-[10px] font-black uppercase tracking-widest text-slate-500">{episode.workflow}</span>
                        </div>
                        <p className="mt-2 line-clamp-2 text-sm leading-6 text-slate-500">{episode.script}</p>
                        <div className="mt-4 flex flex-wrap gap-3 text-xs font-semibold text-slate-400">
                          <span>角色 {episode.characters?.length || 0}</span>
                          <span>场景 {episode.scenes?.length || 0}</span>
                          <span>分镜 {episode.frames?.length || 0}</span>
                          <span>{formatDate(episode.updated_at)}</span>
                        </div>
                      </button>
                      <div className="flex items-center gap-2 border-l border-slate-100 px-4">
                        <button
                          onClick={() => navigate(`/projects/${episode.id}/storyboard`)}
                          className="rounded-xl border border-slate-200 p-2 text-slate-500 transition hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-600"
                          title="打开分镜"
                        >
                          <ArrowRight size={18} />
                        </button>
                        <button
                          onClick={() => removeEpisode(episode.id)}
                          className="rounded-xl border border-slate-200 p-2 text-slate-500 transition hover:border-red-200 hover:bg-red-50 hover:text-red-600"
                          title="移出剧集"
                        >
                          <Trash2 size={18} />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="rounded-3xl border-2 border-dashed border-slate-200 py-24 text-center">
                  <Film size={46} className="mx-auto text-slate-300" />
                  <h3 className="mt-4 text-lg font-bold text-slate-500">该剧集还没有关联项目</h3>
                  <p className="mt-2 text-sm text-slate-400">选择左侧剧集后，把当前项目加入为一集。</p>
                </div>
              )}
            </div>
          ) : (
            <div className="rounded-3xl border-2 border-dashed border-slate-200 py-24 text-center">
              <Layers size={46} className="mx-auto text-slate-300" />
              <h3 className="mt-4 text-lg font-bold text-slate-500">请选择或创建剧集</h3>
            </div>
          )}
        </section>
      </div>

      {showCreate && (
        <CreateSeriesModal
          defaultWorkflow={currentProject.workflow || 'r2v'}
          onClose={() => setShowCreate(false)}
          onCreated={async (series) => {
            setShowCreate(false);
            setSelectedSeriesId(series.id);
            await loadData();
          }}
        />
      )}
    </div>
  );
}

function CreateSeriesModal({ defaultWorkflow, onClose, onCreated }: { defaultWorkflow: string; onClose: () => void; onCreated: (series: Series) => void }) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [workflow, setWorkflow] = useState(defaultWorkflow);
  const [loading, setLoading] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setLoading(true);
    try {
      const series = await api.createSeries({ title, description, workflow });
      onCreated(series);
    } catch (error: any) {
      alert(error?.message || '创建剧集失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-slate-900/60 backdrop-blur-sm" onClick={onClose} />
      <form onSubmit={submit} className="relative w-full max-w-xl rounded-3xl bg-white p-8 shadow-2xl">
        <h2 className="text-2xl font-bold text-slate-900">新建剧集</h2>
        <div className="mt-8 space-y-5">
          <label className="block space-y-2">
            <span className="text-sm font-bold text-slate-700">剧集名称</span>
            <input required value={title} onChange={(event) => setTitle(event.target.value)} className="w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100" />
          </label>
          <label className="block space-y-2">
            <span className="text-sm font-bold text-slate-700">描述</span>
            <textarea value={description} onChange={(event) => setDescription(event.target.value)} className="h-28 w-full resize-none rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100" />
          </label>
          <label className="block space-y-2">
            <span className="text-sm font-bold text-slate-700">工作流</span>
            <select value={workflow} onChange={(event) => setWorkflow(event.target.value)} className="w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 font-bold outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100">
              <option value="r2v">参考生图 + 视频</option>
              <option value="i2v">图生视频</option>
              <option value="t2v">文生视频</option>
              <option value="novel2script">小说改编</option>
            </select>
          </label>
        </div>
        <div className="mt-8 flex justify-end gap-3">
          <button type="button" onClick={onClose} className="rounded-xl border border-slate-200 px-5 py-2.5 text-sm font-bold text-slate-600 hover:bg-slate-50">取消</button>
          <button disabled={loading} className="rounded-xl bg-indigo-600 px-6 py-2.5 text-sm font-bold text-white hover:bg-indigo-500 disabled:opacity-60">{loading ? '创建中...' : '创建'}</button>
        </div>
      </form>
    </div>
  );
}
