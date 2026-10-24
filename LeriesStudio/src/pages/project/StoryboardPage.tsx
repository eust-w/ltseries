/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { ChangeEvent, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Check, Clapperboard, Copy, Film, Image as ImageIcon, Lock, Plus, RefreshCcw, Sparkles, Trash2, Upload, Wand2, X } from 'lucide-react';
import { useProject } from '../../contexts/ProjectContext';
import { api, mediaUrl } from '../../lib/api';
import { cn } from '../../lib/utils';
import { StoryboardShot } from '../../types';

export default function StoryboardPage() {
  const { projectId } = useParams();
  const { currentProject, setCurrentProjectId, refreshProject, refreshTasks, tasks } = useProject();
  const [creating, setCreating] = useState(false);
  const [action, setAction] = useState('');
  const [camera, setCamera] = useState('');
  const [sceneId, setSceneId] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (projectId) setCurrentProjectId(projectId);
  }, [projectId, setCurrentProjectId]);

  const frames = useMemo(
    () => [...(currentProject?.frames || [])].sort((a, b) => a.index - b.index),
    [currentProject],
  );

  const handleAnalyze = async () => {
    if (!projectId) return;
    setSubmitting(true);
    setMessage('');
    try {
      await api.analyzeStoryboard(projectId);
      await refreshTasks();
      setMessage('分镜解析任务已提交');
    } catch (error: any) {
      setMessage(error?.message || '分镜解析失败');
    } finally {
      setSubmitting(false);
    }
  };

  const createFrame = async () => {
    if (!projectId || !action.trim()) return;
    setCreating(true);
    setMessage('');
    try {
      await api.addFrame(projectId, { action, camera, scene_id: sceneId });
      setAction('');
      setCamera('');
      setSceneId('');
      await refreshProject();
      setMessage('镜头已保存');
    } catch (error: any) {
      setMessage(error?.message || '新增镜头失败');
    } finally {
      setCreating(false);
    }
  };

  if (!currentProject) {
    return <div className="h-64 rounded-2xl bg-slate-100 animate-pulse" />;
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">分镜制作</h1>
          <p className="mt-1 text-sm text-slate-500">分镜、图片、视频和任务状态均来自后端项目数据。</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {message && <span className="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-600">{message}</span>}
          <button
            onClick={async () => {
              await refreshProject();
              await refreshTasks();
            }}
            className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-bold text-slate-700 transition hover:bg-slate-50"
          >
            <RefreshCcw size={18} />
            刷新
          </button>
          <button
            onClick={handleAnalyze}
            disabled={submitting}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-indigo-600/20 transition hover:bg-indigo-500 disabled:opacity-60"
          >
            <Sparkles size={18} />
            {submitting ? '提交中...' : 'AI 生成分镜'}
          </button>
        </div>
      </div>

      <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-soft">
        <h2 className="mb-5 flex items-center gap-2 text-sm font-black uppercase tracking-widest text-slate-500">
          <Plus size={17} className="text-indigo-600" />
          新增镜头
        </h2>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_220px_220px_auto]">
          <textarea
            value={action}
            onChange={(event) => setAction(event.target.value)}
            className="h-24 resize-none rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm font-medium leading-6 outline-none transition focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100"
            placeholder="镜头动作、画面内容、情绪变化..."
          />
          <input
            value={camera}
            onChange={(event) => setCamera(event.target.value)}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm font-medium outline-none transition focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100"
            placeholder="镜头语言"
          />
          <select
            value={sceneId}
            onChange={(event) => setSceneId(event.target.value)}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm font-bold text-slate-700 outline-none transition focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100"
          >
            <option value="">不绑定场景</option>
            {currentProject.scenes?.map((scene) => (
              <option key={scene.id} value={scene.id}>{scene.name}</option>
            ))}
          </select>
          <button
            onClick={createFrame}
            disabled={creating || !action.trim()}
            className="rounded-2xl bg-slate-900 px-6 py-3 text-sm font-bold text-white transition hover:bg-slate-800 disabled:opacity-50"
          >
            {creating ? '保存中...' : '新增'}
          </button>
        </div>
      </section>

      {frames.length > 0 ? (
        <div className="space-y-5">
          {frames.map((frame) => (
            <FrameCard
              key={frame.id}
              frame={frame}
              project={currentProject}
              projectId={projectId!}
              tasks={tasks}
              onUpdate={async () => {
                await refreshProject();
                await refreshTasks();
              }}
            />
          ))}
        </div>
      ) : (
        <div className="rounded-3xl border-2 border-dashed border-slate-200 bg-white py-24 text-center">
          <Clapperboard size={48} className="mx-auto text-slate-300" />
          <h3 className="mt-4 text-lg font-bold text-slate-500">暂无分镜</h3>
          <p className="mt-2 text-sm text-slate-400">点击 AI 生成分镜，或手动新增第一条镜头。</p>
        </div>
      )}
    </div>
  );
}

