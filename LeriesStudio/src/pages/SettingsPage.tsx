/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { FormEvent, useEffect, useState } from 'react';
import { api } from '../lib/api';
import { RuntimeConfig } from '../types';
import { Save, Key, Globe, Layout, CheckCircle2 } from 'lucide-react';

export default function SettingsPage() {
  const [config, setConfig] = useState<Partial<RuntimeConfig>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    api.getConfig().then(res => {
      setConfig(res);
      setLoading(false);
    });
  }, []);

  const handleSave = async (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      const saved = await api.updateConfig(config);
      setConfig(saved);
      setMessage('设置已成功保存');
    } catch (err) {
      alert('保存失败');
    } finally {
      setSaving(false);
    }
  };

  const updateField = (key: keyof RuntimeConfig, value: string) => {
    setConfig(prev => ({ ...prev, [key]: value }));
  };

  if (loading) return <div className="animate-pulse h-64 bg-slate-100 rounded-3xl" />;

  return (
    <div className="max-w-4xl space-y-8">
      <div>
        <h1 className="text-3xl font-bold text-slate-900">系统设置</h1>
        <p className="text-slate-500 mt-2">配置全局 AI 服务商、模型及系统偏好</p>
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        {/* Provider Config */}
        <section className="bg-white rounded-3xl border border-slate-200 shadow-soft overflow-hidden">
          <div className="p-6 border-b border-slate-50 bg-slate-50/50 flex items-center gap-3">
             <Globe className="text-indigo-600" size={20} />
             <h3 className="font-bold text-slate-900">核心引擎 (Provider)</h3>
          </div>
          <div className="p-8 grid grid-cols-1 md:grid-cols-2 gap-8">
             <div className="space-y-4">
               <div>
                  <label className="block text-sm font-bold text-slate-700 mb-2">默认 AI 服务商</label>
                  <select
                    value={config.ai_provider}
                    onChange={e => updateField('ai_provider', e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 outline-none focus:ring-2 focus:ring-indigo-600/10 transition-all"
                  >
                    <option value="dashscope">阿里云百炼 (DashScope)</option>
                    <option value="openai">OpenAI Compatible</option>
                  </select>
               </div>
               <div>
                  <label className="block text-sm font-bold text-slate-700 mb-2">DashScope API Key</label>
                  <div className="relative">
                    <Key className="absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" size={16} />
                    <input
                      type="password"
                      value={config.dashscope_api_key || ''}
                      onChange={e => updateField('dashscope_api_key', e.target.value)}
                      placeholder={config.dashscope_api_key_set ? "••••••••••••••••" : "尚未设置"}
                      className="w-full bg-slate-50 border border-slate-200 rounded-xl pl-11 pr-4 py-2.5 outline-none focus:ring-2 focus:ring-indigo-600/10 transition-all"
                    />
                  </div>
                  <p className="text-[10px] text-slate-400 mt-2 ml-1">为了您的安全，已保存的密钥将不会在此明文显示。</p>
               </div>
             </div>

             <div className="space-y-4">
                <div>
                  <label className="block text-sm font-bold text-slate-700 mb-2">LLM 模型节点</label>
                  <input
                    type="text"
                    value={config.llm_model}
                    onChange={e => updateField('llm_model', e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 outline-none focus:ring-2 focus:ring-indigo-600/10 transition-all font-mono text-sm"
                  />
                </div>
                 <div>
                  <label className="block text-sm font-bold text-slate-700 mb-2">LLM Base URL</label>
                  <input
                    type="text"
                    value={config.llm_base_url}
                    onChange={e => updateField('llm_base_url', e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 outline-none focus:ring-2 focus:ring-indigo-600/10 transition-all font-mono text-sm"
                  />
                </div>
             </div>
          </div>
        </section>

        {/* Defaults */}
        <section className="bg-white rounded-3xl border border-slate-200 shadow-soft overflow-hidden">
          <div className="p-6 border-b border-slate-50 bg-slate-50/50 flex items-center gap-3">
             <Layout className="text-indigo-600" size={20} />
             <h3 className="font-bold text-slate-900">默认模型预设</h3>
          </div>
          <div className="p-8 grid grid-cols-1 md:grid-cols-2 gap-8">
             <div>
                <label className="block text-sm font-bold text-slate-700 mb-2">默认图片生成模型 (General)</label>
                <input
                  type="text"
                  value={config.default_image_model}
                  onChange={e => updateField('default_image_model', e.target.value)}
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 outline-none focus:ring-2 focus:ring-indigo-600/10 transition-all font-mono text-sm"
                />
             </div>
             <div>
                <label className="block text-sm font-bold text-slate-700 mb-2">默认视频生成模型 (Generative Video)</label>
                <input
                  type="text"
                  value={config.default_video_model}
                  onChange={e => updateField('default_video_model', e.target.value)}
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 outline-none focus:ring-2 focus:ring-indigo-600/10 transition-all font-mono text-sm"
                />
             </div>
          </div>
        </section>

        <div className="flex items-center justify-between pt-4">
           {message && (
             <div className="flex items-center gap-2 text-emerald-600 font-bold animate-in fade-in slide-in-from-left-2 transition-all">
                <CheckCircle2 size={20} />
                {message}
             </div>
           )}
           <button
              type="submit"
              disabled={saving}
              className="ml-auto flex items-center gap-2 bg-indigo-600 hover:bg-indigo-500 text-white px-8 py-3 rounded-xl font-bold shadow-xl shadow-indigo-600/20 active:scale-95 transition-all disabled:opacity-50"
            >
              <Save size={20} />
              {saving ? '正在保存...' : '确认并保存'}
           </button>
        </div>
      </form>
    </div>
  );
}
