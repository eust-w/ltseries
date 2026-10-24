/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { CheckCircle2, Cpu, Save, Type } from 'lucide-react';
import { useProject } from '../../contexts/ProjectContext';
import { api } from '../../lib/api';

export default function ProjectSettingsPage() {
  const { projectId } = useParams();
  const { currentProject, setCurrentProjectId, refreshProject } = useProject();
  const [modelSettings, setModelSettings] = useState<Record<string, string>>({});
  const [promptConfig, setPromptConfig] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (projectId) setCurrentProjectId(projectId);
  }, [projectId, setCurrentProjectId]);

  useEffect(() => {
    if (!currentProject) return;
    setModelSettings({
      t2i_model: currentProject.model_settings?.t2i_model || '',
      i2i_model: currentProject.model_settings?.i2i_model || '',
      image_model: currentProject.model_settings?.image_model || '',
      i2v_model: currentProject.model_settings?.i2v_model || '',
      character_aspect_ratio: currentProject.model_settings?.character_aspect_ratio || '',
      scene_aspect_ratio: currentProject.model_settings?.scene_aspect_ratio || '',
      prop_aspect_ratio: currentProject.model_settings?.prop_aspect_ratio || '',
      storyboard_aspect_ratio: currentProject.model_settings?.storyboard_aspect_ratio || '',
    });
    setPromptConfig({
      storyboard_polish: currentProject.prompt_config?.storyboard_polish || '',
      video_polish: currentProject.prompt_config?.video_polish || '',
      r2v_polish: currentProject.prompt_config?.r2v_polish || '',
    });
  }, [currentProject]);

  const updateModel = (key: string, value: string) => {
    setModelSettings((current) => ({ ...current, [key]: value }));
  };

  const updatePrompt = (key: string, value: string) => {
    setPromptConfig((current) => ({ ...current, [key]: value }));
  };

  const handleSave = async () => {
    if (!projectId) return;
    setLoading(true);
    setMessage('');
    try {
      await api.updateModelSettings(projectId, modelSettings);
      await api.updatePromptConfig(projectId, promptConfig);
      await refreshProject();
      setMessage('项目设置已保存到后端');
    } catch (error: any) {
      alert(error?.message || '保存失败');
    } finally {
      setLoading(false);
    }
  };

  if (!currentProject) {
    return <div className="h-64 rounded-2xl bg-slate-100 animate-pulse" />;
  }

  return (
    <div className="max-w-5xl space-y-8">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">项目设置</h1>
          <p className="mt-1 text-sm text-slate-500">字段与后端 ModelSettings、PromptConfig 一一对应。</p>
        </div>
        <div className="flex items-center gap-3">
          {message && (
            <span className="inline-flex items-center gap-2 rounded-xl bg-emerald-50 px-4 py-2 text-sm font-bold text-emerald-700">
              <CheckCircle2 size={17} />
              {message}
            </span>
          )}
          <button
            onClick={handleSave}
            disabled={loading}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-6 py-3 text-sm font-bold text-white shadow-lg shadow-indigo-600/20 transition hover:bg-indigo-500 disabled:opacity-60"
          >
            <Save size={18} />
            {loading ? '保存中...' : '保存设置'}
          </button>
        </div>
      </div>

      <section className="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-soft">
        <div className="flex items-center gap-3 border-b border-slate-100 bg-slate-50 px-6 py-5">
          <Cpu className="text-indigo-600" size={20} />
          <h2 className="font-bold text-slate-900">模型选型</h2>
        </div>
        <div className="grid grid-cols-1 gap-6 p-6 md:grid-cols-2">
          <Field label="文生图模型 t2i_model" value={modelSettings.t2i_model || ''} onChange={(value) => updateModel('t2i_model', value)} />
          <Field label="图生图模型 i2i_model" value={modelSettings.i2i_model || ''} onChange={(value) => updateModel('i2i_model', value)} />
          <Field label="资产默认图片模型 image_model" value={modelSettings.image_model || ''} onChange={(value) => updateModel('image_model', value)} />
          <Field label="图生视频模型 i2v_model" value={modelSettings.i2v_model || ''} onChange={(value) => updateModel('i2v_model', value)} />
        </div>
      </section>

      <section className="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-soft">
        <div className="flex items-center gap-3 border-b border-slate-100 bg-slate-50 px-6 py-5">
          <Cpu className="text-indigo-600" size={20} />
          <h2 className="font-bold text-slate-900">画幅配置</h2>
        </div>
        <div className="grid grid-cols-1 gap-6 p-6 md:grid-cols-4">
          <Field label="角色画幅" value={modelSettings.character_aspect_ratio || ''} onChange={(value) => updateModel('character_aspect_ratio', value)} />
          <Field label="场景画幅" value={modelSettings.scene_aspect_ratio || ''} onChange={(value) => updateModel('scene_aspect_ratio', value)} />
          <Field label="道具画幅" value={modelSettings.prop_aspect_ratio || ''} onChange={(value) => updateModel('prop_aspect_ratio', value)} />
          <Field label="分镜画幅" value={modelSettings.storyboard_aspect_ratio || ''} onChange={(value) => updateModel('storyboard_aspect_ratio', value)} />
        </div>
      </section>

      <section className="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-soft">
        <div className="flex items-center gap-3 border-b border-slate-100 bg-slate-50 px-6 py-5">
          <Type className="text-indigo-600" size={20} />
          <h2 className="font-bold text-slate-900">提示词修饰</h2>
        </div>
        <div className="space-y-6 p-6">
          <TextArea label="分镜提示词 storyboard_polish" value={promptConfig.storyboard_polish || ''} onChange={(value) => updatePrompt('storyboard_polish', value)} />
          <TextArea label="视频提示词 video_polish" value={promptConfig.video_polish || ''} onChange={(value) => updatePrompt('video_polish', value)} />
          <TextArea label="参考生视频提示词 r2v_polish" value={promptConfig.r2v_polish || ''} onChange={(value) => updatePrompt('r2v_polish', value)} />
        </div>
      </section>
    </div>
  );
}

function Field({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="block space-y-2">
      <span className="block text-xs font-black uppercase tracking-widest text-slate-400">{label}</span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-xs outline-none transition focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100"
      />
    </label>
  );
}

function TextArea({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="block space-y-2">
      <span className="block text-xs font-black uppercase tracking-widest text-slate-400">{label}</span>
      <textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="h-28 w-full resize-y rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm leading-6 outline-none transition focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100"
      />
    </label>
  );
}
