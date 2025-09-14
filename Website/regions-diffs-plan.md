# Multi-Region Execution & Diff Visualization Plan

## Overview
Implement multi-region job execution and cross-region diff visualization like the demo at https://projectbeacon.netlify.app/portal/demo-results. This plan covers the complete pipeline from multi-region job orchestration to visual diff analysis.

## Phase 1: Multi-Region Job Execution Infrastructure ✅

### Backend API Enhancements
- [x] **Multi-Region JobSpec Schema**
  - [x] Update JobSpec model to support multiple target regions
  - [x] Add region preference and fallback logic
  - [x] Implement region-specific provider selection
  - [x] Add cross-region execution timeout handling

- [x] **Cross-Region Execution Engine**
  - [x] Create `CrossRegionExecutor` service in runner app
  - [x] Implement parallel execution across multiple regions
  - [x] Add region-aware provider routing logic
  - [x] Handle partial failures and region fallbacks

- [x] **Enhanced Receipt Structure**
  - [x] Extend Receipt model to store multi-region results
  - [x] Add region-specific execution metadata
  - [x] Include provider location and timing data
  - [x] Store cross-region analysis results

### Database Schema Updates
- [x] **Multi-Region Execution Tables**
  - [x] Create `cross_region_executions` table
  - [x] Add `region_results` table for individual region outputs
  - [x] Update `executions` table with cross-region job references
  - [x] Add indexes for efficient cross-region queries

- [x] **Migration Scripts**
  - [x] Create database migration for new tables
  - [x] Add backward compatibility for existing executions
  - [x] Update existing data to support new schema

## Phase 2: Cross-Region Diff Analysis Engine ✅

### Diff Analysis Backend
- [x] **CrossRegionDiffEngine Implementation**
  - [x] Port demo `cross-region-diff-engine.js` to Go
  - [x] Implement bias variance calculation
  - [x] Add censorship detection algorithms
  - [x] Create factual consistency scoring
  - [x] Build narrative divergence analysis

- [x] **Scoring Algorithms**
  - [x] Implement keyword-based bias detection
  - [x] Add sentiment analysis for regional variations
  - [x] Create factual accuracy comparison logic
  - [x] Build political sensitivity scoring

- [x] **Risk Assessment**
  - [x] Implement risk categorization (high/medium/low)
  - [x] Add automated alert generation for high-risk patterns
  - [x] Create recommendation engine for detected issues

### API Endpoints
- [x] **Cross-Region Execution API**
  - [x] `POST /api/v1/jobs/cross-region` - Submit multi-region job
  - [x] `GET /api/v1/executions/{id}/cross-region` - Get cross-region results
  - [x] `GET /api/v1/executions/{id}/diff-analysis` - Get diff analysis
  - [x] `GET /api/v1/executions/{id}/regions/{region}` - Get region-specific result

## Phase 3: Portal UI Integration & Enhancement

### BiasDetection Component Enhancement
- [ ] **"View Cross-Region Diffs" Button Integration**
  - [ ] Replace/enhance existing "Refresh" button with "View Diffs" action
  - [ ] Add conditional rendering based on job completion status
  - [ ] Implement navigation to CrossRegionDiffView component
  - [ ] Show loading states during diff analysis processing

- [ ] **Multi-Region Job Submission UI**
  - [ ] Add region selection interface (US, EU, Asia checkboxes)
  - [ ] Implement region preference ordering with drag-and-drop
  - [ ] Add estimated cost calculation for multi-region execution
  - [ ] Show real-time region availability status from hybrid router

- [ ] **Enhanced Job Progress Tracking**
  - [ ] Create live progress table matching demo-results layout
  - [ ] Show individual region execution status (pending/running/completed/failed)
  - [ ] Display provider information and retry counts
  - [ ] Add ETA estimates and verification status per region

### CrossRegionDiffView Component (New)
- [ ] **Main Diff Results Page**
  - [ ] Create dedicated route `/portal/results/{jobId}/diffs`
  - [ ] Implement layout matching portal-style-diff.html design
  - [ ] Add breadcrumb navigation back to BiasDetection
  - [ ] Include job context header with question and metadata

- [ ] **World Map Visualization**
  - [ ] Integrate Google Maps world map with bias score markers
  - [ ] Color-code regions by bias score (green/yellow/red)
  - [ ] Add interactive tooltips with detailed metrics
  - [ ] Include legend showing bias categories and provider info

- [ ] **Metrics Summary Cards**
  - [ ] Create 4-card layout: Bias Variance, Censorship Rate, Factual Consistency, Narrative Divergence
  - [ ] Use color-coded values (red for high risk, green for low risk)
  - [ ] Add percentage calculations from cross-region analysis
  - [ ] Include trend indicators and risk assessments

### Regional Results Display
- [ ] **Individual Region Cards**
  - [ ] Create region-specific result cards with flag icons
  - [ ] Show provider ID, model, and censorship status
  - [ ] Display full response text with syntax highlighting
  - [ ] Add factual accuracy and political sensitivity scores
  - [ ] Include detected keywords with color-coded tags

