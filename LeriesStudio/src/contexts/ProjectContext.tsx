/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { Project, Task } from '../types';
import { api } from '../lib/api';

interface ProjectContextType {
  currentProject: Project | null;
  tasks: Task[];
  loading: boolean;
  error: string | null;
  refreshProject: () => Promise<void>;
  refreshTasks: () => Promise<void>;
  setCurrentProjectId: (id: string | null) => void;
}

const ProjectContext = createContext<ProjectContextType | undefined>(undefined);

export function ProjectProvider({ children }: { children: React.ReactNode }) {
  const [currentProjectId, setCurrentProjectId] = useState<string | null>(null);
  const [currentProject, setCurrentProject] = useState<Project | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshProject = useCallback(async () => {
    if (!currentProjectId) return;
    try {
      const project = await api.getProject(currentProjectId);
      setCurrentProject(project);
    } catch (err: any) {
      setError(err.message);
    }
  }, [currentProjectId]);

  const refreshTasks = useCallback(async () => {
    try {
      const taskList = await api.getTasks(currentProjectId || undefined);
      setTasks(taskList);
    } catch (err: any) {
      console.error('Failed to fetch tasks:', err);
    }
  }, [currentProjectId]);

  // Initial load
  useEffect(() => {
    if (currentProjectId) {
      setLoading(true);
      Promise.all([refreshProject(), refreshTasks()]).finally(() => setLoading(false));
    } else {
      setCurrentProject(null);
      setTasks([]);
    }
  }, [currentProjectId, refreshProject, refreshTasks]);

  // Polling for tasks and project updates
  useEffect(() => {
    if (!currentProjectId) return;

    const interval = setInterval(() => {
      refreshTasks();

      // If there are processing tasks, refresh project too to see results
      const hasActiveTasks = tasks.some(t => t.status === 'processing' || t.status === 'pending');
      if (hasActiveTasks) {
        refreshProject();
      }
    }, 3000);

    return () => clearInterval(interval);
  }, [currentProjectId, tasks, refreshTasks, refreshProject]);

  return (
    <ProjectContext.Provider value={{
      currentProject,
      tasks,
      loading,
      error,
      refreshProject,
      refreshTasks,
      setCurrentProjectId
    }}>
      {children}
    </ProjectContext.Provider>
  );
}

export function useProject() {
  const context = useContext(ProjectContext);
  if (context === undefined) {
    throw new Error('useProject must be used within a ProjectProvider');
  }
  return context;
}
