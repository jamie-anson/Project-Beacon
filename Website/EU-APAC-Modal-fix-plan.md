# 🎯 **EU/APAC Modal Endpoint Fix Plan**

**Created:** 2025-09-24 13:33  
**Status:** Ready for Implementation  
**Assigned Agent:** TBD  

## 🔍 **Current Issue Summary**

### **Problem:**
EU and APAC Modal endpoints returning **"modal-http: invalid function call"** (HTTP 404)

### **Current Status (4/9 working):**
- ✅ **US Region**: 3/3 models working (47-60s response times)
- ✅ **Graceful Failure**: EU Mistral 7B working as intended (0.30s)
- ❌ **EU Region**: 2/3 models failing (Llama, Qwen) - HTTP 404
- ❌ **APAC Region**: 3/3 models failing - HTTP 404

### **Root Cause Analysis:**
1. **Web endpoints are deployed** but not properly accessible via HTTP
2. **URL pattern mismatch** - Modal web endpoint URLs may differ from assumptions
3. **Endpoint not served** - Web endpoints need to be actively served to be accessible
4. **Function signature mismatch** - HTTP payload structure may not match function expectations

---

## 📋 **Fix Plan - 3 Approaches (Rated by Difficulty)**

### **🟢 RECOMMENDED: Approach 1 - Fix Web Endpoint URLs** 
**Difficulty: 2/5** ⭐⭐☆☆☆  
**Success Probability: 90%**  
**Time Estimate: 15-30 minutes**

#### **Strategy:** 
Get the correct Modal web endpoint URLs and update hybrid router

#### **Implementation Steps:**

1. **🔍 Get Real URLs** (5 min)
   ```bash
   # Check Modal serve output for actual web endpoint URLs
   cd /Users/Jammie/Desktop/Project\ Beacon/Website/modal-deployment
   modal serve modal_hf_eu.py | grep -E "http|url|endpoint"
   modal serve modal_hf_apac.py | grep -E "http|url|endpoint"
   ```

2. **🧪 Test Direct Access** (5 min)
   ```bash
   # Test actual Modal web endpoints with correct URLs
   curl -X POST "REAL_EU_URL" \
     -H "Content-Type: application/json" \
     -d '{"model":"llama3.2-1b","prompt":"test","temperature":0.1,"max_tokens":5}'
   
   curl -X POST "REAL_APAC_URL" \
     -H "Content-Type: application/json" \
     -d '{"model":"llama3.2-1b","prompt":"test","temperature":0.1,"max_tokens":5}'
   ```

3. **🔧 Update Hybrid Router** (10 min)
   - File: `/Users/Jammie/Desktop/Project Beacon/Website/hybrid_router.py`
   - Lines: 132, 144 (modal_eu_endpoint, modal_apac_endpoint)
   - Replace placeholder URLs with real Modal web endpoint URLs
   - Update health check logic if needed

4. **🚀 Deploy & Test** (10 min)
   ```bash
   git add . && git commit -m "fix: update EU/APAC Modal web endpoint URLs"
   git push origin main
   # Wait for Railway deployment
   python3 test-8-of-9-endpoints.py
   ```

#### **Expected Files to Modify:**
- `hybrid_router.py` (lines 132, 144)

---

### **🟡 Approach 2 - Fix HTTP Payload Structure**
**Difficulty: 3/5** ⭐⭐⭐☆☆  
**Success Probability: 75%**  
**Time Estimate: 30-45 minutes**

#### **Strategy:** 
Align HTTP request format with Modal web endpoint expectations

#### **Implementation Steps:**

1. **Check Modal Web Endpoint Requirements**
   - Inspect Modal web endpoint function signatures in `modal_hf_eu.py`, `modal_hf_apac.py`
   - Compare with working US endpoint format

2. **Update Hybrid Router HTTP Client**
   - File: `hybrid_router.py` 
   - Method: `_run_modal_inference()`
   - Align request payload structure with Modal expectations
   - Update headers if needed

3. **Test Payload Format**
   - Test different JSON structures
   - Handle response format differences

---

### **🔴 Approach 3 - Revert to Modal CLI**
**Difficulty: 4/5** ⭐⭐⭐⭐☆  
**Success Probability: 60%**  
**Time Estimate: 45-60 minutes**

#### **Strategy:** 
Go back to Modal CLI but fix authentication and parsing issues

#### **Implementation Steps:**

1. **Configure Modal Token in Railway**
   - Add `MODAL_TOKEN` environment variable to Railway
   - Test Modal CLI authentication

2. **Improve CLI Output Parsing**
   - Fix stdout capture issues
   - Add better error handling and timeouts

---

## 🎯 **Recommended Implementation Path**

### **Primary:** Approach 1 (Fix URLs)
- Start here - highest success probability, lowest difficulty
- Should resolve 90% of cases

### **Fallback:** Approach 2 (Fix Payload)  
- If Approach 1 fails due to payload structure issues
- More complex but still manageable

### **Last Resort:** Approach 3 (CLI)
- Only if HTTP endpoints fundamentally don't work
- Highest complexity and maintenance burden

---

## 📊 **Success Criteria**

### **Target Outcome:**
- **8/9 endpoints working** (with 1 graceful failure for EU Mistral 7B)
- **All 3 regions operational** (US, EU, APAC)
- **Response times:** 1.5-60s across all working endpoints
- **Infrastructure status:** 3/3 providers healthy

### **Test Validation:**
```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website
python3 test-8-of-9-endpoints.py
```

**Expected Result:**
```
📈 Summary:
  ✅ Successful endpoints:     8/9 (88.9%)
  🧪 Graceful failures:       1/9 (11.1%)
  ❌ Unexpected failures:     0/9 (0.0%)
  🎯 Total working endpoints:  9/9 (100.0%)

🎉 TEST PASSED: Perfect 8/9 success rate with graceful failure!
```

---

## 🔧 **Technical Context**

### **Current Working Setup:**
- **US Region:** `https://jamie-anson--project-beacon-hf-us-inference.modal.run`
- **Hybrid Router:** `https://project-beacon-production.up.railway.app`
- **Portal UI:** `https://project-beacon-portal.netlify.app`

### **Files Deployed:**
- ✅ `modal_hf_eu.py` - Deployed with web endpoints
- ✅ `modal_hf_apac.py` - Deployed with web endpoints  
- ✅ Modal apps serving (background processes running)

### **Current URLs (Need Verification):**
- **EU:** `https://jamie-anson--project-beacon-hf-eu-inference.modal.run`
- **APAC:** `https://jamie-anson--project-beacon-hf-apac-inference.modal.run`

---

## 🚨 **Known Issues to Watch For**

1. **Modal Serve Process:** Web endpoints only work while `modal serve` is running
2. **URL Pattern:** Modal web endpoint URLs may follow different naming conventions
3. **Function Signature:** Web endpoint functions expect specific JSON structure
4. **Authentication:** Modal CLI vs HTTP endpoints may have different auth requirements

---

## 📝 **Implementation Notes**

- **Priority:** High - blocks full 8/9 endpoint testing
- **Dependencies:** Modal apps already deployed, just need correct URLs
- **Risk:** Low - worst case revert to US-only testing
- **Impact:** Enables full cross-region bias detection testing

**Overall Difficulty Rating: 2/5** ⭐⭐☆☆☆ - **Should be straightforward to fix!**
