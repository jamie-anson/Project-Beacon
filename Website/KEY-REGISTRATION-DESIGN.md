# Key Registration & Approval Workflow Design

## Overview
Replace manual key management with an automated registration and approval system that supports multiple users while maintaining security.

## Current Problem
- Each browser generates unique Ed25519 keypairs
- Manual addition to trusted-keys.json required for each user
- No scalable way to manage multiple users
- No key lifecycle management (revocation, expiration, audit)

## Proposed Solution

### 1. Key Registration Flow

```
User Flow:
1. User visits portal → generates keypair (as now)
2. Portal detects untrusted key → shows registration UI
3. User fills registration form (name, email, reason)
4. System creates pending key request
5. Admin receives notification for approval
6. Admin approves/rejects via admin interface
7. User gets notification of approval status
8. Approved users can submit jobs immediately
```

### 2. API Endpoints

#### Key Registration
```
POST /api/v1/keys/register
{
  "public_key": "base64-encoded-key",
  "user_name": "John Doe",
  "email": "john@example.com", 
  "organization": "Research Lab",
  "reason": "Running bias detection experiments",
  "portal_version": "1.0.0"
}
```

#### Key Management (Admin)
```
GET /api/v1/admin/keys/pending     # List pending requests
POST /api/v1/admin/keys/approve    # Approve key
POST /api/v1/admin/keys/reject     # Reject key
POST /api/v1/admin/keys/revoke     # Revoke active key
GET /api/v1/admin/keys/active      # List active keys
```

#### Key Status Check
```
GET /api/v1/keys/status/{public_key}
Response: { "status": "pending|approved|rejected|revoked", "expires_at": "..." }
```

### 3. Database Schema

```sql
CREATE TABLE key_requests (
    id UUID PRIMARY KEY,
    public_key TEXT UNIQUE NOT NULL,
    user_name TEXT NOT NULL,
    email TEXT NOT NULL,
    organization TEXT,
    reason TEXT,
    status TEXT CHECK (status IN ('pending', 'approved', 'rejected', 'revoked')),
    requested_at TIMESTAMP DEFAULT NOW(),
    reviewed_at TIMESTAMP,
    reviewed_by TEXT,
    expires_at TIMESTAMP,
    portal_version TEXT,
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_key_requests_status ON key_requests(status);
CREATE INDEX idx_key_requests_public_key ON key_requests(public_key);
```

### 4. Portal UI Changes

#### Registration Modal
```jsx
// Show when job submission fails with "untrusted key"
<KeyRegistrationModal 
  publicKey={userPublicKey}
  onSubmit={handleRegistration}
  onCancel={handleCancel}
/>
```

#### Key Status Component
```jsx
// Show current key status in portal
<KeyStatus 
  publicKey={userPublicKey}
  status={keyStatus}
  onReregister={handleReregister}
/>
```

### 5. Admin Interface

#### Pending Requests Dashboard
- List all pending key registration requests
- Show user details, reason, registration date
- One-click approve/reject with optional notes
- Bulk operations for multiple requests

#### Active Keys Management
- List all approved keys with usage stats
- Revoke keys with reason tracking
- Set expiration dates
- View audit logs

### 6. Security Features

#### Request Validation
- Rate limiting on registration attempts
- Email verification (optional)
- Captcha for spam prevention
- IP address tracking

#### Key Lifecycle
- Automatic expiration (default 1 year)
- Revocation with immediate effect
- Audit trail for all key operations
- Usage monitoring and alerts

#### Admin Controls
- Multi-admin approval (optional)
- Approval workflows with notifications
- Role-based access control
- Audit logging for all admin actions

### 7. Implementation Phases

#### Phase 1: Core Registration (Week 1)
- [ ] Database schema and migrations
- [ ] Registration API endpoints
- [ ] Basic portal registration UI
- [ ] Admin approval interface

#### Phase 2: Enhanced Security (Week 2)
- [ ] Email notifications
- [ ] Key expiration handling
- [ ] Rate limiting and spam protection
- [ ] Audit logging

#### Phase 3: Advanced Features (Week 3)
- [ ] Bulk key management
- [ ] Usage analytics
- [ ] Advanced admin workflows
- [ ] API key management for programmatic access

### 8. Migration Strategy

#### Existing Keys
- Import current trusted-keys.json into database
- Mark existing keys as "approved" with admin user
- Maintain backward compatibility during transition

#### Rollout Plan
1. Deploy registration system alongside existing trust file
2. Test with internal users
3. Gradually migrate to database-only trust validation
4. Deprecate trusted-keys.json file

### 9. Configuration

#### Environment Variables
```bash
# Key registration settings
KEY_REGISTRATION_ENABLED=true
KEY_AUTO_APPROVAL=false
KEY_DEFAULT_EXPIRY_DAYS=365
KEY_ADMIN_EMAIL=admin@projectbeacon.org

# Email notifications
SMTP_HOST=smtp.example.com
SMTP_USER=notifications@projectbeacon.org
SMTP_PASS=secret

# Rate limiting
KEY_REGISTRATION_RATE_LIMIT=5/hour
KEY_REGISTRATION_DAILY_LIMIT=10
```

### 10. Benefits

#### For Users
- Self-service key registration
- Clear status visibility
- No manual admin coordination needed
- Immediate feedback on approval status

#### For Admins
- Centralized key management
- Audit trail and compliance
- Scalable approval process
- Usage monitoring and analytics

#### For System
- Automated trust management
- Better security controls
- Reduced operational overhead
- Improved user experience

## Next Steps

1. **Design Review**: Validate approach with stakeholders
2. **Database Design**: Finalize schema and migrations
3. **API Implementation**: Build registration endpoints
4. **Portal Integration**: Add registration UI components
5. **Admin Interface**: Create key management dashboard
6. **Testing**: Comprehensive testing of registration flow
7. **Documentation**: User and admin guides
8. **Deployment**: Phased rollout with monitoring
