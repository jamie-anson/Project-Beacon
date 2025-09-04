import React, { useState, useEffect } from 'react';
import { Play, Pause, Trash2, Plus, RefreshCw, Clock, CheckCircle, XCircle, AlertCircle } from 'lucide-react';

function JobCard({ job, onExecute, onDelete }) {
  const getStatusIcon = (status) => {
    switch (status?.toLowerCase()) {
      case 'completed':
        return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-400" />;
      case 'running':
        return <RefreshCw className="w-5 h-5 text-blue-400 animate-spin" />;
      default:
        return <Clock className="w-5 h-5 text-yellow-400" />;
    }
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <div className="glass-effect rounded-lg p-6 hover:bg-slate-700/30 transition-colors">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          {getStatusIcon(job.status)}
          <div>
            <h3 className="font-semibold text-white">{job.job_id}</h3>
            <p className="text-sm text-slate-400">
              Created: {formatDate(job.created_at)}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => onExecute(job.job_id)}
            className="p-2 bg-beacon-600 hover:bg-beacon-700 rounded-lg transition-colors"
            title="Execute Job"
          >
            <Play className="w-4 h-4 text-white" />
          </button>
          <button
            onClick={() => onDelete(job.job_id)}
            className="p-2 bg-red-600 hover:bg-red-700 rounded-lg transition-colors"
            title="Delete Job"
          >
            <Trash2 className="w-4 h-4 text-white" />
          </button>
        </div>
      </div>
      
      <div className="text-sm text-slate-300">
        <div className="flex justify-between items-center">
          <span>Status:</span>
          <span className={`font-medium ${
            job.status === 'completed' ? 'text-green-400' :
            job.status === 'failed' ? 'text-red-400' :
            job.status === 'running' ? 'text-blue-400' : 'text-yellow-400'
          }`}>
            {job.status || 'pending'}
          </span>
        </div>
      </div>
    </div>
  );
}

function CreateJobModal({ isOpen, onClose, onSubmit }) {
  const [jobSpec, setJobSpec] = useState('');
  const [loading, setLoading] = useState(false);

  const defaultJobSpec = {
    "id": "who-are-you-benchmark-v1",
    "version": "1.0.0",
    "name": "Who are you? Benchmark",
    "description": "Simple identity benchmark across multiple regions",
    "benchmark": {
      "type": "identity",
      "container": {
        "image": "alpine",
        "command": ["sh", "-c", "echo 'I am an AI assistant running on Golem Network'"],
        "environment": {
          "BENCHMARK_TYPE": "identity"
        }
      }
    },
    "constraints": {
      "regions": ["US", "EU", "APAC"],
      "min_regions": 3,
      "timeout": "10m",
      "max_cost": 1.0
    },
    "signature": "",
    "public_key": ""
  };

  useEffect(() => {
    if (isOpen) {
      setJobSpec(JSON.stringify(defaultJobSpec, null, 2));
    }
  }, [isOpen]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    
    try {
      const parsedSpec = JSON.parse(jobSpec);
      await onSubmit(parsedSpec);
      onClose();
      setJobSpec('');
    } catch (error) {
      alert('Invalid JSON: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="glass-effect rounded-xl max-w-2xl w-full max-h-[80vh] overflow-hidden">
        <div className="p-6 border-b border-slate-700">
          <h2 className="text-xl font-semibold text-white">Create New Job</h2>
          <p className="text-slate-400 text-sm mt-1">Define a JobSpec for multi-region execution</p>
        </div>
        
        <form onSubmit={handleSubmit} className="p-6">
          <div className="mb-4">
            <label className="block text-sm font-medium text-slate-300 mb-2">
              JobSpec JSON
            </label>
            <textarea
              value={jobSpec}
              onChange={(e) => setJobSpec(e.target.value)}
              className="w-full h-64 bg-slate-800 border border-slate-600 rounded-lg p-3 text-white font-mono text-sm resize-none focus:ring-2 focus:ring-beacon-500 focus:border-transparent"
              placeholder="Enter JobSpec JSON..."
              required
            />
          </div>
          
          <div className="flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 bg-slate-600 hover:bg-slate-700 text-white rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 bg-beacon-600 hover:bg-beacon-700 text-white rounded-lg transition-colors disabled:opacity-50"
            >
              {loading ? 'Creating...' : 'Create Job'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function JobManager() {
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [executing, setExecuting] = useState(new Set());

  useEffect(() => {
    fetchJobs();
  }, []);

  const fetchJobs = async () => {
    try {
      const response = await fetch('/api/v1/jobs');
      const data = await response.json();
      setJobs(data.jobs || []);
    } catch (error) {
      console.error('Failed to fetch jobs:', error);
      setJobs([]);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateJob = async (jobSpec) => {
    try {
      const response = await fetch('/api/v1/jobs', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(jobSpec),
      });

      if (response.ok) {
        await fetchJobs();
      } else {
        const error = await response.json();
        alert('Failed to create job: ' + (error.details || error.error));
      }
    } catch (error) {
      alert('Failed to create job: ' + error.message);
    }
  };

  const handleExecuteJob = async (jobId) => {
    setExecuting(prev => new Set([...prev, jobId]));
    
    try {
      const response = await fetch(`/api/v1/jobs/${jobId}/execute`, {
        method: 'POST',
      });

      if (response.ok) {
        const result = await response.json();
        alert(`Execution started for job ${jobId}`);
        await fetchJobs();
      } else {
        const error = await response.json();
        alert('Execution failed: ' + (error.details || error.error));
      }
    } catch (error) {
      alert('Execution failed: ' + error.message);
    } finally {
      setExecuting(prev => {
        const newSet = new Set(prev);
        newSet.delete(jobId);
        return newSet;
      });
    }
  };

  const handleDeleteJob = async (jobId) => {
    if (!confirm(`Are you sure you want to delete job ${jobId}?`)) {
      return;
    }

    try {
      const response = await fetch(`/api/v1/jobs/${jobId}`, {
        method: 'DELETE',
      });

      if (response.ok) {
        await fetchJobs();
      } else {
        const error = await response.json();
        alert('Failed to delete job: ' + (error.details || error.error));
      }
    } catch (error) {
      alert('Failed to delete job: ' + error.message);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-8 h-8 border-4 border-beacon-400 border-t-transparent rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-white">Job Management</h2>
          <p className="text-slate-400 mt-1">Create and manage benchmark execution jobs</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-beacon-600 hover:bg-beacon-700 text-white rounded-lg transition-colors"
        >
          <Plus className="w-5 h-5" />
          Create Job
        </button>
      </div>

      {jobs.length === 0 ? (
        <div className="glass-effect rounded-xl p-12 text-center">
          <AlertCircle className="w-12 h-12 text-slate-400 mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-white mb-2">No Jobs Found</h3>
          <p className="text-slate-400 mb-6">
            Create your first job to start running multi-region benchmarks
          </p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-6 py-3 bg-beacon-600 hover:bg-beacon-700 text-white rounded-lg transition-colors"
          >
            Create Your First Job
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {jobs.map((job) => (
            <JobCard
              key={job.job_id}
              job={job}
              onExecute={handleExecuteJob}
              onDelete={handleDeleteJob}
            />
          ))}
        </div>
      )}

      <CreateJobModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSubmit={handleCreateJob}
      />
    </div>
  );
}

export default JobManager;
