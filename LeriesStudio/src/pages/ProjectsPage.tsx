/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../lib/api';
import { Project } from '../types';
import { Plus, Search, Calendar, Film, Trash2, Pencil } from 'lucide-react';
import { formatDate } from '../lib/utils';
import { useProject } from '../contexts/ProjectContext';

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const { setCurrentProjectId } = useProject();
  const navigate = useNavigate();

  useEffect(() => {
    setCurrentProjectId(null); // Clear current project when on list
    fetchProjects();
  }, [setCurrentProjectId]);

  const fetchProjects = async () => {
    setLoading(true);
    try {
      const data = await api.getProjects();
      setProjects(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error(err);
      setProjects([]);
    } finally {
      setLoading(false);
    }
  };

  const filteredProjects = projects.filter(p =>
    (p.title || '').toLowerCase().includes(search.toLowerCase()) ||
    (p.script || '').toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-8">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-slate-900 font-sans">我的项目</h1>
          <p className="text-slate-500 mt-1">管理您的所有短剧生产流程</p>
        </div>
        <button
          onClick={() => setIsCreateModalOpen(true)}
          className="inline-flex items-center gap-2 bg-indigo-600 hover:bg-indigo-500 text-white px-5 py-2.5 rounded-xl font-semibold shadow-lg shadow-indigo-600/20 transition-all active:scale-95"
        >
          <Plus size={20} />
          新建项目
        </button>
      </div>

      <div className="relative">
        <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-400" size={20} />
        <input
          type="text"
          placeholder="搜索项目标题或剧本内容..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full bg-white border border-slate-200 rounded-2xl py-4 pl-12 pr-6 shadow-soft focus:outline-none focus:ring-2 focus:ring-indigo-600/10 focus:border-indigo-600/50 transition-all"
        />
      </div>

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map(i => (
            <div key={i} className="h-64 bg-slate-100 rounded-2xl animate-pulse" />
          ))}
        </div>
      ) : filteredProjects.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredProjects.map(project => (
            <ProjectCard
              key={project.id}
              project={project}
              onDelete={fetchProjects}
              onEdit={(p: Project) => setEditingProject(p)}
            />
          ))}
        </div>
      ) : (
        <div className="text-center py-20 bg-white rounded-3xl border-2 border-dashed border-slate-200">
          <Film size={48} className="mx-auto text-slate-300 mb-4" />
          <h3 className="text-lg font-semibold text-slate-900">未发现项目</h3>
          <p className="text-slate-500 mt-1">开始创建您的第一个短剧项目吧</p>
          <button
            onClick={() => setIsCreateModalOpen(true)}
            className="mt-6 text-indigo-600 font-semibold hover:underline"
          >
            点击新建项目
          </button>
        </div>
      )}

      {isCreateModalOpen && (
        <ProjectFormModal
          onClose={() => setIsCreateModalOpen(false)}
          onSuccess={(p) => navigate(`/projects/${p.id}/script`)}
          mode="create"
        />
      )}

      {editingProject && (
        <ProjectFormModal
          project={editingProject}
          onClose={() => setEditingProject(null)}
          onSuccess={() => { setEditingProject(null); fetchProjects(); }}
          mode="edit"
        />
      )}
    </div>
  );
}

type ProjectCardProps = {
  project: Project;
  onDelete: () => Promise<void>;
  onEdit: (project: Project) => void;
};

