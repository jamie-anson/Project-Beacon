import React from 'react';
import { Routes, Route, Link, NavLink } from 'react-router-dom';
import Dashboard from './pages/Dashboard.jsx';
import Home from './pages/Home.jsx';
import Jobs from './pages/Jobs.jsx';
import JobDetail from './pages/JobDetail.jsx';
import CreateJob from './pages/CreateJob.jsx';
import Questions from './pages/Questions.jsx';
import WorldView from './pages/WorldView.jsx';
import AIs from './pages/AIs.jsx';
import Diffs from './pages/Diffs.jsx';
import Settings from './pages/Settings.jsx';
import BiasDetection from './pages/BiasDetection.jsx';
import Executions from './pages/Executions.jsx';
import DemoResults from './pages/DemoResults.jsx';
import useWs from './state/useWs.js';
import { BUILD_CID, BUILD_COMMIT, shortCommit } from './lib/buildInfo.js';

function Layout({ children }) {
  const { connected, error: wsError, retries, nextDelayMs } = useWs('/ws');
  const [mobileMenuOpen, setMobileMenuOpen] = React.useState(false);
  
  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      <header className="border-b bg-white">
        <div className="max-w-6xl mx-auto px-4 py-3">
          <div className="flex items-center justify-between">
            <Link to="/" className="flex items-center gap-2 font-semibold">
              <img src="/images/Icon.webp" alt="Project Beacon" className="w-8 h-8" />
              <span className="text-slate-400">Portal</span>
            </Link>
            
            {/* Desktop Navigation */}
            <nav className="hidden md:flex items-center gap-4 text-sm">
              <NavLink to="/" end className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Home</NavLink>
              <NavLink to="/questions" className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Questions</NavLink>
              <NavLink to="/ais" className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Models</NavLink>
              <NavLink to="/bias-detection" className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Bias Detection</NavLink>
              <NavLink to="/dashboard" className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Dashboard</NavLink>
              <NavLink to="/demo-results" className={({isActive}) => isActive ? 'text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900'}>Demo Results</NavLink>
            </nav>
            
            {/* Mobile menu button */}
            <button 
              className="md:hidden p-2 rounded-md text-slate-600 hover:text-slate-900 hover:bg-slate-100"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              aria-label="Toggle menu"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                {mobileMenuOpen ? (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                ) : (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                )}
              </svg>
            </button>
            {/* Desktop status and settings */}
            <div className="hidden md:flex items-center gap-2 text-xs">
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
          
          {/* Mobile Navigation Menu */}
          {mobileMenuOpen && (
            <div className="md:hidden mt-3 pt-3 border-t border-slate-200">
              <nav className="flex flex-col gap-2">
                <NavLink 
                  to="/" 
                  end 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-beacon-50 text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Home
                </NavLink>
                <NavLink 
                  to="/questions" 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-beacon-50 text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Questions
                </NavLink>
                <NavLink 
                  to="/ais" 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-beacon-50 text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Models
                </NavLink>
                <NavLink 
                  to="/bias-detection" 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-beacon-50 text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Bias Detection
                </NavLink>
                <NavLink 
                  to="/dashboard" 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-beacon-50 text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Dashboard
                </NavLink>
                <NavLink 
                  to="/demo-results" 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-beacon-50 text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Demo Results
                </NavLink>
                <div className="border-t border-slate-200 mt-2 pt-2">
                  <div className="px-3 py-2">
                    <span
                      className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs ${connected ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}
                    >
                      <span className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`}></span>
                      {connected ? 'Online' : 'Offline'}
                    </span>
                  </div>
                  <NavLink 
                    to="/settings" 
                    className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-beacon-50 text-beacon-600 font-medium' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'}`}
                    onClick={() => setMobileMenuOpen(false)}
                  >
                    Settings
                  </NavLink>
                </div>
              </nav>
            </div>
          )}
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
        <Route path="/" element={<Home />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/jobs" element={<Jobs />} />
        <Route path="/jobs/new" element={<CreateJob />} />
        <Route path="/jobs/:id" element={<JobDetail />} />
        <Route path="/questions" element={<Questions />} />
        <Route path="/bias-detection" element={<BiasDetection />} />
        <Route path="/world" element={<WorldView />} />
        <Route path="/ais" element={<AIs />} />
        <Route path="/executions" element={<Executions />} />
        <Route path="/demo-results" element={<DemoResults />} />
        <Route path="/results" element={<Diffs />} />
        <Route path="/diffs" element={<Diffs />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Layout>
  );
}
