/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ProjectProvider } from './contexts/ProjectContext.tsx';
import Layout from './components/Layout.tsx';

// Pages
import LoginPage from './pages/LoginPage.tsx';
import ProjectsPage from './pages/ProjectsPage.tsx';
import SettingsPage from './pages/SettingsPage.tsx';
import ScriptPage from './pages/project/ScriptPage.tsx';
import ArtDirectionPage from './pages/project/ArtDirectionPage.tsx';
import AssetsPage from './pages/project/AssetsPage.tsx';
import StoryboardPage from './pages/project/StoryboardPage.tsx';
import AssemblyPage from './pages/project/AssemblyPage.tsx';
import EpisodesPage from './pages/project/EpisodesPage.tsx';
import ProjectSettingsPage from './pages/project/ProjectSettingsPage.tsx';

export default function App() {
  return (
    <BrowserRouter>
      <ProjectProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />

          <Route path="/" element={<Layout><Navigate to="/projects" replace /></Layout>} />

          <Route path="/projects" element={<Layout><ProjectsPage /></Layout>} />
          <Route path="/settings" element={<Layout><SettingsPage /></Layout>} />

          <Route path="/projects/:projectId" element={<Layout><Navigate to="script" replace /></Layout>} />
          <Route path="/projects/:projectId/script" element={<Layout><ScriptPage /></Layout>} />
          <Route path="/projects/:projectId/art" element={<Layout><ArtDirectionPage /></Layout>} />
          <Route path="/projects/:projectId/assets" element={<Layout><AssetsPage /></Layout>} />
          <Route path="/projects/:projectId/storyboard" element={<Layout><StoryboardPage /></Layout>} />
          <Route path="/projects/:projectId/assembly" element={<Layout><AssemblyPage /></Layout>} />
          <Route path="/projects/:projectId/episodes" element={<Layout><EpisodesPage /></Layout>} />
          <Route path="/projects/:projectId/settings" element={<Layout><ProjectSettingsPage /></Layout>} />

          <Route path="*" element={<Navigate to="/projects" replace />} />
        </Routes>
      </ProjectProvider>
    </BrowserRouter>
  );
}
