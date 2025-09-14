import React, { useState, useEffect } from 'react';
import { getInfrastructureHealth } from '../lib/api';

const StatusIndicator = ({ status, name, error, responseTime }) => {
  const getStatusColor = (status) => {
    switch (status) {
      case 'healthy': return 'bg-green-900/20 text-green-400 border-green-700';
      case 'degraded': return 'bg-yellow-900/20 text-yellow-400 border-yellow-700';
      case 'down': return 'bg-red-900/20 text-red-400 border-red-700';
      default: return 'bg-gray-700 text-gray-300 border-gray-600';
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'healthy':
        return (
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
          </svg>
        );
      case 'degraded':
        return (
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
          </svg>
        );
      case 'down':
        return (
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
          </svg>
        );
      default:
        return (
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-3a1 1 0 00-.867.5 1 1 0 11-1.731-1A3 3 0 0113 8a3.001 3.001 0 01-2 2.83V11a1 1 0 11-2 0v-1a1 1 0 011-1 1 1 0 100-2zm0 8a1 1 0 100-2 1 1 0 000 2z" clipRule="evenodd" />
          </svg>
        );
    }
  };

  return (
    <div className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium border ${getStatusColor(status)}`}>
      {getStatusIcon(status)}
      <span className="ml-2 capitalize">{name}</span>
      {responseTime && (
        <span className="ml-2 text-xs opacity-75">({responseTime}ms)</span>
      )}
      {error && status === 'down' && (
        <span className="ml-2 text-xs opacity-75" title={error}>⚠️</span>
      )}
    </div>
  );
};

const InfrastructureStatus = ({ compact = false }) => {
  const [infraHealth, setInfraHealth] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchInfraHealth = async () => {
      try {
        setLoading(true);
        const health = await getInfrastructureHealth();
        setInfraHealth(health);
        setError(null);
      } catch (err) {
        console.error('Failed to fetch infrastructure health:', err);
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchInfraHealth();
    // Refresh every 30 seconds
    const interval = setInterval(fetchInfraHealth, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className={compact ? "text-sm text-gray-400" : "bg-gray-800 rounded-lg border border-gray-700 p-4"}>
        <div className="animate-pulse flex items-center">
          <div className="w-4 h-4 bg-gray-600 rounded-full mr-2"></div>
          <div className="h-4 bg-gray-600 rounded w-32"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={compact ? "text-sm text-red-400" : "bg-gray-800 rounded-lg border border-gray-700 p-4"}>
        <div className="flex items-center text-red-400">
          <svg className="w-4 h-4 mr-2" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
          </svg>
          Infrastructure status unavailable
        </div>
      </div>
    );
  }

  if (!infraHealth) return null;

  const getOverallStatusColor = (status) => {
    switch (status) {
      case 'healthy': return 'text-green-400';
      case 'degraded': return 'text-yellow-400';
      case 'down': return 'text-red-400';
      default: return 'text-gray-400';
    }
  };

  if (compact) {
    return (
      <div className="flex items-center text-sm">
        <div className={`font-medium ${getOverallStatusColor(infraHealth.overall_status)}`}>
          Infrastructure: {infraHealth.overall_status}
        </div>
        <div className="ml-2 text-gray-400">
          ({infraHealth.healthy_services}/{infraHealth.total_services} healthy)
        </div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800 rounded-lg border border-gray-700 p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-100">Infrastructure Status</h3>
        <div className={`text-sm font-medium ${getOverallStatusColor(infraHealth.overall_status)}`}>
          Overall: {infraHealth.overall_status.toUpperCase()}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
        {Object.entries(infraHealth.services || {}).map(([serviceName, service]) => (
          <div key={serviceName} className="flex items-center justify-between p-3 bg-gray-700 rounded-lg">
            <div>
              <div className="font-medium text-gray-100 capitalize">
                {serviceName.replace('_', ' ')}
              </div>
              {service.error && (
                <div className="text-xs text-gray-400 mt-1" title={service.error}>
                  {service.error.length > 50 ? service.error.substring(0, 50) + '...' : service.error}
                </div>
              )}
            </div>
            <StatusIndicator 
              status={service.status} 
              name={service.status}
              error={service.error}
              responseTime={service.response_time_ms}
            />
          </div>
        ))}
      </div>

      <div className="flex items-center justify-between text-sm text-gray-400">
        <div>
          {infraHealth.healthy_services} healthy, {infraHealth.degraded_services} degraded, {infraHealth.down_services} down
        </div>
        <div>
          Last checked: {new Date(infraHealth.last_checked).toLocaleTimeString()}
        </div>
      </div>

      {infraHealth.overall_status !== 'healthy' && (
        <div className="mt-4 p-3 bg-yellow-900/20 border border-yellow-700 rounded-lg">
          <div className="flex items-start">
            <svg className="w-5 h-5 text-yellow-400 mt-0.5 mr-2" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
            </svg>
            <div>
              <div className="text-sm font-medium text-yellow-400">
                Infrastructure Issues Detected
              </div>
              <div className="text-sm text-yellow-300 mt-1">
                Some services are experiencing issues. Job execution and tracking may be affected. 
                Please try again in a few minutes.
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default InfrastructureStatus;
