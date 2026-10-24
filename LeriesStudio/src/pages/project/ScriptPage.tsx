/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { BookOpen, Boxes, Clapperboard, FileText, MapPin, RefreshCcw, Save, Sparkles, Users } from 'lucide-react';
import { useProject } from '../../contexts/ProjectContext';
import { api } from '../../lib/api';
import { cn } from '../../lib/utils';

const WORKFLOWS = [
  { value: 'r2v', label: '参考生图 + 视频' },
  { value: 'i2v', label: '图生视频' },
  { value: 't2v', label: '文生视频' },
  { value: 'novel2script', label: '小说改编' },
];

export default function ScriptPage() {
  const { projectId } = useParams();
  const { currentProject, setCurrentProjectId, refreshProject, refreshTasks, tasks } = useProject();
  const [title, setTitle] = useState('');
  const [script, setScript] = useState('');
  const [workflow, setWorkflow] = useState('r2v');
  const [saving, setSaving] = useState(false);
  const [submitting, setSubmitting] = useState<'analysis' | 'reparse' | null>(null);
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (projectId) setCurrentProjectId(projectId);
  }, [projectId, setCurrentProjectId]);

  useEffect(() => {
    if (!currentProject) return;
    setTitle(currentProject.title);
    setScript(currentProject.script);
    setWorkflow(currentProject.workflow || 'r2v');
  }, [currentProject]);

  const stats = useMemo(() => {
    const lines = script.split('\n').filter((line) => line.trim()).length;
    return {
      characters: script.length,
      lines,
      frames: currentProject?.frames?.length || 0,
      assets: (currentProject?.characters?.length || 0) + (currentProject?.scenes?.length || 0) + (currentProject?.props?.length || 0),
    };
  }, [currentProject, script]);

  const latestAnalysisTask = useMemo(
    () => tasks.find((task) => task.type === 'analysis'),
    [tasks],
  );

  const saveProject = async () => {
    if (!projectId) return;
    setSaving(true);
    setMessage('');
    try {
      await api.updateProject(projectId, { title, script, workflow });
      await refreshProject();
      setMessage('剧本已保存到后端');
    } catch (error: any) {
      setMessage(error?.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const submitAnalysis = async (type: 'analysis' | 'reparse') => {
    if (!projectId) return;
    setSubmitting(type);
    setMessage('');
    try {
      await saveProject();
      if (type === 'analysis') {
        await api.analyzeProject(projectId);
      } else {
        await api.reparseProject(projectId);
      }
      await refreshTasks();
      await refreshProject();
      setMessage(type === 'analysis' ? '剧本分析任务已提交' : '重新解析任务已提交');
    } catch (error: any) {
      setMessage(error?.message || '任务提交失败');
    } finally {
      setSubmitting(null);
    }
  };

  if (!currentProject) {
    return <div className="h-64 rounded-2xl bg-slate-100 animate-pulse" />;
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">剧本原文</h1>
          <p className="mt-1 text-sm text-slate-500">编辑、保存并触发后端剧本解析任务，解析结果会回写到项目资产和分镜。</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {message && <span className="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-600">{message}</span>}
          <button
            onClick={saveProject}
            disabled={saving}
            className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-bold text-slate-700 transition hover:bg-slate-50 disabled:opacity-60"
          >
            <Save size={18} />
            {saving ? '保存中...' : '保存原文'}
          </button>
          <button
            onClick={() => submitAnalysis('reparse')}
            disabled={Boolean(submitting)}
            className="inline-flex items-center gap-2 rounded-xl border border-indigo-200 bg-indigo-50 px-4 py-2.5 text-sm font-bold text-indigo-700 transition hover:bg-indigo-100 disabled:opacity-60"
          >
            <RefreshCcw size={18} />
            {submitting === 'reparse' ? '提交中...' : '重新解析'}
          </button>
          <button
            onClick={() => submitAnalysis('analysis')}
            disabled={Boolean(submitting)}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-indigo-600/20 transition hover:bg-indigo-500 disabled:opacity-60"
          >
            <Sparkles size={18} />
            {submitting === 'analysis' ? '提交中...' : 'AI 剧本分析'}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
        <section className="rounded-3xl border border-slate-200 bg-white shadow-soft">
          <div className="border-b border-slate-100 p-6">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-[minmax(0,1fr)_220px]">
              <label className="space-y-2">
                <span className="block text-xs font-black uppercase tracking-widest text-slate-400">项目标题</span>
                <input
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  className="w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-base font-bold text-slate-900 outline-none transition focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
                />
              </label>
              <label className="space-y-2">
                <span className="block text-xs font-black uppercase tracking-widest text-slate-400">工作流</span>
                <select
                  value={workflow}
                  onChange={(event) => setWorkflow(event.target.value)}
                  className="w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm font-bold text-slate-700 outline-none transition focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
                >
                  {WORKFLOWS.map((item) => (
                    <option key={item.value} value={item.value}>{item.label}</option>
                  ))}
                </select>
              </label>
            </div>
          </div>
          <div className="p-6">
            <label className="mb-3 flex items-center gap-2 text-xs font-black uppercase tracking-widest text-slate-400">
              <FileText size={14} />
              剧本正文
            </label>
            <textarea
              value={script}
              onChange={(event) => setScript(event.target.value)}
              className="min-h-[620px] w-full resize-y rounded-2xl border border-slate-200 bg-slate-50 px-5 py-4 text-sm font-medium leading-7 text-slate-800 outline-none transition focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100"
              placeholder="输入或粘贴剧本原文..."
            />
          </div>
        </section>

        <aside className="space-y-6">
          <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
            <h2 className="mb-5 flex items-center gap-2 text-sm font-black uppercase tracking-widest text-slate-500">
              <BookOpen size={17} className="text-indigo-600" />
              当前数据
            </h2>
            <div className="grid grid-cols-2 gap-3">
              <StatCard label="字符数" value={stats.characters} />
              <StatCard label="有效行" value={stats.lines} />
              <StatCard label="资产数" value={stats.assets} />
              <StatCard label="分镜数" value={stats.frames} />
            </div>
          </section>

          <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
            <h2 className="mb-5 text-sm font-black uppercase tracking-widest text-slate-500">解析状态</h2>
            {latestAnalysisTask ? (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-bold text-slate-700">{latestAnalysisTask.status}</span>
                  <span className="text-sm font-black text-indigo-600">{latestAnalysisTask.progress}%</span>
                </div>
                <div className="h-2 overflow-hidden rounded-full bg-slate-100">
                  <div className="h-full bg-indigo-600 transition-all" style={{ width: `${Math.max(latestAnalysisTask.progress, latestAnalysisTask.status === 'pending' ? 8 : 0)}%` }} />
                </div>
                {latestAnalysisTask.error && <p className="rounded-xl bg-red-50 p-3 text-xs font-semibold leading-5 text-red-600">{latestAnalysisTask.error}</p>}
              </div>
            ) : (
              <p className="text-sm leading-6 text-slate-500">暂无剧本解析任务。点击 AI 剧本分析后，后端会创建任务并写入 MongoDB 队列。</p>
            )}
          </section>

          <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
            <h2 className="mb-5 text-sm font-black uppercase tracking-widest text-slate-500">解析回写</h2>
            <div className="space-y-3">
              <SummaryRow icon={Users} label="角色" value={currentProject.characters?.length || 0} />
              <SummaryRow icon={MapPin} label="场景" value={currentProject.scenes?.length || 0} />
              <SummaryRow icon={Boxes} label="道具" value={currentProject.props?.length || 0} />
              <SummaryRow icon={Clapperboard} label="分镜" value={currentProject.frames?.length || 0} />
            </div>
          </section>

          {currentProject.analysis?.synopsis && (
            <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
              <h2 className="mb-3 text-sm font-black uppercase tracking-widest text-slate-500">故事摘要</h2>
              <p className="text-sm leading-7 text-slate-600">{currentProject.analysis.synopsis}</p>
            </section>
          )}
        </aside>
      </div>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-2xl border border-slate-100 bg-slate-50 p-4">
      <div className="text-[10px] font-black uppercase tracking-widest text-slate-400">{label}</div>
      <div className="mt-2 text-2xl font-black text-slate-900">{value}</div>
    </div>
  );
}

function SummaryRow({ icon: Icon, label, value }: { icon: any; label: string; value: number }) {
  return (
    <div className={cn('flex items-center justify-between rounded-2xl border border-slate-100 bg-slate-50 px-4 py-3')}>
      <div className="flex items-center gap-3 text-sm font-bold text-slate-700">
        <Icon size={17} className="text-indigo-600" />
        {label}
      </div>
      <span className="text-sm font-black text-slate-900">{value}</span>
    </div>
  );
}
