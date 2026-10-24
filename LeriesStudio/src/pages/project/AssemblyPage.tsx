/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { AlertCircle, CheckCircle2, Download, Film, RefreshCw, Scissors, ShieldCheck, Sparkles, Video } from 'lucide-react';
import { useProject } from '../../contexts/ProjectContext';
import { api, mediaUrl } from '../../lib/api';
import { cn } from '../../lib/utils';
import { ProductionAuditReport } from '../../types';

export default function AssemblyPage() {
  const { projectId } = useParams();
  const { currentProject, setCurrentProjectId, refreshProject, refreshTasks, tasks } = useProject();
  const [submitting, setSubmitting] = useState<'pipeline' | 'regenerate' | 'merge' | 'export' | null>(null);
  const [exportUrl, setExportUrl] = useState<string | null>(null);
  const [message, setMessage] = useState('');
  const [audit, setAudit] = useState<ProductionAuditReport | null>(null);

  useEffect(() => {
    if (projectId) setCurrentProjectId(projectId);
  }, [projectId, setCurrentProjectId]);

  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    api.getProductionAudit(projectId)
      .then((report) => {
        if (!cancelled) setAudit(report);
      })
      .catch(() => {
        if (!cancelled) setAudit(null);
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, currentProject?.updated_at, tasks.length]);

  const refreshAll = async () => {
    if (!projectId) return;
    await refreshProject();
    await refreshTasks();
    try {
      setAudit(await api.getProductionAudit(projectId));
    } catch {
      setAudit(null);
    }
  };

  const frames = useMemo(
    () => [...(currentProject?.frames || [])].sort((a, b) => a.index - b.index),
    [currentProject],
  );
  const readyFrames = frames.filter((frame) => frame.video_url);
  const mergeTasks = tasks.filter((task) => task.type === 'merge');
  const pipelineTasks = tasks.filter((task) => task.type === 'storyboard_pipeline');

  const completeStoryboard = async (regenerate = false) => {
    if (!projectId) return;
    setSubmitting(regenerate ? 'regenerate' : 'pipeline');
    setMessage('');
    try {
      await api.completeStoryboard(projectId, {
        model: currentProject?.model_settings?.i2v_model,
        duration: 5,
        regenerate_images: regenerate,
        regenerate_videos: regenerate,
        parameters: {
          ratio: currentProject?.model_settings?.storyboard_aspect_ratio || '9:16',
          audio: false,
          watermark: false,
        },
      });
      await refreshTasks();
      setAudit(await api.getProductionAudit(projectId));
      setMessage(regenerate ? '一致性重生成任务已提交：会按资产参考重做全部镜头并拼接' : '一键补全任务已提交：会逐镜头补图、生成视频并拼接成片');
    } catch (error: any) {
      setMessage(error?.message || '一键补全任务启动失败');
    } finally {
      setSubmitting(null);
    }
  };

  const startMerge = async () => {
    if (!projectId) return;
    setSubmitting('merge');
    setMessage('');
    try {
      await api.mergeProject(projectId);
      await refreshTasks();
      setAudit(await api.getProductionAudit(projectId));
      setMessage('视频拼接任务已提交');
    } catch (error: any) {
      setMessage(error?.message || '拼接任务启动失败');
    } finally {
      setSubmitting(null);
    }
  };

  const exportProject = async () => {
    if (!projectId) return;
    setSubmitting('export');
    setMessage('');
    try {
      const response = await api.exportProject(projectId);
      if (response.url) {
        setExportUrl(response.url);
        setMessage('成片地址已返回');
      } else {
        await refreshTasks();
        setMessage('导出任务已提交');
      }
    } catch (error: any) {
      setMessage(error?.message || '导出失败');
    } finally {
      setSubmitting(null);
    }
  };

  if (!currentProject) {
    return <div className="h-64 rounded-2xl bg-slate-100 animate-pulse" />;
  }

  const previewUrl = exportUrl || currentProject.merged_video_url;

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">成片管理</h1>
          <p className="mt-1 text-sm text-slate-500">按后端已有分镜视频拼接成片，结果写入项目 merged_video_url。</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {message && <span className="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-600">{message}</span>}
          <button
            onClick={async () => {
              await refreshAll();
            }}
            className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-bold text-slate-700 transition hover:bg-slate-50"
          >
            <RefreshCw size={18} />
            刷新
          </button>
          <button
            onClick={startMerge}
            disabled={Boolean(submitting) || readyFrames.length === 0}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-indigo-600/20 transition hover:bg-indigo-500 disabled:opacity-60"
          >
            <Scissors size={18} />
            {submitting === 'merge' ? '提交中...' : '拼接视频'}
          </button>
          <button
            onClick={() => completeStoryboard(false)}
            disabled={Boolean(submitting) || frames.length === 0}
            className="inline-flex items-center gap-2 rounded-xl bg-slate-900 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-slate-900/10 transition hover:bg-slate-800 disabled:opacity-60"
          >
            <Sparkles size={18} />
            {submitting === 'pipeline' ? '提交中...' : '一键补全并拼接'}
          </button>
          <button
            onClick={() => completeStoryboard(true)}
            disabled={Boolean(submitting) || frames.length === 0}
            className="inline-flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm font-bold text-amber-800 transition hover:bg-amber-100 disabled:opacity-60"
          >
            <Sparkles size={18} />
            {submitting === 'regenerate' ? '提交中...' : '按资产重生成全片'}
          </button>
          <button
            onClick={exportProject}
            disabled={Boolean(submitting)}
            className="inline-flex items-center gap-2 rounded-xl border border-indigo-200 bg-indigo-50 px-4 py-2.5 text-sm font-bold text-indigo-700 transition hover:bg-indigo-100 disabled:opacity-60"
          >
            <Download size={18} />
            {submitting === 'export' ? '提交中...' : '导出成片'}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-[minmax(0,1fr)_340px]">
        <main className="space-y-6">
          {audit && <ProductionAuditPanel audit={audit} />}

          <section className="aspect-video overflow-hidden rounded-3xl border border-slate-800 bg-slate-950 shadow-2xl">
            {previewUrl ? (
              <video src={mediaUrl(previewUrl)} controls className="h-full w-full object-contain" />
            ) : (
              <div className="flex h-full flex-col items-center justify-center gap-4 text-slate-500">
                <Film size={52} />
                <div className="text-center">
                  <p className="font-bold text-slate-400">尚未生成成片</p>
                  <p className="mt-1 text-xs text-slate-600">先生成分镜视频，再提交拼接任务。</p>
                </div>
              </div>
            )}
          </section>

          {previewUrl && (
            <section className="flex flex-col gap-3 rounded-2xl border border-emerald-200 bg-emerald-50 p-4 md:flex-row md:items-center md:justify-between">
              <div className="flex items-center gap-3">
                <CheckCircle2 size={24} className="text-emerald-600" />
                <div>
                  <h3 className="font-bold text-emerald-900">成片可预览</h3>
                  <p className="text-xs text-emerald-700">该地址由后端返回，可直接下载。</p>
                </div>
              </div>
              <a href={mediaUrl(previewUrl)} download className="inline-flex items-center justify-center gap-2 rounded-xl bg-emerald-600 px-4 py-2 text-sm font-bold text-white">
                <Download size={17} />
                下载
              </a>
            </section>
          )}

          <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
            <h2 className="mb-5 flex items-center gap-2 text-sm font-black uppercase tracking-widest text-slate-500">
              <Video size={17} className="text-indigo-600" />
              拼接镜头列表 ({readyFrames.length} / {frames.length})
            </h2>
            {frames.length > 0 ? (
              <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
                {frames.map((frame) => (
                  <div key={frame.id} className="space-y-2">
                    <div className={cn('relative aspect-video overflow-hidden rounded-2xl border-2 bg-slate-100', frame.video_url ? 'border-emerald-400' : 'border-slate-100')}>
                      {frame.video_url ? (
                        <video src={mediaUrl(frame.video_url)} className="h-full w-full object-cover" muted loop onMouseOver={(event) => event.currentTarget.play()} onMouseOut={(event) => event.currentTarget.pause()} />
                      ) : (
                        <div className="flex h-full items-center justify-center text-slate-300">
                          <AlertCircle size={24} />
                        </div>
                      )}
                      <span className="absolute left-2 top-2 rounded-lg bg-black/50 px-2 py-1 text-[10px] font-black text-white">#{frame.index}</span>
                    </div>
                    <p className="truncate text-xs font-bold text-slate-500">{frame.action || '未命名镜头'}</p>
                  </div>
                ))}
              </div>
            ) : (
              <div className="rounded-2xl border border-dashed border-slate-200 py-16 text-center text-sm font-semibold text-slate-400">暂无分镜</div>
            )}
          </section>
        </main>

        <aside className="space-y-6">
          <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
            <h2 className="mb-5 text-sm font-black uppercase tracking-widest text-slate-500">项目摘要</h2>
            <div className="space-y-3">
              <Summary label="项目" value={currentProject.title} />
              <Summary label="工作流" value={currentProject.workflow} />
              <Summary label="画幅" value={currentProject.model_settings?.storyboard_aspect_ratio || '未配置'} />
              <Summary label="分镜视频" value={`${readyFrames.length}/${frames.length}`} />
              {audit && <Summary label="生产体检" value={`${audit.score}/100 · ${audit.ready_for_export ? '可交付' : '需复核'}`} />}
            </div>
          </section>

          <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
            <h2 className="mb-5 text-sm font-black uppercase tracking-widest text-slate-500">拼接任务</h2>
            {[...pipelineTasks, ...mergeTasks].length > 0 ? (
              <div className="space-y-3">
                {[...pipelineTasks, ...mergeTasks].slice(0, 5).map((task) => (
                  <div key={task.id} className="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <div className="flex items-center justify-between text-xs font-bold text-slate-500">
                      <span>{task.type === 'storyboard_pipeline' ? '一键补全' : '拼接'} / {task.status}</span>
                      <span>{task.progress}%</span>
                    </div>
                    {task.error && <p className="mt-2 text-xs leading-5 text-red-600">{task.error}</p>}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm leading-6 text-slate-500">暂无拼接任务记录。</p>
            )}
          </section>
        </aside>
      </div>
    </div>
  );
}

function ProductionAuditPanel({ audit }: { audit: ProductionAuditReport }) {
  const topIssues = audit.issues.slice(0, 4);
  const frameCount = audit.metrics.frames || 0;
  const lastFrameCount = audit.metrics.frame_last_frames || 0;
  return (
    <section className={cn('rounded-3xl border p-6 shadow-soft', audit.ready_for_export ? 'border-emerald-200 bg-emerald-50' : 'border-amber-200 bg-amber-50')}>
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div className="flex items-start gap-3">
          <div className={cn('rounded-2xl p-3', audit.ready_for_export ? 'bg-emerald-600 text-white' : 'bg-amber-500 text-white')}>
            <ShieldCheck size={24} />
          </div>
          <div>
            <h2 className={cn('text-lg font-black', audit.ready_for_export ? 'text-emerald-950' : 'text-amber-950')}>生产流程体检</h2>
            <p className={cn('mt-1 text-sm font-semibold', audit.ready_for_export ? 'text-emerald-700' : 'text-amber-800')}>
              {audit.ready_for_export ? '当前达到预览交付条件，仍建议人工抽检关键镜头。' : '当前仍有生产风险，建议先处理下方问题。'}
            </p>
          </div>
        </div>
        <div className="text-right">
          <div className={cn('text-4xl font-black', audit.ready_for_export ? 'text-emerald-700' : 'text-amber-700')}>{audit.score}</div>
          <div className="text-xs font-black uppercase tracking-widest text-slate-500">Score</div>
        </div>
      </div>

      <div className="mt-5 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Metric label="效率分" value={`${audit.efficiency_score || 0}`} />
        <Metric label="一致性分" value={`${audit.consistency_score || 0}`} />
        <Metric label="分镜" value={`${audit.metrics.frame_videos || 0}/${audit.metrics.frames || 0}`} />
        <Metric label="资产参考视频" value={`${audit.metrics.referenced_frame_video_tasks || 0}/${audit.metrics.frames || 0}`} />
        <Metric label="角色三视图" value={`${audit.metrics.character_three_views || 0}/${audit.metrics.characters || 0}`} />
        <Metric label="末帧记录" value={`${lastFrameCount}/${frameCount}`} />
      </div>

      {audit.workflow_stages?.length > 0 && (
        <div className="mt-5 grid grid-cols-1 gap-3 md:grid-cols-3">
          {audit.workflow_stages.map((stage) => (
            <div key={stage.id} className="rounded-2xl border border-white/70 bg-white/70 p-4">
              <div className="flex items-center justify-between gap-2">
                <div className="text-sm font-black text-slate-900">{stage.label}</div>
                <span className={cn('rounded-full px-2 py-1 text-[10px] font-black uppercase', stage.status === 'done' ? 'bg-emerald-100 text-emerald-700' : stage.status === 'blocked' ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700')}>
                  {stage.status}
                </span>
              </div>
              <div className="mt-2 text-xs font-semibold leading-5 text-slate-600">{stage.detail}</div>
            </div>
          ))}
        </div>
      )}

      {audit.frame_audits?.length > 0 && (
        <div className="mt-5 rounded-2xl border border-white/70 bg-white/70 p-4">
          <div className="mb-3 text-xs font-black uppercase tracking-widest text-slate-500">镜头一致性抽检</div>
          <div className="grid grid-cols-2 gap-2 md:grid-cols-6">
            {audit.frame_audits.map((frame) => (
              <div key={frame.frame_id} className="rounded-xl bg-white p-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-black text-slate-500">#{frame.index}</span>
                  <span className={cn('rounded-full px-2 py-0.5 text-[10px] font-black uppercase', frame.risk === 'low' ? 'bg-emerald-100 text-emerald-700' : frame.risk === 'medium' ? 'bg-amber-100 text-amber-700' : 'bg-red-100 text-red-700')}>{frame.risk}</span>
                </div>
                <div className="mt-2 text-xs font-semibold text-slate-500">参考 {frame.reference_count} · 末帧 {frame.has_last_frame ? '有' : '无'}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {topIssues.length > 0 && (
        <div className="mt-5 space-y-2">
          {topIssues.map((issue, index) => (
            <div key={`${issue.area}-${index}`} className="rounded-2xl border border-white/70 bg-white/70 p-3">
              <div className="text-xs font-black text-slate-500">{issue.area} · {issue.severity}</div>
              <div className="mt-1 text-sm font-bold text-slate-800">{issue.message}</div>
              {issue.action && <div className="mt-1 text-xs font-semibold text-slate-500">{issue.action}</div>}
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-white/75 p-4">
      <div className="text-[10px] font-black uppercase tracking-widest text-slate-400">{label}</div>
      <div className="mt-1 text-lg font-black text-slate-900">{value}</div>
    </div>
  );
}

function Summary({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-slate-100 bg-slate-50 p-4">
      <div className="mb-1 text-[10px] font-black uppercase tracking-widest text-slate-400">{label}</div>
      <div className="break-words text-sm font-bold text-slate-800">{value}</div>
    </div>
  );
}