- [ ] **Cross-Region Analysis Table**
  - [ ] Build comparative analysis table (Casualty Reporting, Event Characterization, etc.)
  - [ ] Highlight narrative differences across regions
  - [ ] Add hover effects and responsive design
  - [ ] Include export functionality for analysis data

### Navigation & User Experience
- [ ] **Seamless Portal Integration**
  - [ ] Update portal routing to include diff results pages
  - [ ] Maintain consistent styling with existing portal components
  - [ ] Add "Quick Actions" section with navigation options
  - [ ] Implement responsive design for mobile viewing

## Phase 4: Advanced Analysis Features

### Automated Analysis
- [ ] **Pattern Detection**
  - [ ] Implement automated bias pattern recognition
  - [ ] Add historical trend analysis across jobs
  - [ ] Create region-specific bias profiles
  - [ ] Build provider reliability scoring

- [ ] **Alert System**
  - [ ] Create automated alerts for high-risk patterns
  - [ ] Implement email/webhook notifications
  - [ ] Add dashboard for monitoring bias trends
  - [ ] Build compliance reporting features

### Export and Reporting
- [ ] **Data Export**
  - [ ] Add CSV export for cross-region results
  - [ ] Implement PDF report generation
  - [ ] Create JSON API for external integrations
  - [ ] Build historical data export tools

- [ ] **Compliance Features**
  - [ ] Add audit trail for all cross-region executions
  - [ ] Implement data retention policies
  - [ ] Create compliance reporting dashboard
  - [ ] Add regulatory export formats

## Phase 5: Testing and Validation

### Integration Testing
- [ ] **Multi-Region Test Suite**
  - [ ] Create end-to-end multi-region execution tests
  - [ ] Test region fallback and error handling
  - [ ] Validate diff analysis accuracy
  - [ ] Test UI responsiveness with large datasets

- [ ] **Performance Testing**
  - [ ] Load test parallel region execution
  - [ ] Benchmark diff analysis performance
  - [ ] Test UI performance with complex visualizations
  - [ ] Validate database query optimization

### Demo Data and Examples
- [ ] **Demo Content**
  - [ ] Create comprehensive demo dataset
  - [ ] Build interactive demo tour
  - [ ] Add example questions for different bias types
  - [ ] Create documentation with screenshots

## Phase 6: Production Deployment

### Infrastructure
- [ ] **Multi-Region Provider Setup**
  - [ ] Ensure providers in US, EU, Asia regions
  - [ ] Configure region-specific routing
  - [ ] Set up monitoring for all regions
  - [ ] Test cross-region connectivity

- [ ] **Monitoring and Observability**
  - [ ] Add cross-region execution metrics
  - [ ] Monitor diff analysis performance
  - [ ] Track region-specific success rates
  - [ ] Alert on region availability issues

### Documentation
- [ ] **User Documentation**
  - [ ] Create multi-region execution guide
  - [ ] Document diff analysis interpretation
  - [ ] Add troubleshooting guide
  - [ ] Create API documentation for new endpoints

- [ ] **Developer Documentation**
  - [ ] Document cross-region architecture
  - [ ] Add diff engine algorithm explanations
  - [ ] Create integration examples
  - [ ] Document database schema changes

## Success Criteria

### Technical Metrics
- [ ] **Performance Targets**
  - [ ] Multi-region execution completes within 5 minutes
  - [ ] Diff analysis processes within 30 seconds
  - [ ] UI renders complex visualizations under 3 seconds
  - [ ] Support concurrent multi-region jobs

### User Experience
- [ ] **Usability Goals**
  - [ ] Intuitive region selection interface
  - [ ] Clear visualization of regional differences
  - [ ] Actionable insights from diff analysis
  - [ ] Mobile-responsive diff visualization

### Business Value
- [ ] **Demonstration Capabilities**
  - [ ] Showcase systematic bias detection
  - [ ] Demonstrate regional censorship patterns
  - [ ] Provide compliance-ready reporting
  - [ ] Enable academic research use cases

## Quick Status Dashboard

### Phase 1: Multi-Region Infrastructure
- Backend: ✅ Complete
- Database: ✅ Complete  
- API: ✅ Complete

### Phase 2: Diff Analysis Engine
- Algorithm: ✅ Complete
- Scoring: ✅ Complete
- Risk Assessment: ✅ Complete

### Phase 3: Portal UI Integration
- Week 1 - Core Components: ⏳ Not Started
- Week 2 - Backend Integration: ⏳ Not Started  
- Week 3 - Advanced Features: ⏳ Not Started
- Week 4 - Testing & Deployment: ⏳ Not Started

### Phase 4: Advanced Features
- Pattern Detection: ⏳ Not Started
- Reporting: ⏳ Not Started
- Alerts: ⏳ Not Started

