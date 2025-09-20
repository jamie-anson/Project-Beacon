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
import CrossRegionDiffView from './pages/CrossRegionDiffView.jsx';
import Executions from './pages/Executions.jsx';
import ExecutionDetail from './pages/ExecutionDetail.jsx';
import DemoResults from './pages/DemoResults.jsx';
import useWs from './state/useWs.js';
import { BUILD_CID, BUILD_COMMIT, shortCommit } from './lib/buildInfo.js';

function Layout({ children }) {
  const { connected, error: wsError, retries, nextDelayMs } = useWs('/ws');
  const [mobileMenuOpen, setMobileMenuOpen] = React.useState(false);
  
  return (
    <div className="min-h-screen bg-gray-900 text-gray-100">
      <header className="border-b border-gray-700 bg-gray-800">
        <div className="max-w-6xl mx-auto px-4 py-3">
          <div className="flex items-center justify-between">
            <Link to="/" className="flex items-center gap-2 font-semibold">
              <img src="/images/Icon.webp" alt="Project Beacon" className="w-8 h-8" />
              <span className="text-gray-400">Portal</span>
            </Link>
            
            {/* Desktop Navigation */}
            <nav className="hidden md:flex items-center gap-4 text-sm">
              <NavLink to="/" end className={({isActive}) => isActive ? 'text-orange-300 font-medium' : 'text-gray-300 hover:text-white'}>Home</NavLink>
              <NavLink to="/questions" className={({isActive}) => isActive ? 'text-orange-300 font-medium' : 'text-gray-300 hover:text-white'}>Questions</NavLink>
              <NavLink to="/ais" className={({isActive}) => isActive ? 'text-orange-300 font-medium' : 'text-gray-300 hover:text-white'}>Models</NavLink>
              <NavLink to="/bias-detection" className={({isActive}) => isActive ? 'text-orange-300 font-medium' : 'text-gray-300 hover:text-white'}>Bias Detection</NavLink>
              <NavLink to="/dashboard" className={({isActive}) => isActive ? 'text-orange-300 font-medium' : 'text-gray-300 hover:text-white'}>Dashboard</NavLink>
              <NavLink to="/demo-results" className={({isActive}) => isActive ? 'text-orange-300 font-medium' : 'text-gray-300 hover:text-white'}>Demo Results</NavLink>
            </nav>
            
            {/* Mobile menu button */}
            <button 
              className="md:hidden p-2 rounded-md text-gray-300 hover:text-white hover:bg-gray-700"
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
                className={`inline-flex items-center gap-1 px-2 py-1 rounded-full ${connected ? 'bg-green-900/50 text-green-400' : 'bg-red-900/50 text-red-400'}`}
                aria-label="Live updates connection status"
                title={wsError ? `WebSocket error: ${wsError.message || String(wsError)}${retries ? ` • retries: ${retries}, next: ${Math.round(nextDelayMs/1000)}s` : ''}` : 'Live updates use a real-time connection to the Runner'}
              >
                <span className={`w-2 h-2 rounded-full ${connected ? 'bg-green-400' : 'bg-red-400'}`}></span>
                {connected ? 'Live updates: Online' : (wsError ? 'Live updates: Error' : 'Live updates: Offline')}
              </span>
              <NavLink to="/settings" className={({isActive}) => isActive ? 'text-orange-300' : 'text-gray-400 hover:text-white'} title="Settings">Settings</NavLink>
            </div>
          </div>
          
          {/* Mobile Navigation Menu */}
          {mobileMenuOpen && (
            <div className="md:hidden mt-3 pt-3 border-t border-gray-700">
              <nav className="flex flex-col gap-2">
                <NavLink 
                  to="/" 
                  end 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-gray-700 text-orange-300 font-medium' : 'text-gray-300 hover:text-white hover:bg-gray-800'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Home
                </NavLink>
                <NavLink 
                  to="/questions" 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-gray-700 text-orange-300 font-medium' : 'text-gray-300 hover:text-white hover:bg-gray-800'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Questions
                </NavLink>
                <NavLink 
                  to="/ais" 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-gray-700 text-orange-300 font-medium' : 'text-gray-300 hover:text-white hover:bg-gray-800'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Models
                </NavLink>
                <NavLink 
                  to="/bias-detection" 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-gray-700 text-orange-300 font-medium' : 'text-gray-300 hover:text-white hover:bg-gray-800'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Bias Detection
                </NavLink>
                <NavLink 
                  to="/dashboard" 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-gray-700 text-orange-300 font-medium' : 'text-gray-300 hover:text-white hover:bg-gray-800'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Dashboard
                </NavLink>
                <NavLink 
                  to="/demo-results" 
                  className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-gray-700 text-orange-300 font-medium' : 'text-gray-300 hover:text-white hover:bg-gray-800'}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Demo Results
                </NavLink>
                <div className="border-t border-gray-700 mt-2 pt-2">
                  <div className="px-3 py-2">
                    <span
                      className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs ${connected ? 'bg-green-900/50 text-green-400' : 'bg-red-900/50 text-red-400'}`}
                    >
                      <span className={`w-2 h-2 rounded-full ${connected ? 'bg-green-400' : 'bg-red-400'}`}></span>
                      {connected ? 'Online' : 'Offline'}
                    </span>
                  </div>
                  <NavLink 
                    to="/settings" 
                    className={({isActive}) => `block px-3 py-2 rounded-md text-sm ${isActive ? 'bg-gray-700 text-orange-300 font-medium' : 'text-gray-300 hover:text-white hover:bg-gray-800'}`}
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
      <footer className="border-t border-gray-700 bg-gray-800">
        <div className="max-w-6xl mx-auto px-4 py-3 text-xs text-gray-400 flex flex-wrap items-center justify-between gap-2">
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
        <Route path="/results/:jobId/diffs" element={<CrossRegionDiffView />} />
        <Route path="/world" element={<WorldView />} />
        <Route path="/ais" element={<AIs />} />
        <Route path="/executions" element={<Executions />} />
        <Route path="/executions/:id" element={<ExecutionDetail />} />
        <Route path="/demo-results" element={<DemoResults />} />
        <Route path="/results" element={<Diffs />} />
        <Route path="/diffs" element={<Diffs />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Layout>
  );
}
