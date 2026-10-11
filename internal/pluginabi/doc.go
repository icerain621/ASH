// Package pluginabi is the out-of-process gRPC Plugin ABI (Indexer and similar).
//
// In-process agent/memory component registration lives in internal/plugins (B03).
// Keep naming and registration surfaces separate: do not register gRPC ABI ids
// on the in-process Host, and do not treat process-in component ids as ABI plugins.
package pluginabi
