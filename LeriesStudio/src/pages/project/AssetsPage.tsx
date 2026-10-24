/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { ChangeEvent, FormEvent, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Box, Check, ImageIcon, Lock, Map, Plus, RefreshCw, Sparkles, Star, Trash2, Unlock, Upload, Users, Video, Wand2, X } from 'lucide-react';
import { useProject } from '../../contexts/ProjectContext';
import { api, mediaUrl } from '../../lib/api';
import { cn } from '../../lib/utils';
import { AssetUnit, Entity, ImageAsset, ImageVariant, VideoVariant } from '../../types';

type AssetType = 'character' | 'scene' | 'prop';

const ASSET_TABS = [
  { id: 'character', label: '角色', icon: Users },
  { id: 'scene', label: '场景', icon: Map },
  { id: 'prop', label: '道具', icon: Box },
] as const;

export default function AssetsPage() {
  const { projectId } = useParams();
  const { currentProject, setCurrentProjectId, refreshProject, refreshTasks, tasks } = useProject();
  const [activeTab, setActiveTab] = useState<AssetType>('character');
  const [showCreate, setShowCreate] = useState(false);
  const [generatingAll, setGeneratingAll] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (projectId) setCurrentProjectId(projectId);
  }, [projectId, setCurrentProjectId]);

  const entities = useMemo(() => {
    if (!currentProject) return [];
    if (activeTab === 'character') return currentProject.characters || [];
    if (activeTab === 'scene') return currentProject.scenes || [];
    return currentProject.props || [];
  }, [activeTab, currentProject]);

  const generateAllAssets = async (force = false) => {
    if (!projectId || !currentProject) return;
    setGeneratingAll(true);
    setMessage('');
    try {
      await api.generateAssets(projectId, {
        model: currentProject.model_settings?.image_model || currentProject.model_settings?.t2i_model || '',
        force,
        parameters: {},
      });
      await refreshTasks();
      setMessage(force ? '刷新参考资产任务已提交' : '资产生成任务已提交');
    } catch (error: any) {
      setMessage(error?.message || '任务提交失败');
    } finally {
      setGeneratingAll(false);
    }
  };

  if (!currentProject) {
    return <div className="h-64 rounded-2xl bg-slate-100 animate-pulse" />;
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">资产库</h1>
          <p className="mt-1 text-sm text-slate-500">角色三视图、全身图、头像、场景、道具图片与视频均从后端读取并写回。</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {message && <span className="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-600">{message}</span>}
          <button
            onClick={() => generateAllAssets(false)}
            disabled={generatingAll}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-indigo-600/20 transition hover:bg-indigo-500 disabled:opacity-60"
          >
            <Sparkles size={18} />
            {generatingAll ? '提交中...' : '补全缺失资产图'}
          </button>
          <button
            onClick={() => generateAllAssets(true)}
            disabled={generatingAll}
            className="inline-flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm font-bold text-amber-800 transition hover:bg-amber-100 disabled:opacity-60"
          >
            <RefreshCw size={18} />
            {generatingAll ? '提交中...' : '刷新参考资产'}
          </button>
          <button
            onClick={() => setShowCreate(true)}
            className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-bold text-slate-700 transition hover:bg-slate-50"
          >
            <Plus size={18} />
            添加资产
          </button>
        </div>
      </div>

      <div className="flex flex-wrap gap-2 border-b border-slate-200">
        {ASSET_TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={cn(
              'flex items-center gap-2 border-b-2 px-5 py-4 text-sm font-bold transition',
              activeTab === tab.id
                ? 'border-indigo-600 text-indigo-600'
                : 'border-transparent text-slate-400 hover:text-slate-600',
            )}
          >
            <tab.icon size={18} />
            {tab.label}
            <span className="rounded-lg bg-slate-100 px-2 py-0.5 text-[10px] font-black text-slate-500">
              {tab.id === 'character' ? currentProject.characters?.length || 0 : tab.id === 'scene' ? currentProject.scenes?.length || 0 : currentProject.props?.length || 0}
            </span>
          </button>
        ))}
      </div>

      {entities.length > 0 ? (
        <div className="space-y-8">
          {entities.map((entity) => (
            <AssetCard
              key={entity.id}
              entity={entity}
              type={activeTab}
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
          <Box size={48} className="mx-auto text-slate-300" />
          <h3 className="mt-4 text-lg font-bold text-slate-500">暂无{labelForType(activeTab)}</h3>
          <p className="mt-2 text-sm text-slate-400">先运行剧本分析，或手动添加资产。</p>
        </div>
      )}

      {showCreate && (
        <CreateAssetModal
          type={activeTab}
          projectId={projectId!}
          onClose={() => setShowCreate(false)}
          onCreated={async () => {
            setShowCreate(false);
            await refreshProject();
          }}
        />
      )}
    </div>
  );
}

function AssetCard({ entity, type, project, projectId, tasks, onUpdate }: { key?: string; entity: Entity; type: AssetType; project: any; projectId: string; tasks: any[]; onUpdate: () => void }) {
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(entity.name);
  const [description, setDescription] = useState(entity.description);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setName(entity.name);
    setDescription(entity.description);
  }, [entity]);

  const save = async () => {
    setSaving(true);
    try {
      await api.updateAsset(projectId, type, entity.id, { name, description });
      setEditing(false);
      await onUpdate();
    } catch (error: any) {
      alert(error?.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!window.confirm(`确定删除「${entity.name}」吗？`)) return;
    try {
      await api.deleteAsset(projectId, type, entity.id);
      await onUpdate();
    } catch (error: any) {
      alert(error?.message || '删除失败');
    }
  };

  const slots = type === 'character'
    ? [
        { key: 'full_body', title: '全身图', unit: entity.full_body },
        { key: 'three_views', title: '三视图', unit: entity.three_views },
        { key: 'head_shot', title: '头像特写', unit: entity.head_shot },
        { key: 'image', title: '通用图', unit: entity.image },
      ]
    : [{ key: 'image', title: '主要图', unit: entity.image }];

  const entityTasks = tasks.filter((task) => task.payload?.asset_id === entity.id);

  return (
    <article className="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-soft">
      <div className="flex flex-col gap-4 border-b border-slate-100 p-6 xl:flex-row xl:items-start xl:justify-between">
        <div className="min-w-0 flex-1">
          {editing ? (
            <div className="space-y-4">
              <input
                value={name}
                onChange={(event) => setName(event.target.value)}
                className="w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-lg font-bold outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              />
              <textarea
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                className="h-28 w-full resize-y rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm leading-6 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              />
            </div>
          ) : (
            <>
              <div className="flex flex-wrap items-center gap-3">
                <h2 className="text-xl font-black text-slate-900">{entity.name}</h2>
                <span className="rounded-lg bg-slate-100 px-2 py-1 text-[10px] font-black uppercase tracking-widest text-slate-500">{entity.status}</span>
              </div>
              <p className="mt-2 max-w-4xl text-sm leading-7 text-slate-500">{entity.description}</p>
            </>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button
            onClick={async () => {
              await api.toggleAssetLock(projectId, type, entity.id);
              await onUpdate();
            }}
            className={cn('inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-xs font-bold', entity.locked ? 'border-amber-200 bg-amber-50 text-amber-700' : 'border-slate-200 bg-slate-50 text-slate-600')}
          >
            {entity.locked ? <Lock size={15} /> : <Unlock size={15} />}
            {entity.locked ? '已锁定' : '未锁定'}
          </button>
          {editing ? (
            <>
              <button onClick={() => setEditing(false)} className="rounded-xl border border-slate-200 px-3 py-2 text-xs font-bold text-slate-600 hover:bg-slate-50">
                取消
              </button>
              <button onClick={save} disabled={saving} className="rounded-xl bg-indigo-600 px-4 py-2 text-xs font-bold text-white hover:bg-indigo-500 disabled:opacity-60">
                {saving ? '保存中...' : '保存'}
              </button>
            </>
          ) : (
            <button onClick={() => setEditing(true)} className="rounded-xl border border-slate-200 px-3 py-2 text-xs font-bold text-slate-600 hover:bg-slate-50">
              修改
            </button>
          )}
          <button onClick={remove} className="rounded-xl border border-slate-200 p-2 text-slate-500 hover:border-red-200 hover:bg-red-50 hover:text-red-600">
            <Trash2 size={17} />
          </button>
        </div>
      </div>

      <div className={cn('grid gap-6 p-6', type === 'character' ? 'grid-cols-1 xl:grid-cols-4' : 'grid-cols-1 lg:grid-cols-2')}>
        {slots.map((slot) => (
          <AssetSlot
            key={slot.key}
            title={slot.title}
            uploadType={slot.key}
            entity={entity}
            unit={slot.unit}
            type={type}
            project={project}
            projectId={projectId}
            onUpdate={onUpdate}
          />
        ))}
      </div>

      {entityTasks.length > 0 && (
        <div className="border-t border-slate-100 bg-slate-50 p-5">
          <div className="mb-3 text-xs font-black uppercase tracking-widest text-slate-400">相关任务</div>
          <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
            {entityTasks.slice(0, 6).map((task) => (
              <div key={task.id} className="rounded-2xl border border-slate-200 bg-white p-4">
                <div className="flex items-center justify-between text-xs font-bold text-slate-500">
                  <span>{task.type}</span>
                  <span>{task.status}</span>
                </div>
                {task.error && <p className="mt-2 text-xs leading-5 text-red-600">{task.error}</p>}
              </div>
            ))}
          </div>
        </div>
      )}
    </article>
  );
}

function AssetSlot({ title, uploadType, entity, unit, type, project, projectId, onUpdate }: {
  key?: string;
  title: string;
  uploadType: string;
  entity: Entity;
  unit: AssetUnit | ImageAsset | any;
  type: AssetType;
  project: any;
  projectId: string;
  onUpdate: () => void;
}) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const normalized = normalizeUnit(unit);
  const selectedImage = selectVariant(normalized.imageVariants, normalized.selectedImageId);
  const selectedVideo = selectVariant(normalized.videoVariants, normalized.selectedVideoId) || normalized.videoVariants[0] || (uploadType === 'image' ? entity.video_assets?.[0] : undefined);
  const [imagePrompt, setImagePrompt] = useState(normalized.imagePrompt || defaultAssetPrompt(entity, uploadType));
  const [videoPrompt, setVideoPrompt] = useState(normalized.videoPrompt || defaultVideoPrompt(entity, uploadType));
  const [submitting, setSubmitting] = useState<'image' | 'video' | 'upload' | null>(null);

  useEffect(() => {
    setImagePrompt(normalized.imagePrompt || defaultAssetPrompt(entity, uploadType));
    setVideoPrompt(normalized.videoPrompt || defaultVideoPrompt(entity, uploadType));
  }, [entity.id, entity.description, normalized.imagePrompt, normalized.videoPrompt, uploadType]);

  const upload = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    setSubmitting('upload');
    try {
      await api.uploadAsset(projectId, type, entity.id, uploadType, file, imagePrompt);
      await onUpdate();
    } catch (error: any) {
      alert(error?.message || '上传失败');
    } finally {
      setSubmitting(null);
      event.target.value = '';
    }
  };

  const generateImage = async () => {
    setSubmitting('image');
    try {
      await api.generateAssetImage(projectId, type, entity.id, {
        asset_type: uploadType,
        prompt: imagePrompt,
        model: project.model_settings?.image_model || project.model_settings?.t2i_model || '',
        parameters: {},
      });
      await onUpdate();
    } catch (error: any) {
      alert(error?.message || '图片生成任务失败');
    } finally {
      setSubmitting(null);
    }
  };

  const generateVideo = async () => {
    if (!selectedImage?.url) {
      alert('请先上传或生成该资产图片。');
      return;
    }
    setSubmitting('video');
    try {
      await api.generateAssetVideo(projectId, type, entity.id, {
        upload_type: uploadType,
        prompt: videoPrompt,
        model: project.model_settings?.i2v_model || '',
        generation_mode: project.workflow || '',
        parameters: {},
      });
      await onUpdate();
    } catch (error: any) {
      alert(error?.message || '视频生成任务失败');
    } finally {
      setSubmitting(null);
    }
  };

  const selectImage = async (variant: ImageVariant) => {
    try {
      await api.selectAssetVariant(projectId, type, entity.id, { upload_type: uploadType, variant_id: variant.id });
      await onUpdate();
    } catch (error: any) {
      alert(error?.message || '选择失败');
    }
  };

  return (
    <div className="rounded-3xl border border-slate-200 bg-slate-50 p-4">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-sm font-black text-slate-700">{title}</h3>
        <div className="flex items-center gap-1">
          <input ref={fileInputRef} type="file" accept="image/*" className="hidden" onChange={upload} />
          <button onClick={() => fileInputRef.current?.click()} className="rounded-lg p-2 text-slate-500 hover:bg-white hover:text-indigo-600" title="上传图片">
            <Upload size={16} className={submitting === 'upload' ? 'animate-pulse' : ''} />
          </button>
          <button onClick={generateImage} disabled={submitting === 'image'} className="rounded-lg p-2 text-slate-500 hover:bg-white hover:text-indigo-600 disabled:opacity-50" title="生成图片">
            <Wand2 size={16} className={submitting === 'image' ? 'animate-spin' : ''} />
          </button>
        </div>
      </div>

      <div className="aspect-[4/5] overflow-hidden rounded-2xl border border-slate-200 bg-white">
        {selectedImage?.url ? (
          <img src={mediaUrl(selectedImage.url)} className="h-full w-full object-cover" />
        ) : (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-slate-300">
            <ImageIcon size={36} />
            <span className="text-xs font-bold">暂无图片</span>
          </div>
        )}
      </div>

      <textarea
        value={imagePrompt}
        onChange={(event) => setImagePrompt(event.target.value)}
        className="mt-4 h-24 w-full resize-y rounded-2xl border border-slate-200 bg-white px-3 py-2 text-xs leading-5 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
        placeholder="图片生成提示词"
      />

      {normalized.imageVariants.length > 0 && (
        <div className="mt-4 flex gap-2 overflow-x-auto pb-1">
          {normalized.imageVariants.map((variant) => (
            <div key={variant.id} className="group relative h-14 w-14 flex-shrink-0">
              <button
                onClick={() => selectImage(variant)}
                className={cn('h-14 w-14 overflow-hidden rounded-xl border-2 bg-white', variant.id === normalized.selectedImageId ? 'border-indigo-600' : 'border-transparent opacity-70 hover:opacity-100')}
              >
                <img src={mediaUrl(variant.url)} className="h-full w-full object-cover" />
              </button>
              <div className="absolute inset-x-0 -top-8 hidden justify-center gap-1 group-hover:flex">
                <button
                  onClick={async () => {
                    await api.favoriteAssetVariant(projectId, type, entity.id, uploadType, variant.id, !variant.favorited);
                    await onUpdate();
                  }}
                  className="rounded-lg bg-white p-1 text-slate-500 shadow hover:text-amber-500"
                  title="收藏"
                >
                  <Star size={13} className={variant.favorited ? 'fill-amber-400 text-amber-400' : ''} />
                </button>
                <button
                  onClick={async () => {
                    if (!window.confirm('确定删除这个图片版本吗？')) return;
                    await api.deleteAssetVariant(projectId, type, entity.id, uploadType, variant.id);
                    await onUpdate();
                  }}
                  className="rounded-lg bg-white p-1 text-slate-500 shadow hover:text-red-600"
                  title="删除"
                >
                  <Trash2 size={13} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="mt-5 rounded-2xl border border-slate-200 bg-white p-3">
        <div className="mb-3 flex items-center justify-between">
          <div className="flex items-center gap-2 text-xs font-black uppercase tracking-widest text-slate-400">
            <Video size={14} />
            视频
          </div>
          <button onClick={generateVideo} disabled={submitting === 'video'} className="rounded-lg bg-slate-900 px-3 py-1.5 text-[11px] font-bold text-white hover:bg-slate-800 disabled:opacity-50">
            {submitting === 'video' ? '提交中...' : '生成'}
          </button>
        </div>
        {videoURL(selectedVideo) ? (
          <video src={mediaUrl(videoURL(selectedVideo))} controls className="aspect-video w-full rounded-xl bg-slate-950 object-contain" />
        ) : (
          <div className="flex aspect-video items-center justify-center rounded-xl bg-slate-100 text-xs font-bold text-slate-400">暂无视频</div>
        )}
        <textarea
          value={videoPrompt}
          onChange={(event) => setVideoPrompt(event.target.value)}
          className="mt-3 h-20 w-full resize-y rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-xs leading-5 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
          placeholder="视频生成提示词"
        />
        {(normalized.videoVariants.length > 0 || entity.video_assets?.length > 0) && (
          <div className="mt-3 space-y-2">
            {[...normalized.videoVariants, ...(uploadType === 'image' ? entity.video_assets || [] : [])].map((video: VideoVariant | any) => (
              <div key={video.id} className="flex items-center justify-between rounded-xl bg-slate-50 px-3 py-2 text-xs font-semibold text-slate-500">
                <span className="truncate">{video.prompt || video.id}</span>
                <button
                  onClick={async () => {
                    if (!window.confirm('确定删除这个视频版本吗？')) return;
                    await api.deleteAssetVideo(projectId, type, entity.id, uploadType, video.id);
                    await onUpdate();
                  }}
                  className="ml-2 text-slate-400 hover:text-red-600"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function CreateAssetModal({ type, projectId, onClose, onCreated }: { type: AssetType; projectId: string; onClose: () => void; onCreated: () => void }) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setLoading(true);
    try {
      await api.addAsset(projectId, type, { name, description });
      await onCreated();
    } catch (error: any) {
      alert(error?.message || '添加失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-slate-900/60 backdrop-blur-sm" onClick={onClose} />
      <form onSubmit={submit} className="relative w-full max-w-xl rounded-3xl bg-white p-8 shadow-2xl">
        <h2 className="text-2xl font-bold text-slate-900">添加{labelForType(type)}</h2>
        <div className="mt-8 space-y-5">
          <label className="block space-y-2">
            <span className="text-sm font-bold text-slate-700">名称</span>
            <input required value={name} onChange={(event) => setName(event.target.value)} className="w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100" />
          </label>
          <label className="block space-y-2">
            <span className="text-sm font-bold text-slate-700">描述</span>
            <textarea required value={description} onChange={(event) => setDescription(event.target.value)} className="h-36 w-full resize-y rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 leading-6 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100" />
          </label>
        </div>
        <div className="mt-8 flex justify-end gap-3">
          <button type="button" onClick={onClose} className="rounded-xl border border-slate-200 px-5 py-2.5 text-sm font-bold text-slate-600 hover:bg-slate-50">取消</button>
          <button disabled={loading} className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-6 py-2.5 text-sm font-bold text-white hover:bg-indigo-500 disabled:opacity-60">
            <Check size={17} />
            {loading ? '提交中...' : '保存'}
          </button>
        </div>
      </form>
    </div>
  );
}

function normalizeUnit(unit: AssetUnit | ImageAsset | any) {
  return {
    selectedImageId: unit?.selected_image_id || unit?.selected_id || '',
    imageVariants: (unit?.image_variants || unit?.variants || []) as ImageVariant[],
    selectedVideoId: unit?.selected_video_id || '',
    videoVariants: (unit?.video_variants || []) as VideoVariant[],
    imagePrompt: unit?.image_prompt || '',
    videoPrompt: unit?.video_prompt || '',
  };
}

function selectVariant<T extends { id: string }>(variants: T[], selectedId?: string) {
  return variants.find((variant) => variant.id === selectedId) || variants[0];
}

function videoURL(video: VideoVariant | any) {
  return video?.url || video?.video_url || '';
}

function defaultAssetPrompt(entity: Entity, uploadType: string) {
  const base = `${entity.name}，${entity.description}`.trim();
  if (uploadType === 'full_body') return `${base}\n完整全身设定图，服装、体态和关键配饰清晰。`;
  if (uploadType === 'three_views') return `${base}\n同一角色三视图，正面、侧面、背面保持一致。`;
  if (uploadType === 'head_shot') return `${base}\n头像特写，五官、发型和表情细节清晰。`;
  return base;
}

function defaultVideoPrompt(entity: Entity, uploadType: string) {
  return `${entity.name}，基于「${uploadType}」资产生成短剧镜头，保持角色、场景和道具一致。`;
}

function labelForType(type: AssetType) {
  if (type === 'character') return '角色';
  if (type === 'scene') return '场景';
  return '道具';
}
