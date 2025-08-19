import React from 'react';
import { Routes, Route, Link, NavLink } from 'react-router-dom';
import Dashboard from './pages/Dashboard.jsx';
import Jobs from './pages/Jobs.jsx';
import JobDetail from './pages/JobDetail.jsx';
import CreateJob from './pages/CreateJob.jsx';
import TemplateViewer from './pages/TemplateViewer.jsx';
import Executions from './pages/Executions.jsx';
import Diffs from './pages/Diffs.jsx';
import Settings from './pages/Settings.jsx';
import useWs from './state/useWs.js';
import { BUILD_CID, BUILD_COMMIT, shortCommit } from './lib/buildInfo.js';

function Layout({ children }) {
  const { connected, error: wsError, retries, nextDelayMs } = useWs('/ws');
  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      <header className="border-b bg-white">
        <div className="max-w-6xl mx-auto px-4 py-3 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2 font-semibold">
            <span>Project Beacon</span>
            <span className="text-slate-400">Portal</span>
          </Link>
          <nav className="flex items-center gap-4 text-sm">
            <NavLink to="/" end className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Dashboard</NavLink>
            <NavLink to="/jobs" className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Jobs</NavLink>
            <NavLink to="/jobs/who-are-you" className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Who are you?</NavLink>
            <NavLink to="/executions" className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Executions</NavLink>
            <NavLink to="/diffs" className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Diffs</NavLink>
          </nav>
          <div className="ml-4 flex items-center gap-2 text-xs">
            <span
              className={`inline-flex items-center gap-1 px-2 py-1 rounded-full ${connected ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}
              aria-label="Live updates connection status"
              title={wsError ? `WebSocket error: ${wsError.message || String(wsError)}${retries ? ` • retries: ${retries}, next: ${Math.round(nextDelayMs/1000)}s` : ''}` : 'Live updates use a real-time connection to the Runner'}
            >
              <span className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`}></span>
              {connected ? 'Live updates: Online' : (wsError ? 'Live updates: Error' : 'Live updates: Offline')}
            </span>
            <NavLink to="/settings" className={({isActive}) => isActive ? 'text-beacon-600' : 'text-slate-500 hover:text-slate-900'} title="Settings">Settings</NavLink>
          </div>
        </div>
      </header>
      <main className="max-w-6xl mx-auto px-4 py-6">
        {children}
      </main>
      <footer className="border-t bg-white">
        <div className="max-w-6xl mx-auto px-4 py-3 text-xs text-slate-500 flex flex-wrap items-center justify-between gap-2">
          <span>
            Build CID: <a className="font-mono underline decoration-dotted" href={`https://ipfs.io/ipfs/${BUILD_CID}`} target="_blank" rel="noreferrer">{BUILD_CID}</a>
          </span>
          <span>
            Commit: <code className="font-mono">{shortCommit(BUILD_COMMIT)}</code>
          </span>
        </div>
      </footer>
    </div>
  );
}

export default function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/jobs" element={<Jobs />} />
        <Route path="/jobs/new" element={<CreateJob />} />
        <Route path="/jobs/:id" element={<JobDetail />} />
        <Route path="/jobs/who-are-you" element={<TemplateViewer />} />
        <Route path="/executions" element={<Executions />} />
        <Route path="/diffs" element={<Diffs />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Layout>
  );
}