const ProjectCard: React.FC<ProjectCardProps> = ({ project, onDelete, onEdit }) => {
  const navigate = useNavigate();
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDelete = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!window.confirm('确定要删除这个项目吗？')) return;
    setIsDeleting(true);
    try {
      await api.deleteProject(project.id);
      await onDelete();
    } catch (err) {
      alert('删除失败');
      setIsDeleting(false);
    }
  };

  return (
    <div
      onClick={() => navigate(`/projects/${project.id}/script`)}
      className="group bg-white rounded-2xl border border-slate-200 p-6 shadow-soft hover:shadow-xl hover:-translate-y-1 transition-all cursor-pointer relative overflow-hidden"
    >
      <div className="absolute top-0 left-0 w-full h-1.5 bg-indigo-600 opacity-0 group-hover:opacity-100 transition-opacity" />

      <div className="flex justify-between items-start mb-4">
        <div className="w-12 h-12 rounded-xl bg-slate-50 flex items-center justify-center text-slate-400 group-hover:bg-indigo-50 group-hover:text-indigo-600 transition-colors">
          <Film size={24} />
        </div>
        <div className="flex gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); onEdit(project); }}
            className="p-2 text-slate-400 hover:text-indigo-600 hover:bg-indigo-50 rounded-lg transition-colors opacity-0 group-hover:opacity-100"
          >
            <Pencil size={18} />
          </button>
          <button
            onClick={handleDelete}
            disabled={isDeleting}
            className="p-2 text-slate-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors opacity-0 group-hover:opacity-100"
          >
            <Trash2 size={18} />
          </button>
        </div>
      </div>

      <h3 className="text-lg font-bold text-slate-900 mb-2 truncate">{project.title}</h3>
      <p className="text-slate-500 text-sm line-clamp-2 mb-6 h-10">
        {project.script || '暂无剧本内容'}
      </p>

      <div className="flex items-center justify-between pt-4 border-t border-slate-50">
        <div className="flex items-center gap-1.5 text-xs font-medium text-slate-400">
          <Calendar size={14} />
          {formatDate(project.created_at)}
        </div>
        <div className="text-xs font-bold text-indigo-600 bg-indigo-50 px-2 py-1 rounded-md uppercase">
          {project.workflow}
        </div>
      </div>
    </div>
  );
};

function ProjectFormModal({
  onClose,
  onSuccess,
  project,
  mode = 'create'
}: {
  onClose: () => void,
  onSuccess: (p: Project) => void,
  project?: Project,
  mode: 'create' | 'edit'
}) {
  const [title, setTitle] = useState(project?.title || '');
  const [script, setScript] = useState(project?.script || '');
  const [workflow, setWorkflow] = useState(project?.workflow || 'r2v');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      let p;
      if (mode === 'create') {
        p = await api.createProject({ title, script, workflow });
      } else {
        p = await api.updateProject(project!.id, { title, script, workflow });
      }
      onSuccess(p);
    } catch (err) {
      alert(mode === 'create' ? '创建项目失败' : '更新项目失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-slate-900/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-white rounded-3xl shadow-2xl w-full max-w-xl overflow-hidden">
        <div className="p-8">
          <h2 className="text-2xl font-bold text-slate-900 mb-1">
            {mode === 'create' ? '新建项目' : '编辑项目信息'}
          </h2>
          <p className="text-slate-500 mb-8">
            {mode === 'create' ? '开始一个新的短剧创作之旅' : '修改项目的基础设定'}
          </p>

          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label className="block text-sm font-semibold text-slate-700 mb-2">项目标题</label>
              <input
                type="text"
                required
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-3 focus:outline-none focus:ring-2 focus:ring-indigo-600/20 focus:border-indigo-600 transition-all font-medium"
                placeholder="例如：第一集 · 雨夜邂逅"
              />
            </div>

            <div>
              <label className="block text-sm font-semibold text-slate-700 mb-2">剧本原文</label>
              <textarea
                required
                value={script}
                onChange={(e) => setScript(e.target.value)}
                className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-3 h-40 focus:outline-none focus:ring-2 focus:ring-indigo-600/20 focus:border-indigo-600 transition-all font-medium resize-none"
                placeholder="在此输入或粘贴剧本内容..."
              />
            </div>

            <div>
              <label className="block text-sm font-semibold text-slate-700 mb-3">默认工作流</label>
              <div className="grid grid-cols-3 gap-3">
                {['r2v', 'i2v', 't2v'].map(w => (
                  <button
                    key={w}
                    type="button"
                    onClick={() => setWorkflow(w)}
                    className={`py-3 rounded-xl border text-sm font-bold transition-all ${
                      workflow === w
                        ? 'bg-indigo-600 border-indigo-600 text-white shadow-lg shadow-indigo-600/20'
                        : 'bg-white border-slate-200 text-slate-600 hover:border-indigo-600/30'
                    }`}
                  >
                    {w.toUpperCase()}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex gap-4 pt-4">
              <button
                type="button"
                onClick={onClose}
                className="flex-1 px-6 py-3 border border-slate-200 rounded-xl font-bold text-slate-600 hover:bg-slate-50 transition-all"
              >
                取消
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex-3 bg-indigo-600 hover:bg-indigo-500 text-white px-8 py-3 rounded-xl font-bold shadow-lg shadow-indigo-600/20 active:scale-95 transition-all disabled:opacity-50"
              >
                {loading ? '提交中...' : (mode === 'create' ? '确认创建' : '确认修改')}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
