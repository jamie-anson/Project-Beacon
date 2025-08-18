import React from 'react';
import { CheckCircle, XCircle, AlertTriangle, Zap, Clock } from 'lucide-react';

function NotificationToast({ notifications, onDismiss }) {
  const getNotificationConfig = (type) => {
    switch (type) {
      case 'job_execution_started':
        return {
          icon: Zap,
          color: 'text-blue-400',
          bg: 'bg-blue-900/20 border-blue-700/50',
          title: 'Execution Started'
        };
      case 'job_execution_completed':
        return {
          icon: CheckCircle,
          color: 'text-green-400',
          bg: 'bg-green-900/20 border-green-700/50',
          title: 'Execution Completed'
        };
      case 'job_execution_failed':
        return {
          icon: XCircle,
          color: 'text-red-400',
          bg: 'bg-red-900/20 border-red-700/50',
          title: 'Execution Failed'
        };
      case 'job_validation_failed':
        return {
          icon: AlertTriangle,
          color: 'text-yellow-400',
          bg: 'bg-yellow-900/20 border-yellow-700/50',
          title: 'Validation Failed'
        };
      default:
        return {
          icon: Clock,
          color: 'text-slate-400',
          bg: 'bg-slate-900/20 border-slate-700/50',
          title: 'Notification'
        };
    }
  };

  if (notifications.length === 0) return null;

  return (
    <div className="fixed top-4 right-4 z-50 space-y-2">
      {notifications.map((notification) => {
        const { icon: Icon, color, bg, title } = getNotificationConfig(notification.type);
        
        return (
          <div
            key={notification.id}
            className={`glass-effect rounded-lg p-4 border ${bg} max-w-sm animate-slide-in`}
          >
            <div className="flex items-start gap-3">
              <Icon className={`w-5 h-5 ${color} mt-0.5`} />
              <div className="flex-1 min-w-0">
                <h4 className="text-sm font-semibold text-white">{title}</h4>
                <p className="text-xs text-slate-300 mt-1">
                  Job: {notification.data?.job_id || 'Unknown'}
                </p>
                {notification.data?.error && (
                  <p className="text-xs text-red-400 mt-1">
                    {notification.data.error}
                  </p>
                )}
                <p className="text-xs text-slate-400 mt-1">
                  {new Date(notification.timestamp).toLocaleTimeString()}
                </p>
              </div>
              <button
                onClick={() => onDismiss(notification.id)}
                className="text-slate-400 hover:text-white transition-colors"
              >
                <XCircle className="w-4 h-4" />
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export default NotificationToast;
