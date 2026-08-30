// Package apicodes documents stable HTTP API error codes returned as {"error":{"code","message"}}.
package apicodes

// Entry describes one error code.
type Entry struct {
	Domain  string
	Summary string
}

// Catalog is the authoritative list of API error codes (M0+).
// New errorBody("CODE", ...) usages must register here; apicodes tests enforce parity.
var Catalog = map[string]Entry{
	// common / authz
	"INVALID_REQUEST":          {Domain: "common", Summary: "Request body or parameters failed validation"},
	"FORBIDDEN":                {Domain: "authz", Summary: "Authenticated but missing role or permission"},
	"UNAUTHORIZED":             {Domain: "auth", Summary: "Missing or invalid bearer token / actor"},
	"SPACE_ACCESS_DENIED":      {Domain: "authz", Summary: "Actor cannot access the requested space"},
	"PERMISSION_CHECK_FAILED":  {Domain: "authz", Summary: "Permission evaluation failed"},
	"PERMISSION_LOOKUP_FAILED": {Domain: "authz", Summary: "Failed to load actor permissions"},
	"RUN_SCOPE_CHECK_FAILED":   {Domain: "authz", Summary: "Failed to verify run belongs to active space"},

	// auth
	"INVALID_CREDENTIALS":    {Domain: "auth", Summary: "Email/password mismatch"},
	"USER_DISABLED":        {Domain: "auth", Summary: "User account is not active"},
	"USER_NOT_FOUND":         {Domain: "auth", Summary: "Authenticated user record not found"},
	"LOGIN_SCOPE_FAILED":     {Domain: "auth", Summary: "Failed to resolve login space membership"},
	"TOKEN_SIGN_FAILED":      {Domain: "auth", Summary: "Failed to sign session JWT"},
	"WEAK_PASSWORD":          {Domain: "auth", Summary: "Password does not meet minimum length"},
	"PASSWORD_HASH_FAILED":   {Domain: "auth", Summary: "Failed to hash password"},
	"PASSWORD_UPDATE_FAILED": {Domain: "auth", Summary: "Failed to persist password change"},
	"SPACE_LOOKUP_FAILED":    {Domain: "auth", Summary: "Failed to load space for session"},

	// runs
	"RUN_NOT_FOUND":           {Domain: "runs", Summary: "Run id not found"},
	"RUN_CREATE_FAILED":       {Domain: "runs", Summary: "Failed to create or start run"},
	"RUN_LIST_FAILED":         {Domain: "runs", Summary: "Failed to list runs"},
	"RUN_CONTROL_FAILED":      {Domain: "runs", Summary: "Resume/replay/cancel operation failed"},
	"RUN_NOT_RESUMABLE":       {Domain: "runs", Summary: "Run status does not allow resume"},
	"RUN_NOT_APPROVABLE":      {Domain: "runs", Summary: "Run status does not allow approve"},
	"RUN_NOT_REPLAYABLE":      {Domain: "runs", Summary: "Run status does not allow replay"},
	"ILLEGAL_STATUS_TRANSITION": {Domain: "runs", Summary: "Requested run status transition is illegal"},
	"RUN_CANCELED":            {Domain: "runs", Summary: "Run was canceled and cannot continue"},
	"RUN_META_MISSING":        {Domain: "runs", Summary: "Run metadata required for control action is missing"},
	"INVALID_REPLAY_MODE":     {Domain: "runs", Summary: "Replay mode must be exact or latest_memory"},
	"RUN_LOOKUP_FAILED":       {Domain: "runs", Summary: "Failed to load run for provenance"},
	"RUN_TIMELINE_NOT_FOUND":  {Domain: "runs", Summary: "Run timeline not available"},
	"EVENT_LIST_FAILED":       {Domain: "runs", Summary: "Failed to list run events for SSE"},
	"SSE_UNSUPPORTED":         {Domain: "runs", Summary: "Response writer does not support streaming"},
	"TOOL_CALL_LIST_FAILED":   {Domain: "runs", Summary: "Failed to list tool calls"},
	"AGENT_TASK_LIST_FAILED":  {Domain: "runs", Summary: "Failed to list agent tasks"},
	"ARTIFACTS_NOT_FOUND":     {Domain: "runs", Summary: "Run artifacts manifest not found"},
	"ARTIFACT_ACCESS_NOT_FOUND":   {Domain: "runs", Summary: "Artifact access URL could not be built"},
	"CHECKPOINT_LIST_NOT_FOUND":   {Domain: "runs", Summary: "Run checkpoints not found"},
	"CHECKPOINT_ACCESS_NOT_FOUND": {Domain: "runs", Summary: "Checkpoint access URL could not be built"},

	// rules / scenarios
	"SCENARIO_NOT_FOUND": {Domain: "rules", Summary: "Scenario name/version not loaded"},

	// memory
	"MEMORY_CREATE_FAILED":       {Domain: "memory", Summary: "Failed to create memory candidate"},
	"MEMORY_LIST_FAILED":         {Domain: "memory", Summary: "Failed to list memory candidates"},
	"MEMORY_NOT_FOUND":           {Domain: "memory", Summary: "Memory record or candidate not found"},
	"MEMORY_SCOPE_CHECK_FAILED":  {Domain: "memory", Summary: "Failed to verify memory belongs to space"},
	"MEMORY_REVIEW_FAILED":       {Domain: "memory", Summary: "Memory review decision failed"},
	"MEMORY_QUERY_FAILED":        {Domain: "memory", Summary: "Memory query failed"},
	"MEMORY_HIT_FAILED":          {Domain: "memory", Summary: "Failed to record memory hit-used"},
	"MEMORY_MIGRATION_FAILED":    {Domain: "memory", Summary: "Memory catalog migration failed"},
	"MEMORY_TTL_QUEUE_FAILED":    {Domain: "memory", Summary: "Failed to list memory TTL review queue"},
	"MEMORY_TTL_SWEEP_FAILED":    {Domain: "memory", Summary: "Memory TTL sweep failed"},

	// rag
	"RAG_INDEX_FAILED": {Domain: "rag", Summary: "Repository index build failed"},
	"RAG_QUERY_FAILED": {Domain: "rag", Summary: "Retrieval query failed"},

	// model router
	"MODEL_USAGE_RECORD_FAILED": {Domain: "model", Summary: "Failed to persist model usage row"},

	// tool / mcp
	"MCP_TOOL_LIST_FAILED":   {Domain: "tool", Summary: "Failed to list MCP tools"},
	"MCP_TOOL_CREATE_FAILED": {Domain: "tool", Summary: "Failed to register MCP tool"},

	// improve
	"BASELINE_NOT_READY":        {Domain: "improve", Summary: "Baseline run not finished for proposal"},
	"IMPROVE_CREATE_FAILED":     {Domain: "improve", Summary: "Failed to create improvement proposal"},
	"IMPROVE_LIST_FAILED":       {Domain: "improve", Summary: "Failed to list proposals"},
	"IMPROVE_GET_FAILED":        {Domain: "improve", Summary: "Failed to load proposal"},
	"PROPOSAL_NOT_FOUND":        {Domain: "improve", Summary: "Proposal id not found"},
	"INVALID_STATE":             {Domain: "improve", Summary: "Proposal not in required status"},
	"IMPROVE_EXPERIMENT_FAILED": {Domain: "improve", Summary: "Experiment replay failed"},
	"IMPROVE_CANARY_FAILED":     {Domain: "improve", Summary: "Canary rollout failed"},
	"IMPROVE_PROMOTE_FAILED":    {Domain: "improve", Summary: "Promotion failed"},
	"IMPROVE_ROLLBACK_FAILED":   {Domain: "improve", Summary: "Rollback failed"},

	// harness (v2)
	"HARNESS_LIST_FAILED":        {Domain: "harness", Summary: "Failed to list harness profiles"},
	"HARNESS_CREATE_FAILED":      {Domain: "harness", Summary: "Failed to create harness profile"},
	"HARNESS_GET_FAILED":         {Domain: "harness", Summary: "Failed to load harness profile"},
	"HARNESS_UPDATE_FAILED":      {Domain: "harness", Summary: "Failed to update harness profile"},
	"HARNESS_SUBMIT_FAILED":      {Domain: "harness", Summary: "Failed to submit harness profile for review"},
	"HARNESS_PROMOTE_FAILED":     {Domain: "harness", Summary: "Failed to promote harness profile"},
	"HARNESS_ROLLBACK_FAILED":    {Domain: "harness", Summary: "Failed to rollback harness profile"},
	"HARNESS_LOAD_ACTIVE_FAILED": {Domain: "harness", Summary: "Failed to load active harness profile"},
	"HARNESS_NOT_FOUND":          {Domain: "harness", Summary: "Harness profile not found"},
	"SCENARIO_PATCH_LIST_FAILED":   {Domain: "evolve", Summary: "Failed to list scenario patches"},
	"SCENARIO_PATCH_CREATE_FAILED": {Domain: "evolve", Summary: "Failed to create scenario patch"},
	"SCENARIO_PATCH_SUBMIT_FAILED": {Domain: "evolve", Summary: "Failed to submit scenario patch"},
	"SCENARIO_PATCH_NOT_FOUND":     {Domain: "evolve", Summary: "Scenario patch not found"},
	"INVALID_TARGET_TYPE":        {Domain: "feedback", Summary: "Unsupported feedback targetType"},
	"FEEDBACK_DUPLICATE":         {Domain: "feedback", Summary: "Duplicate feedback for run target"},
	"REVIEWS_QUEUE_FAILED":       {Domain: "evolve", Summary: "Failed to list review queue"},
	"INVALID_REVIEW_ID":          {Domain: "evolve", Summary: "Invalid review queue item id"},
	"REVIEW_DECIDE_FAILED":       {Domain: "evolve", Summary: "Failed to decide review item"},
	"GOAL_ROUTE_FAILED":          {Domain: "goal", Summary: "Failed to route goal to scenario"},
	"GOAL_PLAN_NOT_FOUND":        {Domain: "goal", Summary: "Goal plan not found"},
	"GOAL_PLAN_GET_FAILED":       {Domain: "goal", Summary: "Failed to load goal plan"},
	"GOAL_PLAN_APPROVE_FAILED":   {Domain: "goal", Summary: "Failed to approve goal plan"},
	"GOAL_PLAN_REJECT_FAILED":    {Domain: "goal", Summary: "Failed to reject goal plan"},
	"QUEST_BOARD_FAILED":         {Domain: "quest", Summary: "Failed to load quest board"},
	"REPO_PROFILE_FAILED":        {Domain: "knowledge", Summary: "Failed to build repo profile"},
	"WIKI_LIST_FAILED":           {Domain: "knowledge", Summary: "Failed to list wiki pages"},
	"WIKI_GET_FAILED":            {Domain: "knowledge", Summary: "Failed to get wiki page"},
	"SPACE_RULES_GET_FAILED":     {Domain: "spaces", Summary: "Failed to load space rules"},
	"SPACE_RULES_PUT_FAILED":     {Domain: "spaces", Summary: "Failed to save space rules"},
	"SPACE_RULES_IMPORT_FAILED":  {Domain: "spaces", Summary: "Failed to import space rules from file"},
	"SPACE_RULES_EXPORT_FAILED":  {Domain: "spaces", Summary: "Failed to export space rules to file"},
	"SPACE_RULES_PREVIEW_FAILED": {Domain: "spaces", Summary: "Failed to preview space rules routing"},
	"SKILLS_LIST_FAILED":         {Domain: "skills", Summary: "Failed to list repository skills"},
	"SKILLS_GET_FAILED":          {Domain: "skills", Summary: "Failed to get skill"},
	"SUBRUN_SPAWN_FAILED":        {Domain: "runs", Summary: "Failed to spawn sub-run"},
	"SUBRUN_TREE_FAILED":         {Domain: "runs", Summary: "Failed to load run spawn tree"},
	"DIFF_GET_FAILED":            {Domain: "quest", Summary: "Failed to load run diff"},
	"DIFF_COMMENT_LIST_FAILED":   {Domain: "quest", Summary: "Failed to list diff comments"},
	"DIFF_COMMENT_CREATE_FAILED": {Domain: "quest", Summary: "Failed to create diff comment"},

	// ci / repo
	"PLAINTEXT_TOKEN_REJECTED":      {Domain: "ci", Summary: "Repo connection must use secretId, not plaintext token"},
	"INVALID_SECRET_REFERENCE":      {Domain: "ci", Summary: "secretId not found or inactive in space"},
	"REPO_CONNECTION_CREATE_FAILED": {Domain: "ci", Summary: "Failed to create repo connection"},
	"REPO_CONNECTION_LIST_FAILED":   {Domain: "ci", Summary: "Failed to list repo connections"},
	"CI_RUN_LIST_FAILED":            {Domain: "ci", Summary: "Failed to list CI runs"},
	"CI_JOB_LIST_FAILED":            {Domain: "ci", Summary: "Failed to list CI jobs"},
	"CI_PROVIDER_UNAVAILABLE":       {Domain: "ci", Summary: "GitHub CI provider unavailable after retry/circuit open"},
	"CI_DIAGNOSE_FAILED":            {Domain: "ci", Summary: "CI failure diagnosis failed"},
	"CI_DIAGNOSIS_LIST_FAILED":      {Domain: "ci", Summary: "Failed to list CI diagnoses"},
	"CI_DIAGNOSIS_NOT_FOUND":        {Domain: "ci", Summary: "CI diagnosis not found"},
	"CI_DIAGNOSIS_DECISION_FAILED":  {Domain: "ci", Summary: "Failed to adopt/dismiss diagnosis"},
	"INVALID_FROM":                  {Domain: "metrics", Summary: "metrics overview from timestamp invalid"},
	"INVALID_TO":                    {Domain: "metrics", Summary: "metrics overview to timestamp invalid"},
	"METRICS_OVERVIEW_FAILED":       {Domain: "metrics", Summary: "KPI overview query failed"},

	// observability
	"ALERT_LIST_FAILED":          {Domain: "observability", Summary: "Failed to list alert events"},
	"ALERT_RULE_LIST_FAILED":     {Domain: "observability", Summary: "Failed to list alert rules"},
	"ALERT_RULE_UPDATE_FAILED":   {Domain: "observability", Summary: "Failed to update alert rules"},
	"ALERT_EVALUATE_FAILED":      {Domain: "observability", Summary: "Alert rule evaluation failed"},
	"TRACE_LOOKUP_FAILED":        {Domain: "observability", Summary: "Trace-linked record lookup failed"},
	"WATERFALL_NOT_FOUND":        {Domain: "observability", Summary: "Run waterfall not found"},
	"QUALITY_METRIC_LIST_FAILED": {Domain: "observability", Summary: "Failed to list quality metrics"},
	"OBS_CONFIG_LOAD_FAILED":     {Domain: "observability", Summary: "Failed to load observability config"},

	// feedback
	"FEEDBACK_CREATE_FAILED": {Domain: "feedback", Summary: "Failed to create feedback"},
	"FEEDBACK_LIST_FAILED":   {Domain: "feedback", Summary: "Failed to list feedback"},
	"FEEDBACK_NOT_FOUND":     {Domain: "feedback", Summary: "Feedback id not found"},
	"FEEDBACK_UPDATE_FAILED": {Domain: "feedback", Summary: "Failed to update feedback"},
	"FEEDBACK_RELOAD_FAILED": {Domain: "feedback", Summary: "Failed to reload feedback after update"},

	// orgs / spaces / permissions
	"ORG_LIST_FAILED":                {Domain: "platform", Summary: "Failed to list organizations"},
	"ORG_CREATE_FAILED":              {Domain: "platform", Summary: "Failed to create organization"},
	"ORG_TEMPLATE_UNKNOWN":           {Domain: "platform", Summary: "Unknown organization commercial template"},
	"ORG_TEMPLATE_PROVISION_FAILED":  {Domain: "platform", Summary: "Failed to provision organization template"},
	"ORG_NOT_FOUND":                  {Domain: "platform", Summary: "Organization not found"},
	"SPACE_LIST_FAILED":              {Domain: "platform", Summary: "Failed to list spaces"},
	"SPACE_CREATE_FAILED":            {Domain: "platform", Summary: "Failed to create space"},
	"SPACE_NOT_FOUND":                {Domain: "platform", Summary: "Space not found"},
	"ROLE_LIST_FAILED":               {Domain: "platform", Summary: "Failed to list roles"},
	"ROLE_CREATE_FAILED":             {Domain: "platform", Summary: "Failed to create role"},
	"ROLE_NOT_FOUND":                 {Domain: "platform", Summary: "Role not found"},
	"ROLE_SCOPE_MISMATCH":            {Domain: "platform", Summary: "Role org does not match space org"},
	"MEMBER_LIST_FAILED":             {Domain: "platform", Summary: "Failed to list space members"},
	"MEMBER_CREATE_FAILED":           {Domain: "platform", Summary: "Failed to add space member"},
	"RESOURCE_SCOPE_LIST_FAILED":     {Domain: "platform", Summary: "Failed to list resource scopes"},
	"RESOURCE_SCOPE_NOT_FOUND":       {Domain: "platform", Summary: "Resource scope not found"},
	"RESOURCE_SCOPE_UPDATE_FAILED":   {Domain: "platform", Summary: "Failed to update resource scope"},
	"INVALID_POLICY":                 {Domain: "platform", Summary: "Scenario policy JSON invalid"},
	"INVALID_RESOURCE_TYPE":          {Domain: "platform", Summary: "Resource scope type does not support update"},
	"PERMISSION_MATRIX_FAILED":       {Domain: "platform", Summary: "Failed to build permission matrix"},

	// audit
	"AUDIT_EXPORT_CREATE_FAILED":   {Domain: "audit", Summary: "Failed to create audit export job"},
	"AUDIT_EXPORT_FAILED":          {Domain: "audit", Summary: "Audit export processing failed"},
	"AUDIT_EXPORT_LIST_FAILED":     {Domain: "audit", Summary: "Failed to list audit exports"},
	"AUDIT_EXPORT_NOT_FOUND":       {Domain: "audit", Summary: "Audit export id not found"},
	"AUDIT_EXPORT_NOT_READY":       {Domain: "audit", Summary: "Audit export not ready for download"},
	"AUDIT_EXPORT_ACCESS_FAILED":   {Domain: "audit", Summary: "Failed to build audit export access URL"},
	"AUDIT_LOG_LIST_FAILED":        {Domain: "audit", Summary: "Failed to list audit logs"},
	"AUDIT_POLICY_GET_FAILED":      {Domain: "audit", Summary: "Failed to load audit policy"},
	"INVALID_AUDIT_POLICY":         {Domain: "audit", Summary: "Audit retention policy out of range"},
	"AUDIT_POLICY_LOCKED":          {Domain: "audit", Summary: "Audit policy is locked"},
	"AUDIT_POLICY_UPDATE_FAILED":   {Domain: "audit", Summary: "Failed to update audit policy"},
	"AUDIT_RETENTION_COUNT_FAILED": {Domain: "audit", Summary: "Failed to count rows for retention"},
	"AUDIT_RETENTION_APPLY_FAILED": {Domain: "audit", Summary: "Failed to apply audit retention"},
	"EVENTS_RETENTION_COUNT_FAILED":    {Domain: "compliance", Summary: "Failed to count run_events for retention"},
	"EVENTS_RETENTION_APPLY_FAILED":    {Domain: "compliance", Summary: "Failed to apply run_events retention"},
	"ARTIFACTS_RETENTION_COUNT_FAILED": {Domain: "compliance", Summary: "Failed to count artifacts for retention"},
	"ARTIFACTS_RETENTION_APPLY_FAILED": {Domain: "compliance", Summary: "Failed to apply artifacts retention"},

	// plugins
	"PLUGIN_LIST_FAILED":            {Domain: "plugins", Summary: "Failed to list plugins"},
	"PLUGIN_REGISTER_FAILED":        {Domain: "plugins", Summary: "Failed to register plugin"},
	"PLUGIN_SIGNATURE_INVALID":      {Domain: "plugins", Summary: "Plugin registration signature missing or invalid"},
	"PLUGIN_NOT_FOUND":              {Domain: "plugins", Summary: "Plugin id not found"},
	"PLUGIN_VERIFY_FAILED":          {Domain: "plugins", Summary: "Plugin ABI verification failed"},
	"PLUGIN_ABI_PROFILE_FAILED":     {Domain: "plugins", Summary: "Failed to load plugin ABI profile"},
	"PLUGIN_EXPORT_AUDIT_FAILED":    {Domain: "plugins", Summary: "Failed to write plugin export audit"},
	"PLUGIN_EXPORT_EVENT_FAILED":    {Domain: "plugins", Summary: "Failed to emit plugin export run event"},
	"PLUGIN_EXPORT_REPORT_FAILED":   {Domain: "plugins", Summary: "Failed to record plugin export report"},
	"PLUGIN_HEALTH_FAILED":          {Domain: "plugins", Summary: "Failed to load plugin export health"},

	// secrets
	"SECRET_LIST_FAILED":    {Domain: "secrets", Summary: "Failed to list secrets"},
	"INVALID_SECRET_NAME":   {Domain: "secrets", Summary: "Secret name format invalid"},
	"SECRET_ALREADY_EXISTS": {Domain: "secrets", Summary: "Secret name already used in space"},
	"SECRET_LOOKUP_FAILED":  {Domain: "secrets", Summary: "Secret lookup failed"},
	"SECRET_ENCRYPT_FAILED": {Domain: "secrets", Summary: "Secret encryption failed"},
	"INVALID_SECRET_SCOPE":  {Domain: "secrets", Summary: "Secret scope JSON invalid"},
	"SECRET_CREATE_FAILED":  {Domain: "secrets", Summary: "Failed to create secret"},
	"SECRET_ROTATE_FAILED":  {Domain: "secrets", Summary: "Failed to rotate secret"},
	"SECRET_DELETE_FAILED":  {Domain: "secrets", Summary: "Failed to delete secret"},
	"SECRET_NOT_FOUND":      {Domain: "secrets", Summary: "Secret id not found"},

	// releases
	"RELEASE_CREATE_FAILED":           {Domain: "releases", Summary: "Failed to create release record"},
	"RELEASE_LIST_FAILED":             {Domain: "releases", Summary: "Failed to list releases"},
	"RELEASE_CHECKLIST_FAILED":        {Domain: "releases", Summary: "Failed to load release checklist"},
	"RELEASE_CHECKLIST_UPDATE_FAILED": {Domain: "releases", Summary: "Failed to update checklist items"},
	"RELEASE_GATE_FAILED":             {Domain: "releases", Summary: "Release gate evaluation failed"},
	"ROLLBACK_DRILL_CREATE_FAILED":    {Domain: "releases", Summary: "Failed to record rollback drill"},

	// doctor / compliance
	"DOCTOR_FAILED":    {Domain: "doctor", Summary: "Doctor suite execution failed"},
	"REPORT_NOT_FOUND": {Domain: "doctor", Summary: "Doctor report id not found"},

	// approvals
	"APPROVAL_LIST_FAILED":   {Domain: "approvals", Summary: "Failed to list approval requests"},
	"APPROVAL_REJECT_FAILED": {Domain: "approvals", Summary: "Failed to reject approval"},
	"APPROVAL_NOT_FOUND":     {Domain: "approvals", Summary: "Approval id not found"},
}
