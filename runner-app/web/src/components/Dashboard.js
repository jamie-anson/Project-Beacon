import React, { useState, useEffect } from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line, PieChart, Pie, Cell } from 'recharts';
import { Activity, Zap, Globe, Clock, TrendingUp, Server, Database, Wifi } from 'lucide-react';

const COLORS = ['#0ea5e9', '#10b981', '#f59e0b', '#ef4444'];

function MetricCard({ title, value, icon: Icon, trend, color = 'text-beacon-400' }) {
  return (
    <div className="glass-effect rounded-xl p-6">
      <div className="flex items-center justify-between mb-4">
        <div className={`p-3 rounded-lg bg-slate-700/50`}>
          <Icon className={`w-6 h-6 ${color}`} />
        </div>
        {trend && (
          <div className={`flex items-center gap-1 text-sm ${trend > 0 ? 'text-green-400' : 'text-red-400'}`}>
            <TrendingUp className="w-4 h-4" />
            {Math.abs(trend)}%
          </div>
        )}
      </div>
      <h3 className="text-2xl font-bold text-white mb-1">{value}</h3>
      <p className="text-slate-400 text-sm">{title}</p>
    </div>
  );
}

function RegionStatus({ region, status, executions, avgDuration }) {
  const statusColor = status === 'healthy' ? 'bg-green-500' : 
                     status === 'warning' ? 'bg-yellow-500' : 'bg-red-500';
  
  return (
    <div className="glass-effect rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3">
          <div className={`w-3 h-3 rounded-full ${statusColor}`}></div>
          <h4 className="font-semibold text-white">{region}</h4>
        </div>
        <Globe className="w-5 h-5 text-slate-400" />
      </div>
      <div className="space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-slate-400">Executions</span>
          <span className="text-white font-medium">{executions}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-slate-400">Avg Duration</span>
          <span className="text-white font-medium">{avgDuration}s</span>
        </div>
      </div>
    </div>
  );
}

function Dashboard({ systemHealth }) {
  const [metrics, setMetrics] = useState(null);
  const [executions, setExecutions] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        // Fetch metrics summary
        const metricsResponse = await fetch('/api/v1/metrics/summary');
        const metricsData = await metricsResponse.json();
        setMetrics(metricsData);

        // Fetch recent executions
        const executionsResponse = await fetch('/api/v1/executions');
        const executionsData = await executionsResponse.json();
        setExecutions(executionsData.executions || []);
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 15000); // Update every 15s
    return () => clearInterval(interval);
  }, []);

  // Mock data for demonstration
  const mockExecutionData = [
    { time: '00:00', US: 12, EU: 8, APAC: 15 },
    { time: '04:00', US: 19, EU: 12, APAC: 22 },
    { time: '08:00', US: 25, EU: 18, APAC: 28 },
    { time: '12:00', US: 32, EU: 25, APAC: 35 },
    { time: '16:00', US: 28, EU: 22, APAC: 30 },
    { time: '20:00', US: 22, EU: 16, APAC: 25 },
  ];

  const mockRegionData = [
    { name: 'US', value: 45, color: '#0ea5e9' },
    { name: 'EU', value: 30, color: '#10b981' },
    { name: 'APAC', value: 25, color: '#f59e0b' },
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-8 h-8 border-4 border-beacon-400 border-t-transparent rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Metrics Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <MetricCard
          title="Total Executions"
          value="1,247"
          icon={Activity}
          trend={12}
          color="text-beacon-400"
        />
        <MetricCard
          title="Success Rate"
          value="98.5%"
          icon={Zap}
          trend={2}
          color="text-green-400"
        />
        <MetricCard
          title="Avg Duration"
          value="9.2s"
          icon={Clock}
          trend={-5}
          color="text-yellow-400"
        />
        <MetricCard
          title="Active Regions"
          value="3"
          icon={Globe}
          color="text-purple-400"
        />
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Execution Timeline */}
        <div className="glass-effect rounded-xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Execution Timeline</h3>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={mockExecutionData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
              <XAxis dataKey="time" stroke="#9ca3af" />
              <YAxis stroke="#9ca3af" />
              <Tooltip 
                contentStyle={{ 
                  backgroundColor: '#1f2937', 
                  border: '1px solid #374151',
                  borderRadius: '8px'
                }} 
              />
              <Line type="monotone" dataKey="US" stroke="#0ea5e9" strokeWidth={2} />
              <Line type="monotone" dataKey="EU" stroke="#10b981" strokeWidth={2} />
              <Line type="monotone" dataKey="APAC" stroke="#f59e0b" strokeWidth={2} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Region Distribution */}
        <div className="glass-effect rounded-xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Region Distribution</h3>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie
                data={mockRegionData}
                cx="50%"
                cy="50%"
                outerRadius={100}
                fill="#8884d8"
                dataKey="value"
                label={({ name, value }) => `${name}: ${value}%`}
              >
                {mockRegionData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.color} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Region Status and System Health */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Region Status */}
        <div className="lg:col-span-2">
          <h3 className="text-lg font-semibold text-white mb-4">Region Status</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <RegionStatus region="US East" status="healthy" executions={342} avgDuration={8.7} />
            <RegionStatus region="EU West" status="healthy" executions={298} avgDuration={9.1} />
            <RegionStatus region="APAC" status="warning" executions={267} avgDuration={10.2} />
          </div>
        </div>

        {/* System Health */}
        <div className="glass-effect rounded-xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">System Health</h3>
          <div className="space-y-4">
            {systemHealth?.components && Object.entries(systemHealth.components).map(([component, status]) => {
              const getIcon = (comp) => {
                if (comp.includes('postgres')) return Database;
                if (comp.includes('redis')) return Server;
                if (comp.includes('yagna')) return Wifi;
                return Activity;
              };
              
              const Icon = getIcon(component);
              const statusValue = typeof status === 'object' ? status.status : status;
              const statusColor = statusValue === 'ready' ? 'text-green-400' : 
                                statusValue === 'error' ? 'text-red-400' : 'text-yellow-400';
              
              return (
                <div key={component} className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <Icon className="w-4 h-4 text-slate-400" />
                    <span className="text-sm text-slate-300 capitalize">
                      {component.replace(/_/g, ' ')}
                    </span>
                  </div>
                  <span className={`text-sm font-medium ${statusColor}`}>
                    {statusValue}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

export default Dashboard;
