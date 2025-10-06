/**
 * useCountdownTimer Hook
 * Manages countdown timer with tick updates for job progress
 */

import { useState, useEffect } from 'react';
import { calculateTimeRemaining } from '../lib/utils/progressUtils';

/**
 * Custom hook for countdown timer
 * @param {boolean} isActive - Whether the timer should be active
 * @param {boolean} isCompleted - Whether job is completed
 * @param {boolean} isFailed - Whether job failed
 * @param {Object} jobStartTime - Object with jobId and startTime
 * @returns {Object} Timer state
 */
export function useCountdownTimer(isActive, isCompleted, isFailed, jobStartTime) {
  const [tick, setTick] = useState(0);
  
  // Update countdown timer every second when job is active
  useEffect(() => {
    if (isActive && !isCompleted && !isFailed) {
      const interval = setInterval(() => {
        setTick(t => t + 1);
      }, 1000);
      return () => clearInterval(interval);
    } else {
      // Reset tick when job becomes inactive to prevent stale state
      setTick(0);
    }
  }, [isActive, isCompleted, isFailed]);
  
  // Calculate time remaining (tick forces re-calculation)
  const timeRemaining = calculateTimeRemaining(jobStartTime, tick, isCompleted, isFailed);
  
  return {
    tick,
    timeRemaining
  };
}
