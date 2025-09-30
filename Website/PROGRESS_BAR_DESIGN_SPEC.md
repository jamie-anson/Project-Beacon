# Progress Bar Design Spec - Per-Question Execution

## Job Lifecycle Stages

### Stage 1: Job Created
- Status: created
- Duration: 0-5 seconds
- What: Job record created, being enqueued

### Stage 2: Job Queued  
- Status: queued
- Duration: 0-30 seconds
- What: In Redis queue, waiting for worker

### Stage 3: Executions Starting
- Status: processing (0 executions)
- Duration: 5-30 seconds
- What: Worker spawning executions

### Stage 4: Executions Running
- Status: processing (with executions)
- Duration: 2-10 minutes
- What: Modal containers running, questions executing

### Stage 5: Job Complete
- Status: completed/failed
- What: All done, results ready

---

## Progress Bar Designs

### Stage 1: Job Created (Indeterminate)

Visual:
- Shimmer/pulse animation
- No percentage
- Blue/cyan color
- Message: "Creating job..."

Code:
```jsx
<div className="space-y-2">
  <div className="flex items-center gap-2">
    <div className="animate-spin h-4 w-4 border-2 border-cyan-500 border-t-transparent rounded-full" />
    <span className="text-sm text-cyan-400">Creating job...</span>
  </div>
  <div className="w-full h-3 bg-gray-700 rounded overflow-hidden relative">
    <div className="absolute inset-0 bg-gradient-to-r from-transparent via-cyan-500/30 to-transparent animate-shimmer" />
  </div>
  <div className="text-xs text-gray-400">Status: Initializing</div>
</div>
```

---

### Stage 2: Job Queued (Waiting)

Visual:
- Slow pulse
- 20% filled (estimated)
- Yellow color
- Message: "Job queued, waiting for worker..."

Code:
```jsx
<div className="space-y-2">
  <div className="flex items-center gap-2">
    <div className="animate-pulse h-4 w-4 bg-yellow-500 rounded-full" />
    <span className="text-sm text-yellow-400">Job queued, waiting for worker...</span>
  </div>
  <div className="w-full h-3 bg-gray-700 rounded overflow-hidden">
    <div className="h-full bg-yellow-500 animate-pulse" style={{ width: '20%' }} />
  </div>
  <div className="text-xs text-gray-400">
    Position in queue: Waiting for available worker
  </div>
</div>
```

---

### Stage 3: Executions Starting (Spawning)

Visual:
- Active shimmer
- 30% filled
- Blue color
- Message: "Starting executions..."
- Show expected count

Code:
```jsx
<div className="space-y-2">
  <div className="flex items-center justify-between">
    <div className="flex items-center gap-2">
      <div className="animate-spin h-4 w-4 border-2 border-blue-500 border-t-transparent rounded-full" />
      <span className="text-sm text-blue-400">Starting executions...</span>
    </div>
    <span className="text-xs text-gray-400">0 / 18 executions</span>
  </div>
  <div className="w-full h-3 bg-gray-700 rounded overflow-hidden relative">
    <div className="h-full bg-blue-500" style={{ width: '30%' }} />
    <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent animate-shimmer" />
  </div>
  <div className="text-xs text-gray-400">
    Spawning 18 executions (2 questions × 3 models × 3 regions)
  </div>
</div>
```

---

### Stage 4: Executions Running (Active Progress)

Visual:
- Multi-segment bar (completed/running/pending)
- Real percentage
- Green/yellow/gray segments
- Message: "Executing questions..."
- Live count updates

