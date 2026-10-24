/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import React from 'react';
import { NavLink, useNavigate, useParams } from 'react-router-dom';
import {
  Settings,
  LayoutGrid,
  FileText,
  Palette,
  Box,
  Clapperboard,
  Layers,
  Film,
  LogOut,
  Trash2
} from 'lucide-react';
import { cn } from '../lib/utils';
import { useProject } from '../contexts/ProjectContext';
import { api } from '../lib/api';

export default function Layout({ children }: { children: React.ReactNode }) {
  const { projectId } = useParams();
  const navigate = useNavigate();

  const handleLogout = () => {
    localStorage.removeItem('token');
    navigate('/login');
  };

  const navItems = [
    { icon: LayoutGrid, label: '项目列表', path: '/projects' },
    { icon: Settings, label: '系统设置', path: '/settings' },
  ];

  return (
    <div className="flex h-screen overflow-hidden bg-slate-50">
      {/* Sidebar */}
      <aside className="w-64 flex-shrink-0 bg-slate-950 text-slate-300 flex flex-col border-r border-slate-800">
        <div className="p-6 flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center text-white font-bold">
            LT
          </div>
          <span className="font-semibold text-lg tracking-tight text-white">LTSeries Studio</span>
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-2 space-y-6">
          <div>
            <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider px-2 mb-2">系统</div>
            <nav className="space-y-1">
              {navItems.map(item => (
                <NavLink
                  key={item.path}
                  to={item.path}
                  className={({ isActive }) => cn(
                    "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                    isActive ? "bg-indigo-600/10 text-indigo-400" : "hover:bg-slate-900 hover:text-white"
                  )}
                >
                  <item.icon size={18} />
                  {item.label}
                </NavLink>
              ))}
            </nav>
          </div>

          {projectId && (
            <div className="space-y-6">
              <div>
                <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest px-2 mb-3">AI 策划 / 规划</div>
                <nav className="space-y-1">
                  {[
                    { icon: FileText, label: '剧本原文', path: `/projects/${projectId}/script` },
                    { icon: Layers, label: '分集策划', path: `/projects/${projectId}/episodes` },
                    { icon: Palette, label: '美术方向', path: `/projects/${projectId}/art` },
                  ].map(item => (
                    <NavLink
                      key={item.path}
                      to={item.path}
                      className={({ isActive }) => cn(
                        "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                        isActive ? "bg-white/10 text-white" : "hover:bg-slate-900 hover:text-white"
                      )}
                    >
                      <item.icon size={18} />
                      {item.label}
                    </NavLink>
                  ))}
                </nav>
              </div>

              <div>
                <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest px-2 mb-3">资产库 / 生产</div>
                <nav className="space-y-1">
                  {[
                    { icon: Box, label: '核心资产库', path: `/projects/${projectId}/assets` },
                    { icon: Clapperboard, label: '分镜制作', path: `/projects/${projectId}/storyboard` },
                  ].map(item => (
                    <NavLink
                      key={item.path}
                      to={item.path}
                      className={({ isActive }) => cn(
                        "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                        isActive ? "bg-white/10 text-white" : "hover:bg-slate-900 hover:text-white"
                      )}
                    >
                      <item.icon size={18} />
                      {item.label}
                    </NavLink>
                  ))}
                </nav>
              </div>

              <div>
                <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest px-2 mb-3">成品 / 交付</div>
                <nav className="space-y-1">
                  {[
                    { icon: Film, label: '成片管理', path: `/projects/${projectId}/assembly` },
                  ].map(item => (
                    <NavLink
                      key={item.path}
                      to={item.path}
                      className={({ isActive }) => cn(
                        "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                        isActive ? "bg-white/10 text-white" : "hover:bg-slate-900 hover:text-white"
                      )}
                    >
                      <item.icon size={18} />
                      {item.label}
                    </NavLink>
                  ))}
                </nav>
              </div>

              <div>
                <nav className="space-y-1 border-t border-slate-900 pt-4">
                  <NavLink
                    to={`/projects/${projectId}/settings`}
                    className={({ isActive }) => cn(
                      "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                      isActive ? "bg-white/10 text-white" : "hover:bg-slate-900 hover:text-white text-slate-500"
                    )}
                  >
                    <Settings size={18} />
                    项目全局设置
                  </NavLink>
                </nav>
              </div>
            </div>
          )}
        </div>

        <div className="p-4 border-t border-slate-800 mt-auto">
          <button
            onClick={handleLogout}
            className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium text-slate-400 hover:bg-slate-900 hover:text-white transition-colors"
          >
            <LogOut size={18} />
            退出登錄
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-y-auto min-w-0 bg-slate-50 relative">
        <div className="p-8 max-w-7xl mx-auto">
          {children}
        </div>
      </main>

      {/* Task Overlay (Global mini status) */}
      <TaskMonitor />
    </div>
  );
}

function TaskMonitor() {
  const { tasks, refreshTasks } = useProject();
  const activeTasks = tasks.filter(t => t.status === 'processing' || t.status === 'pending');
  const visibleTasks = tasks.slice(0, 8);

  if (visibleTasks.length === 0) return null;

  const deleteTask = async (id: string) => {
    if (!window.confirm('确定删除这条任务记录吗？')) return;
    try {
      await api.deleteTask(id);
      await refreshTasks();
    } catch (error: any) {
      alert(error?.message || '删除任务失败');
    }
  };

  return (
    <div className="fixed bottom-6 right-6 z-50">
      <div className="bg-white rounded-xl shadow-xl border border-slate-200 overflow-hidden w-[340px]">
        <div className="bg-slate-50 border-b border-slate-100 px-4 py-2 flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-600">任务记录 ({activeTasks.length} 进行中 / {tasks.length} 总计)</span>
        </div>
        <div className="max-h-60 overflow-y-auto p-2 space-y-1">
          {visibleTasks.map(task => (
            <div key={task.id} className="p-3 rounded-lg border border-slate-100 bg-slate-50/50">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-xs font-medium text-slate-700 capitalize">{task.type.replace('_', ' ')}</span>
                <div className="flex items-center gap-2">
                  <span className={cn(
                    "text-[10px] font-bold px-1.5 py-0.5 rounded uppercase",
                    task.status === 'completed' ? "bg-emerald-50 text-emerald-700" :
                    task.status === 'failed' ? "bg-red-50 text-red-700" :
                    "bg-indigo-50 text-indigo-700"
                  )}>
                    {task.status}
                  </span>
                  <button onClick={() => deleteTask(task.id)} className="text-slate-300 hover:text-red-500">
                    <Trash2 size={13} />
                  </button>
                </div>
              </div>
              <div className="w-full h-1.5 bg-slate-200 rounded-full overflow-hidden">
                <div
                  className={cn("h-full transition-all duration-500", task.status === 'failed' ? "bg-red-500" : task.status === 'completed' ? "bg-emerald-500" : "bg-indigo-600")}
                  style={{ width: `${task.progress > 0 ? task.progress : 10}%` }}
                />
              </div>
              {task.error && <p className="mt-2 text-[11px] leading-4 text-red-600 line-clamp-2">{task.error}</p>}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
