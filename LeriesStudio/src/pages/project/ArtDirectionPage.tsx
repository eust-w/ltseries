/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { AlertCircle, CheckCircle2, Palette, Save, Sparkles } from 'lucide-react';
import { useProject } from '../../contexts/ProjectContext';
import { api } from '../../lib/api';
import { cn } from '../../lib/utils';

export default function ArtDirectionPage() {
  const { projectId } = useParams();
  const { currentProject, setCurrentProjectId, refreshProject, refreshTasks } = useProject();
  const [presets, setPresets] = useState<any[]>([]);
  const [selectedStyleId, setSelectedStyleId] = useState('');
  const [positivePrompt, setPositivePrompt] = useState('');
  const [negativePrompt, setNegativePrompt] = useState('');
  const [palette, setPalette] = useState('');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    if (projectId) setCurrentProjectId(projectId);
  }, [projectId, setCurrentProjectId]);

  useEffect(() => {
    api.getStylePresets()
      .then((response) => setPresets(response.presets || []))
      .catch((error) => setMessage({ type: 'error', text: error?.message || '读取风格预设失败' }));
  }, []);

  useEffect(() => {
    if (!currentProject?.art_direction) return;
    const config = currentProject.art_direction.style_config || {};
    setSelectedStyleId(currentProject.art_direction.selected_style_id || '');
    setPositivePrompt(String(config.positive_prompt || ''));
    setNegativePrompt(String(config.negative_prompt || ''));
    setPalette(String(config.palette || ''));
  }, [currentProject]);

  const recommendations = useMemo(
    () => currentProject?.art_direction?.ai_recommendations || [],
    [currentProject],
  );

  const applyPreset = (preset: any) => {
    setSelectedStyleId(preset.id || '');
    if (preset.positive_prompt) setPositivePrompt(preset.positive_prompt);
    if (preset.negative_prompt) setNegativePrompt(preset.negative_prompt);
  };

  const saveArtDirection = async () => {
    if (!projectId) return;
    setLoading(true);
    setMessage(null);
    try {
      await api.saveArtDirection(projectId, {
        selected_style_id: selectedStyleId,
        style_config: {
          positive_prompt: positivePrompt,
          negative_prompt: negativePrompt,
          palette,
        },
        custom_styles: currentProject?.art_direction?.custom_styles || [],
        ai_recommendations: currentProject?.art_direction?.ai_recommendations || [],
      });
      await refreshProject();
      setMessage({ type: 'success', text: '美术方向已保存到后端' });
    } catch (error: any) {
      setMessage({ type: 'error', text: error?.message || '保存失败' });
    } finally {
      setLoading(false);
    }
  };

  const analyzeArtDirection = async () => {
    if (!projectId) return;
    setLoading(true);
    setMessage(null);
    try {
      await api.analyzeArtDirection(projectId);
      await refreshTasks();
      setMessage({ type: 'success', text: '美术分析任务已提交' });
    } catch (error: any) {
      setMessage({ type: 'error', text: error?.message || '任务提交失败' });
    } finally {
      setLoading(false);
    }
  };

  if (!currentProject) {
    return <div className="h-64 rounded-2xl bg-slate-100 animate-pulse" />;
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">美术方向</h1>
          <p className="mt-1 text-sm text-slate-500">风格预设、AI 推荐和项目美术提示词均来自后端数据。</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {message && (
            <span className={cn('inline-flex items-center gap-2 rounded-xl px-4 py-2 text-sm font-bold', message.type === 'success' ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700')}>
              {message.type === 'success' ? <CheckCircle2 size={16} /> : <AlertCircle size={16} />}
              {message.text}
            </span>
          )}
          <button
            onClick={analyzeArtDirection}
            disabled={loading}
            className="inline-flex items-center gap-2 rounded-xl border border-indigo-200 bg-indigo-50 px-4 py-2.5 text-sm font-bold text-indigo-700 transition hover:bg-indigo-100 disabled:opacity-60"
          >
            <Sparkles size={18} />
            AI 推荐
          </button>
          <button
            onClick={saveArtDirection}
            disabled={loading}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-indigo-600/20 transition hover:bg-indigo-500 disabled:opacity-60"
          >
            <Save size={18} />
            {loading ? '处理中...' : '保存配置'}
          </button>
        </div>
      </div>

      <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
        <h2 className="mb-5 flex items-center gap-2 text-sm font-black uppercase tracking-widest text-slate-500">
          <Palette size={17} className="text-indigo-600" />
          后端风格预设
        </h2>
        {presets.length > 0 ? (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            {presets.map((preset) => (
              <button
                key={preset.id}
                onClick={() => applyPreset(preset)}
                className={cn(
                  'rounded-2xl border p-5 text-left transition',
                  selectedStyleId === preset.id
                    ? 'border-indigo-300 bg-indigo-50 shadow-sm'
                    : 'border-slate-200 bg-slate-50 hover:border-slate-300',
                )}
              >
                <div className="text-base font-black text-slate-900">{preset.name || preset.id}</div>
                <p className="mt-2 line-clamp-3 text-sm leading-6 text-slate-500">{preset.description || preset.positive_prompt || '后端未提供描述'}</p>
              </button>
            ))}
          </div>
        ) : (
          <div className="rounded-2xl border border-dashed border-slate-200 py-12 text-center text-sm font-semibold text-slate-400">暂无后端预设</div>
        )}
      </section>

      {recommendations.length > 0 && (
        <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
          <h2 className="mb-5 text-sm font-black uppercase tracking-widest text-slate-500">AI 分析推荐</h2>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            {recommendations.map((item: any, index: number) => (
              <button
                key={item.id || index}
                onClick={() => {
                  setSelectedStyleId(item.id || selectedStyleId);
                  if (item.positive_prompt) setPositivePrompt(item.positive_prompt);
                  if (item.negative_prompt) setNegativePrompt(item.negative_prompt);
                }}
                className="rounded-2xl border border-slate-200 bg-slate-50 p-5 text-left transition hover:border-indigo-300 hover:bg-indigo-50"
              >
                <div className="font-black text-slate-900">{item.name || `推荐 ${index + 1}`}</div>
                <p className="mt-2 text-sm leading-6 text-slate-500">{item.description || item.positive_prompt || '后端未提供推荐说明'}</p>
              </button>
            ))}
          </div>
        </section>
      )}

      <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <label className="space-y-3">
            <span className="block text-sm font-bold text-slate-700">正向风格 Prompt</span>
            <textarea
              value={positivePrompt}
              onChange={(event) => setPositivePrompt(event.target.value)}
              className="h-64 w-full resize-y rounded-2xl border border-slate-200 bg-slate-50 px-5 py-4 text-sm font-medium leading-7 outline-none transition focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100"
              placeholder="保存后会写入项目 art_direction.style_config.positive_prompt"
            />
          </label>
          <label className="space-y-3">
            <span className="block text-sm font-bold text-slate-700">负向风格 Prompt</span>
            <textarea
              value={negativePrompt}
              onChange={(event) => setNegativePrompt(event.target.value)}
              className="h-64 w-full resize-y rounded-2xl border border-slate-200 bg-slate-50 px-5 py-4 text-sm font-medium leading-7 outline-none transition focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100"
              placeholder="保存后会写入项目 art_direction.style_config.negative_prompt"
            />
          </label>
        </div>
        <label className="mt-6 block space-y-3">
          <span className="block text-sm font-bold text-slate-700">色彩与摄影备注</span>
          <textarea
            value={palette}
            onChange={(event) => setPalette(event.target.value)}
            className="h-28 w-full resize-y rounded-2xl border border-slate-200 bg-slate-50 px-5 py-4 text-sm font-medium leading-7 outline-none transition focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100"
            placeholder="例如色彩、光线、镜头语言等。"
          />
        </label>
      </section>
    </div>
  );
}