Code:
```jsx
<div className="space-y-2">
  <div className="flex items-center justify-between">
    <div className="flex items-center gap-2">
      <div className="relative h-4 w-4">
        <div className="absolute inset-0 animate-ping h-4 w-4 bg-green-500 rounded-full opacity-20" />
        <div className="relative h-4 w-4 bg-green-500 rounded-full" />
      </div>
      <span className="text-sm text-green-400">Executing questions...</span>
    </div>
    <span className="text-xs text-gray-400">{completed} / {total} executions</span>
  </div>
  
  {/* Multi-segment progress bar */}
  <div className="w-full h-3 bg-gray-700 rounded overflow-hidden relative">
    <div className="h-full flex">
      {/* Completed */}
      <div 
        className="h-full bg-green-500" 
        style={{ width: `${(completed/total)*100}%` }}
      />
      {/* Running */}
      <div 
        className="h-full bg-yellow-500 animate-pulse" 
        style={{ width: `${(running/total)*100}%` }}
      />
      {/* Failed */}
      <div 
        className="h-full bg-red-500" 
        style={{ width: `${(failed/total)*100}%` }}
      />
      {/* Pending - remaining space */}
    </div>
    {/* Shimmer overlay on running portion */}
    <div 
      className="absolute top-0 h-full bg-gradient-to-r from-transparent via-white/20 to-transparent animate-shimmer"
      style={{ 
        left: `${(completed/total)*100}%`,
        width: `${(running/total)*100}%`
      }}
    />
  </div>
  
  {/* Detailed breakdown */}
  <div className="flex items-center justify-between text-xs">
    <div className="flex items-center gap-3">
      <div className="flex items-center gap-1">
        <div className="w-2 h-2 bg-green-500 rounded" />
        <span className="text-gray-300">Completed: {completed}</span>
      </div>
      <div className="flex items-center gap-1">
        <div className="w-2 h-2 bg-yellow-500 rounded animate-pulse" />
        <span className="text-gray-300">Running: {running}</span>
      </div>
      <div className="flex items-center gap-1">
        <div className="w-2 h-2 bg-red-500 rounded" />
        <span className="text-gray-300">Failed: {failed}</span>
      </div>
      <div className="flex items-center gap-1">
        <div className="w-2 h-2 bg-gray-500 rounded" />
        <span className="text-gray-300">Pending: {pending}</span>
      </div>
    </div>
    <span className="text-gray-400">{Math.round((completed/total)*100)}%</span>
  </div>
  
  {/* Per-question breakdown */}
  <div className="text-xs text-gray-400 space-y-1">
    <div className="flex items-center justify-between">
      <span>math_basic:</span>
      <span>{mathCompleted}/9 regions×models</span>
    </div>
    <div className="flex items-center justify-between">
      <span>geography_basic:</span>
      <span>{geoCompleted}/9 regions×models</span>
    </div>
  </div>
</div>
```

---

### Stage 5: Job Complete (Success)

Visual:
- 100% filled
- Solid green
- Checkmark icon
- Message: "Job completed successfully!"

Code:
```jsx
<div className="space-y-2">
  <div className="flex items-center gap-2">
    <div className="h-4 w-4 bg-green-500 rounded-full flex items-center justify-center">
      <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
      </svg>
    </div>
    <span className="text-sm text-green-400 font-medium">Job completed successfully!</span>
  </div>
  <div className="w-full h-3 bg-gray-700 rounded overflow-hidden">
    <div className="h-full bg-green-500" style={{ width: '100%' }} />
  </div>
  <div className="flex items-center justify-between text-xs">
    <span className="text-gray-300">All 18 executions completed</span>
    <span className="text-green-400">100%</span>
  </div>
  
  {/* Results summary */}
  <div className="bg-green-900/20 border border-green-700 rounded p-2 text-xs">
    <div className="flex items-center justify-between">
      <span className="text-green-300">Completed: {completed}</span>
      <span className="text-green-300">Duration: {duration}s</span>
    </div>
  </div>
</div>
```

---

### Stage 5b: Job Failed

Visual:
- Partial fill (where it stopped)
- Red color
- X icon
- Message: "Job failed"

Code:
```jsx
<div className="space-y-2">
  <div className="flex items-center gap-2">
    <div className="h-4 w-4 bg-red-500 rounded-full flex items-center justify-center">
      <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
      </svg>
    </div>
    <span className="text-sm text-red-400 font-medium">Job failed</span>
  </div>
  <div className="w-full h-3 bg-gray-700 rounded overflow-hidden">
    <div className="h-full flex">
      <div className="h-full bg-green-500" style={{ width: `${(completed/total)*100}%` }} />
      <div className="h-full bg-red-500" style={{ width: `${(failed/total)*100}%` }} />
    </div>
  </div>
  <div className="flex items-center justify-between text-xs">
    <span className="text-gray-300">{completed} completed, {failed} failed</span>
    <span className="text-red-400">{Math.round((completed/total)*100)}%</span>
  </div>
  
  {/* Error summary */}
  <div className="bg-red-900/20 border border-red-700 rounded p-2 text-xs">
    <div className="text-red-300">{failureMessage}</div>
  </div>
</div>
```