function FrameCard({ frame, project, projectId, tasks, onUpdate }: { key?: string; frame: StoryboardShot; project: any; projectId: string; tasks: any[]; onUpdate: () => void }) {
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [action, setAction] = useState(frame.action || '');
  const [camera, setCamera] = useState(frame.camera || '');
  const [dialogue, setDialogue] = useState(frame.dialogue || '');
  const [prompt, setPrompt] = useState(frame.prompt || '');
  const [sceneId, setSceneId] = useState(frame.scene_id || '');
  const [characterIds, setCharacterIds] = useState<string[]>(frame.character_ids || []);
  const [rendering, setRendering] = useState(false);
  const [videoing, setVideoing] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setAction(frame.action || '');
    setCamera(frame.camera || '');
    setDialogue(frame.dialogue || '');
    setPrompt(frame.prompt || '');
    setSceneId(frame.scene_id || '');
    setCharacterIds(frame.character_ids || []);
  }, [frame]);

  const selectedScene = project.scenes?.find((scene: any) => scene.id === frame.scene_id);
  const frameTasks = tasks.filter((task) => task.payload?.frame_id === frame.id || task.result?.frame_id === frame.id);
  const previewUrl = frame.rendered_url || frame.image_url || selectedImageURL(frame.rendered_asset) || selectedImageURL(frame.image_asset);

  const save = async () => {
    setSaving(true);
    try {
      await api.updateFrame(projectId, frame.id, {
        action,
        camera,
        dialogue,
        prompt,
        scene_id: sceneId,
        character_ids: characterIds,
      });
      setEditing(false);
      await onUpdate();
    } catch (error: any) {
      alert(error?.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const renderImage = async () => {
    setRendering(true);
    try {
      await api.renderStoryboard(projectId, {
        frame_id: frame.id,
        prompt: prompt || action,
        references: collectFrameReferences(project, characterIds, sceneId, frame),
      });
      await onUpdate();
    } catch (error: any) {
      alert(error?.message || '生成画面失败');
    } finally {
      setRendering(false);
    }
  };

  const createVideo = async () => {
    const imageUrl = previewUrl;
    const workflow = project.workflow || '';
    if (!imageUrl && workflow !== 't2v') {
      alert('请先上传或生成该镜头画面，再生成视频。');
      return;
    }
    setVideoing(true);
    try {
      await api.createVideoTask(projectId, {
        frame_id: frame.id,
        prompt: prompt || action,
        image_url: imageUrl,
        generation_mode: workflow,
        model: project.model_settings?.i2v_model || '',
        reference_image_urls: collectFrameReferences(project, characterIds, sceneId, frame),
        parameters: {
          aspect_ratio: project.model_settings?.storyboard_aspect_ratio || '',
        },
      });
      await onUpdate();
    } catch (error: any) {
      alert(error?.message || '生成视频失败');
    } finally {
      setVideoing(false);
    }
  };

  const uploadImage = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    try {
      await api.uploadFrameImage(projectId, frame.id, file, prompt || action);
      await onUpdate();
    } catch (error: any) {
      alert(error?.message || '上传失败');
    } finally {
      event.target.value = '';
    }
  };

  return (
    <article className="grid grid-cols-1 overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-soft lg:grid-cols-[340px_minmax(0,1fr)_260px]">
      <div className="border-b border-slate-100 bg-slate-50 p-5 lg:border-b-0 lg:border-r">
        <div className="mb-4 flex items-center justify-between">
          <span className="text-xs font-black uppercase tracking-widest text-slate-400">SHOT {String(frame.index).padStart(2, '0')}</span>
          <div className="flex items-center gap-2">
            {frame.locked && <Lock size={16} className="text-amber-500" />}
            <button onClick={() => setEditing(true)} className="rounded-lg border border-slate-200 bg-white px-2 py-1 text-xs font-bold text-slate-500 hover:text-indigo-600">编辑</button>
          </div>
        </div>
        <div className="aspect-video overflow-hidden rounded-2xl border border-slate-200 bg-slate-200">
          {previewUrl ? (
            <img src={mediaUrl(previewUrl)} className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2 text-slate-400">
              <ImageIcon size={30} />
              <span className="text-xs font-bold">暂无画面</span>
            </div>
          )}
        </div>
        <input ref={fileInputRef} type="file" accept="image/*" className="hidden" onChange={uploadImage} />
        <div className="mt-4 grid grid-cols-2 gap-2">
          <button onClick={() => fileInputRef.current?.click()} className="inline-flex items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-bold text-slate-600 hover:bg-slate-50">
            <Upload size={15} />
            上传
          </button>
          <button onClick={renderImage} disabled={rendering} className="inline-flex items-center justify-center gap-2 rounded-xl border border-indigo-200 bg-indigo-50 px-3 py-2 text-xs font-bold text-indigo-700 hover:bg-indigo-100 disabled:opacity-60">
            <Wand2 size={15} className={rendering ? 'animate-spin' : ''} />
            生成图
          </button>
        </div>
      </div>

      <div className="space-y-5 p-6">
        {editing ? (
          <div className="space-y-4">
            <TextArea label="画面动作" value={action} onChange={setAction} rows={4} />
            <TextArea label="对白/旁白" value={dialogue} onChange={setDialogue} rows={3} />
            <TextArea label="生成 Prompt" value={prompt} onChange={setPrompt} rows={4} />
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <label className="space-y-2">
                <span className="block text-xs font-black uppercase tracking-widest text-slate-400">镜头语言</span>
                <input value={camera} onChange={(event) => setCamera(event.target.value)} className="w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100" />
              </label>
              <label className="space-y-2">
                <span className="block text-xs font-black uppercase tracking-widest text-slate-400">场景</span>
                <select value={sceneId} onChange={(event) => setSceneId(event.target.value)} className="w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm font-bold outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100">
                  <option value="">不绑定场景</option>
                  {project.scenes?.map((scene: any) => (
                    <option key={scene.id} value={scene.id}>{scene.name}</option>
                  ))}
                </select>
              </label>
            </div>
            <div>
              <span className="mb-2 block text-xs font-black uppercase tracking-widest text-slate-400">角色引用</span>
              <div className="flex flex-wrap gap-2">
                {project.characters?.map((character: any) => {
                  const checked = characterIds.includes(character.id);
                  return (
                    <button
                      key={character.id}
                      onClick={() => setCharacterIds(checked ? characterIds.filter((id) => id !== character.id) : [...characterIds, character.id])}
                      className={cn('rounded-xl border px-3 py-2 text-xs font-bold transition', checked ? 'border-indigo-300 bg-indigo-50 text-indigo-700' : 'border-slate-200 bg-slate-50 text-slate-500')}
                    >
                      {character.name}
                    </button>
                  );
                })}
              </div>
            </div>
            <div className="flex justify-end gap-3">
              <button onClick={() => setEditing(false)} className="inline-flex items-center gap-2 rounded-xl border border-slate-200 px-4 py-2 text-sm font-bold text-slate-600 hover:bg-slate-50">
                <X size={17} />
                取消
              </button>
              <button onClick={save} disabled={saving} className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-5 py-2 text-sm font-bold text-white hover:bg-indigo-500 disabled:opacity-60">
                <Check size={17} />
                {saving ? '保存中...' : '保存'}
              </button>
            </div>
          </div>
        ) : (
          <>
            <div>
              <div className="mb-2 text-xs font-black uppercase tracking-widest text-slate-400">画面动作</div>
              <p className="text-base font-bold leading-7 text-slate-900">{frame.action}</p>
            </div>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <InfoBlock label="镜头语言" value={frame.camera || '未设置'} />
              <InfoBlock label="场景" value={selectedScene?.name || '未绑定'} />
            </div>
            <InfoBlock label="对白/旁白" value={frame.dialogue || '未设置'} />
            <InfoBlock label="Prompt" value={frame.prompt || '未设置'} />
            <div className="flex flex-wrap gap-2">
              {(frame.character_ids || []).map((id) => {
                const character = project.characters?.find((item: any) => item.id === id);
                return <span key={id} className="rounded-xl bg-indigo-50 px-3 py-1.5 text-xs font-bold text-indigo-700">{character?.name || id}</span>;
              })}
            </div>
          </>
        )}
      </div>

      <div className="border-t border-slate-100 bg-slate-50 p-5 lg:border-l lg:border-t-0">
        <div className="aspect-video overflow-hidden rounded-2xl bg-slate-950">
          {frame.video_url ? (
            <video src={mediaUrl(frame.video_url)} controls className="h-full w-full object-contain" />
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2 text-white/30">
              <Film size={30} />
              <span className="text-xs font-bold">暂无视频</span>
            </div>
          )}
        </div>
        <button onClick={createVideo} disabled={videoing} className="mt-4 w-full rounded-xl bg-slate-900 px-4 py-2.5 text-sm font-bold text-white transition hover:bg-slate-800 disabled:opacity-60">
          {videoing ? '提交中...' : '生成视频'}
        </button>
        <div className="mt-4 grid grid-cols-3 gap-2">
          <button onClick={async () => { await api.copyFrame(projectId, frame.id); await onUpdate(); }} className="rounded-xl border border-slate-200 bg-white p-2 text-slate-500 hover:text-indigo-600" title="复制">
            <Copy size={17} className="mx-auto" />
          </button>
          <button onClick={async () => { await api.toggleFrameLock(projectId, frame.id); await onUpdate(); }} className="rounded-xl border border-slate-200 bg-white p-2 text-slate-500 hover:text-amber-600" title="锁定">
            <Lock size={17} className="mx-auto" />
          </button>
          <button
            onClick={async () => {
              if (!window.confirm('确定删除该镜头吗？')) return;
              await api.deleteFrame(projectId, frame.id);
              await onUpdate();
            }}
            className="rounded-xl border border-slate-200 bg-white p-2 text-slate-500 hover:text-red-600"
            title="删除"
          >
            <Trash2 size={17} className="mx-auto" />
          </button>
        </div>
        {frameTasks.length > 0 && (
          <div className="mt-4 space-y-2">
            {frameTasks.slice(0, 3).map((task) => (
              <div key={task.id} className="rounded-xl border border-slate-200 bg-white p-3">
                <div className="flex items-center justify-between text-[11px] font-bold text-slate-500">
                  <span>{task.type}</span>
                  <span>{task.status}</span>
                </div>
                {task.error && <p className="mt-2 text-xs leading-5 text-red-600">{task.error}</p>}
              </div>
            ))}
          </div>
        )}
      </div>
    </article>
  );
}

function TextArea({ label, value, onChange, rows }: { label: string; value: string; onChange: (value: string) => void; rows: number }) {
  return (
    <label className="space-y-2">
      <span className="block text-xs font-black uppercase tracking-widest text-slate-400">{label}</span>
      <textarea value={value} onChange={(event) => onChange(event.target.value)} rows={rows} className="w-full resize-y rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm leading-6 outline-none focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100" />
    </label>
  );
}

function InfoBlock({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-slate-50 p-4">
      <div className="mb-1 text-[10px] font-black uppercase tracking-widest text-slate-400">{label}</div>
      <div className="text-sm font-semibold leading-6 text-slate-700">{value}</div>
    </div>
  );
}

function selectedImageURL(asset?: any) {
  if (!asset) return '';
  const variants = asset.variants || asset.image_variants || [];
  const selectedId = asset.selected_id || asset.selected_image_id;
  const selected = variants.find((variant: any) => variant.id === selectedId) || variants[0];
  return selected?.url || '';
}

function collectFrameReferences(project: any, characterIds: string[], sceneId: string, frame?: any) {
  const urls: string[] = [];
  const add = (url: string) => {
    if (!url || urls.includes(url)) return;
    urls.push(url);
  };
  for (const id of characterIds) {
    const character = findEntity(project.characters, id);
    add(selectedImageURL(character?.full_body));
    add(selectedImageURL(character?.three_views));
    add(selectedImageURL(character?.head_shot));
    add(selectedImageURL(character?.image));
  }
  const scene = findEntity(project.scenes, sceneId);
  add(selectedImageURL(scene?.image));
  const text = `${frame?.action || ''} ${frame?.camera || ''} ${frame?.dialogue || ''} ${frame?.prompt || ''}`;
  for (const prop of project.props || []) {
    if ((project.props || []).length <= 3 || entityMentioned(text, prop)) {
      add(selectedImageURL(prop.image));
    }
  }
  return urls;
}

function findEntity(entities: any[] = [], reference: string) {
  const key = normalizeReference(reference);
  return entities.find((item) => normalizeReference(item.id) === key || normalizeReference(item.name) === key);
}

function entityMentioned(text: string, entity: any) {
  const normalizedText = normalizeReference(text);
  return [entity?.id, entity?.name].some((value) => {
    const key = normalizeReference(value || '');
    return key && normalizedText.includes(key);
  });
}

function normalizeReference(value: string) {
  return String(value || '').trim().replace(/\s+/g, '').toLowerCase();
}