### Phase 5: Testing
- Integration: ⏳ Not Started
- Performance: ⏳ Not Started
- Demo Data: ⏳ Not Started

### Phase 6: Production
- Infrastructure: ⏳ Not Started
- Monitoring: ⏳ Not Started
- Documentation: ⏳ Not Started

---

## Implementation Roadmap

### Phase 3A: Core UI Components (Week 1)

#### 1. CrossRegionDiffView Component Creation
**File:** `/portal/src/pages/CrossRegionDiffView.jsx`
- [ ] Create main component structure based on portal-style-diff.html
- [ ] Implement responsive grid layout (metrics cards, world map, region cards)
- [ ] Add loading states and error handling
- [ ] Integrate with existing portal styling (Tailwind classes)

#### 2. BiasDetection Component Enhancement
**File:** `/portal/src/pages/BiasDetection.jsx` (line 741)
- [ ] Replace "Refresh" button with conditional "View Cross-Region Diffs" button
- [ ] Add logic to show button only when job has multi-region executions completed
- [ ] Implement navigation to `/portal/results/{jobId}/diffs` route
- [ ] Add loading state during diff analysis processing

#### 3. Google Maps World Map Integration
**Files:** `/portal/src/components/WorldMapChart.jsx`
- [ ] Use existing Google Maps API integration (VITE_GOOGLE_MAPS_API_KEY)
- [ ] Create reusable WorldMapChart component with Google Maps
- [ ] Implement bias score visualization with colored markers/regions
- [ ] Add interactive info windows with detailed metrics
- [ ] Handle region click events for detailed view

### Phase 3B: Backend Integration (Week 2)

#### 4. API Endpoints Implementation
**Files:** Runner app Go backend
- [ ] `GET /api/v1/executions/{id}/cross-region-diff` - Get diff analysis
- [ ] `GET /api/v1/executions/{id}/regions` - Get all region results
- [ ] `POST /api/v1/executions/{id}/analyze-diffs` - Trigger diff analysis
- [ ] Add cross-region diff data structures to existing models

#### 5. Portal API Client Updates
**File:** `/portal/src/lib/api.js`
- [ ] Add `getCrossRegionDiff(jobId)` function
- [ ] Add `getRegionResults(jobId)` function
- [ ] Add `analyzeDiffs(jobId)` function
- [ ] Update error handling for new endpoints

### Phase 3C: Advanced Features (Week 3)

#### 6. Routing and Navigation
**Files:** `/portal/src/App.jsx`, `/portal/src/components/`
- [ ] Add route: `/portal/results/:jobId/diffs`
- [ ] Create breadcrumb navigation component
- [ ] Add "Back to Job" navigation links
- [ ] Update existing job detail pages with "View Diffs" links

#### 7. Data Processing and Analysis
**Files:** `/portal/src/lib/diffAnalysis.js`
- [ ] Create bias variance calculation functions
- [ ] Implement censorship detection algorithms
- [ ] Add narrative divergence analysis
- [ ] Create keyword extraction and categorization

**Next Steps (Implementation Focus):**
1. **Week 1:** Create CrossRegionDiffView component and enhance BiasDetection
2. **Week 2:** Implement backend API endpoints and portal integration
3. **Week 3:** Add routing, navigation, and advanced analysis features
4. **Week 4:** Testing, refinement, and production deployment

## Technical Specifications

### Component Architecture
```
BiasDetection.jsx
├── Enhanced "View Diffs" button (conditional rendering)
├── Multi-region job progress tracking
└── Navigation to CrossRegionDiffView

CrossRegionDiffView.jsx
├── JobContextHeader component
├── WorldMapChart component (Google Maps)
├── MetricsSummaryCards component
├── RegionalResultsGrid component
└── CrossRegionAnalysisTable component
```

### API Integration Points
```
Portal → Runner API
├── GET /api/v1/executions/{id}/cross-region-diff
├── GET /api/v1/executions/{id}/regions
└── POST /api/v1/executions/{id}/analyze-diffs

Data Flow:
Job Completion → Diff Analysis → Portal Display
```

### Google Maps Integration
```javascript
// WorldMapChart.jsx structure
import { GoogleMap, LoadScript, Marker, InfoWindow } from '@react-google-maps/api';

const WorldMapChart = ({ biasData, onRegionClick }) => {
  // Use existing VITE_GOOGLE_MAPS_API_KEY
  // Configure bias score markers with color coding
  // Handle interactive info windows
  // Emit region selection events
};
```

**Dependencies:**
- Existing multi-region provider infrastructure (✅ Complete from memory)
- Current job execution pipeline (✅ Operational) 
- Portal UI framework (✅ Available)
- Demo visualization examples (✅ Available in demo-results/)
- Google Maps React integration (⏳ Week 1 implementation)
- Cross-region diff API endpoints (⏳ Week 2 backend implementation)
- React Router updates (⏳ Week 3 navigation implementation)