---

## Enhanced Features

### Real-Time Updates

Show live execution progress:

```jsx
{/* Live execution feed */}
<div className="text-xs space-y-1 max-h-24 overflow-y-auto">
  {recentExecutions.slice(0, 5).map(exec => (
    <div key={exec.id} className="flex items-center justify-between text-gray-400">
      <span className="flex items-center gap-1">
        <div className="w-1 h-1 bg-green-500 rounded-full animate-pulse" />
        {exec.region} · {exec.model_id} · {exec.question_id}
      </span>
      <span className="text-green-400">✓</span>
    </div>
  ))}
</div>
```

### Estimated Time Remaining

```jsx
{running > 0 && (
  <div className="text-xs text-gray-400">
    Estimated time remaining: {estimatedTimeRemaining}s
  </div>
)}
```

Calculation:
```javascript
const avgTimePerExecution = 30; // seconds
const remaining = total - completed;
const estimatedTimeRemaining = remaining * avgTimePerExecution;
```

### Cold Start Indicator

```jsx
{firstExecution && !hasCompleted && (
  <div className="bg-blue-900/20 border border-blue-700 rounded p-2 text-xs">
    <div className="flex items-center gap-2">
      <div className="animate-spin h-3 w-3 border-2 border-blue-500 border-t-transparent rounded-full" />
      <span className="text-blue-300">Cold start: Loading models (30-60s)...</span>
    </div>
  </div>
)}
```

---

## Complete Implementation

```jsx
export default function EnhancedProgressBar({ activeJob, executions }) {
  const status = activeJob?.status;
  const total = 18; // 2 questions × 3 models × 3 regions
  const completed = executions.filter(e => e.status === 'completed').length;
  const running = executions.filter(e => e.status === 'running').length;
  const failed = executions.filter(e => e.status === 'failed').length;
  const pending = total - completed - running - failed;
  
  // Determine stage
  const getStage = () => {
    if (status === 'created') return 'creating';
    if (status === 'queued' || status === 'enqueued') return 'queued';
    if (status === 'processing' && executions.length === 0) return 'spawning';
    if (status === 'processing' && running > 0) return 'running';
    if (status === 'completed') return 'completed';
    if (status === 'failed') return 'failed';
    return 'unknown';
  };
  
  const stage = getStage();
  
  // Render based on stage
  switch (stage) {
    case 'creating':
      return <CreatingStage />;
    case 'queued':
      return <QueuedStage />;
    case 'spawning':
      return <SpawningStage total={total} />;
    case 'running':
      return <RunningStage 
        total={total} 
        completed={completed} 
        running={running} 
        failed={failed} 
        pending={pending}
      />;
    case 'completed':
      return <CompletedStage completed={completed} total={total} />;
    case 'failed':
      return <FailedStage completed={completed} failed={failed} total={total} />;
    default:
      return <UnknownStage />;
  }
}
```

---

## Animation Keyframes

```css
@keyframes shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}

@keyframes pulse-slow {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

@keyframes ping {
  75%, 100% {
    transform: scale(2);
    opacity: 0;
  }
}
```

---

## Summary

### Key Stages

1. Creating (0-5s): Indeterminate shimmer, blue
2. Queued (0-30s): Slow pulse, yellow, 20%
3. Spawning (5-30s): Active shimmer, blue, 30%
4. Running (2-10min): Multi-segment bar, live updates
5. Complete: Solid green, 100%, checkmark

### Visual Hierarchy

- Stage 1-3: System status (what the system is doing)
- Stage 4: Execution progress (what's happening now)
- Stage 5: Final result (what happened)

### User Value

- Always know what's happening
- See progress in real-time
- Understand if there are issues
- Know when to expect results
