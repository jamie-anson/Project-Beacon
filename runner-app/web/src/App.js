import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation } from 'react-router-dom';
import { Activity, BarChart3, Settings, Zap, Globe, Clock, CheckCircle, XCircle, Wifi, WifiOff } from 'lucide-react';
import Dashboard from './components/Dashboard';
import ExecutionMonitor from './components/ExecutionMonitor';
import JobManager from './components/JobManager';
import DiffViewer from './components/DiffViewer';
import NotificationToast from './components/NotificationToast';
import useWebSocket from './hooks/useWebSocket';

function Navigation() {
  const location = useLocation();
  
  const navItems = [
    { path: '/', icon: BarChart3, label: 'Dashboard' },
    { path: '/jobs', icon: Zap, label: 'Jobs' },
    { path: '/executions', icon: Activity, label: 'Executions' },
    { path: '/diffs', icon: Globe, label: 'Diffs' },
  ];

  return (
    <nav className="bg-slate-800 border-r border-slate-700 w-64 min-h-screen p-4">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-beacon-400 flex items-center gap-2">
          <div className="w-8 h-8 bg-gradient-to-br from-beacon-400 to-beacon-600 rounded-lg flex items-center justify-center">
            <Globe className="w-5 h-5 text-white" />
          </div>
          Project Beacon
        </h1>
        <p className="text-slate-400 text-sm mt-1">Multi-region Golem execution</p>
      </div>
      
      <ul className="space-y-2">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = location.pathname === item.path;
          
          return (
            <li key={item.path}>
              <Link
                to={item.path}
                className={`flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                  isActive
                    ? 'bg-beacon-600 text-white'
                    : 'text-slate-300 hover:bg-slate-700 hover:text-white'
                }`}
              >
                <Icon className="w-5 h-5" />
                {item.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}

function StatusIndicator({ status, className = "" }) {
  const getStatusConfig = (status) => {
    switch (status?.toLowerCase()) {
      case 'healthy':
      case 'ready':
      case 'completed':
        return { icon: CheckCircle, color: 'text-green-400', bg: 'bg-green-400/20' };
      case 'error':
      case 'failed':
        return { icon: XCircle, color: 'text-red-400', bg: 'bg-red-400/20' };
      default:
        return { icon: Clock, color: 'text-yellow-400', bg: 'bg-yellow-400/20' };
    }
  };

  const { icon: Icon, color, bg } = getStatusConfig(status);
  
  return (
    <div className={`flex items-center gap-2 px-3 py-1 rounded-full ${bg} ${className}`}>
      <Icon className={`w-4 h-4 ${color}`} />
      <span className={`text-sm font-medium ${color}`}>{status || 'Unknown'}</span>
    </div>
  );
}

function App() {
  const [systemHealth, setSystemHealth] = useState(null);
  const [loading, setLoading] = useState(true);
  const [notifications, setNotifications] = useState([]);

  // WebSocket connection for real-time updates
  const handleWebSocketMessage = (message) => {
    console.log('WebSocket message received:', message);
    
    // Add notification for job events
    if (message.type.startsWith('job_')) {
      const notification = {
        id: Date.now(),
        type: message.type,
        data: message.data,
        timestamp: new Date().toISOString()
      };
      
      setNotifications(prev => [notification, ...prev.slice(0, 9)]); // Keep last 10
      
      // Auto-remove notification after 5 seconds
      setTimeout(() => {
        setNotifications(prev => prev.filter(n => n.id !== notification.id));
      }, 5000);
    }
  };

  const wsUrl = `ws://${window.location.hostname}:8090/ws`;
  const { isConnected, connectionError } = useWebSocket(wsUrl, handleWebSocketMessage);

  useEffect(() => {
    const fetchHealth = async () => {
      try {
        const response = await fetch('/api/v1/health');
        const data = await response.json();
        setSystemHealth(data);
      } catch (error) {
        console.error('Failed to fetch system health:', error);
        setSystemHealth({ status: 'error', components: {} });
      } finally {
        setLoading(false);
      }
    };

    fetchHealth();
    const interval = setInterval(fetchHealth, 30000); // Update every 30s
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 border-4 border-beacon-400 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-slate-400">Loading Project Beacon...</p>
        </div>
      </div>
    );
  }

  return (
    <Router>
      <div className="min-h-screen bg-slate-900 flex">
        <Navigation />
        
        <div className="flex-1 flex flex-col">
          {/* Header */}
          <header className="bg-slate-800 border-b border-slate-700 px-6 py-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-semibold text-white">
                  Multi-region Benchmark Execution
                </h2>
                <p className="text-slate-400 text-sm">
                  Monitor Golem network executions across US, EU, and APAC regions
                </p>
              </div>
              
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  {isConnected ? (
                    <Wifi className="w-4 h-4 text-green-400" title="WebSocket Connected" />
                  ) : (
                    <WifiOff className="w-4 h-4 text-red-400" title="WebSocket Disconnected" />
                  )}
                  <StatusIndicator 
                    status={systemHealth?.status} 
                    className="text-xs"
                  />
                </div>
                <div className="text-xs text-slate-400">
                  Last updated: {new Date().toLocaleTimeString()}
                </div>
              </div>
            </div>
          </header>

          {/* Main content */}
          <main className="flex-1 p-6 overflow-auto">
            <Routes>
              <Route path="/" element={<Dashboard systemHealth={systemHealth} />} />
              <Route path="/jobs" element={<JobManager />} />
              <Route path="/executions" element={<ExecutionMonitor />} />
              <Route path="/diffs" element={<DiffViewer />} />
            </Routes>
          </main>
        </div>
        
        {/* Real-time notifications */}
        <NotificationToast 
          notifications={notifications}
          onDismiss={(id) => setNotifications(prev => prev.filter(n => n.id !== id))}
        />
      </div>
    </Router>
  );
}

export default App;
