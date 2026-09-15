package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentWorkspaceAPILifecycleAndSessionLink(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_AGENT_EXECUTOR", "static")
	r, _ := newPlatformTestRouter(t)

	createWS := httptest.NewRecorder()
	createWSReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent-workspaces",
		bytes.NewReader([]byte(`{"title":"Ship feat","repoRoot":"."}`)))
	createWSReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(createWS, createWSReq)
	if createWS.Code != http.StatusCreated {
		t.Fatalf("create workspace status=%d body=%s", createWS.Code, createWS.Body.String())
	}
	var ws struct {
		ID         string   `json:"id"`
		Title      string   `json:"title"`
		SessionIDs []string `json:"sessionIds"`
		Status     string   `json:"status"`
	}
	if err := json.Unmarshal(createWS.Body.Bytes(), &ws); err != nil {
		t.Fatal(err)
	}
	if ws.ID == "" || ws.Title != "Ship feat" || ws.Status != "active" {
		t.Fatalf("ws=%+v", ws)
	}

	listW := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent-workspaces?limit=50", nil)
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listW.Code, listW.Body.String())
	}
	var listResp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatal(err)
	}
	if len(listResp.Items) != 1 || listResp.Items[0].ID != ws.ID {
		t.Fatalf("list=%+v", listResp)
	}

	createSess := httptest.NewRecorder()
	createSessReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sessions",
		bytes.NewReader([]byte(`{"workspaceId":"`+ws.ID+`"}`)))
	createSessReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(createSess, createSessReq)
	if createSess.Code != http.StatusCreated {
		t.Fatalf("create session status=%d body=%s", createSess.Code, createSess.Body.String())
	}
	var sess struct {
		ID          string `json:"id"`
		WorkspaceID string `json:"workspaceId"`
	}
	if err := json.Unmarshal(createSess.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.ID == "" || sess.WorkspaceID != ws.ID {
		t.Fatalf("sess=%+v", sess)
	}

	getWS := httptest.NewRecorder()
	// re-list to see sessionIds
	list2 := httptest.NewRecorder()
	r.ServeHTTP(list2, httptest.NewRequest(http.MethodGet, "/api/v1/agent-workspaces?limit=50", nil))
	var list2Resp struct {
		Items []struct {
			ID         string   `json:"id"`
			SessionIDs []string `json:"sessionIds"`
		} `json:"items"`
	}
	if err := json.Unmarshal(list2.Body.Bytes(), &list2Resp); err != nil {
		t.Fatal(err)
	}
	if len(list2Resp.Items) != 1 || len(list2Resp.Items[0].SessionIDs) != 1 || list2Resp.Items[0].SessionIDs[0] != sess.ID {
		t.Fatalf("workspace after create session=%+v", list2Resp)
	}

	attachW := httptest.NewRecorder()
	attachReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent-workspaces/"+ws.ID+"/sessions",
		bytes.NewReader([]byte(`{"sessionId":"sess_extra"}`)))
	attachReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(attachW, attachReq)
	if attachW.Code != http.StatusOK {
		t.Fatalf("attach status=%d body=%s", attachW.Code, attachW.Body.String())
	}
	var attached struct {
		SessionIDs []string `json:"sessionIds"`
	}
	if err := json.Unmarshal(attachW.Body.Bytes(), &attached); err != nil {
		t.Fatal(err)
	}
	if len(attached.SessionIDs) != 2 {
		t.Fatalf("attached=%+v", attached)
	}

	patchW := httptest.NewRecorder()
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/agent-workspaces/"+ws.ID,
		bytes.NewReader([]byte(`{"title":"Renamed WS"}`)))
	patchReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", patchW.Code, patchW.Body.String())
	}
	var patched struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(patchW.Body.Bytes(), &patched); err != nil {
		t.Fatal(err)
	}
	if patched.Title != "Renamed WS" {
		t.Fatalf("patched=%+v", patched)
	}

	detachW := httptest.NewRecorder()
	detachReq := httptest.NewRequest(http.MethodDelete, "/api/v1/agent-workspaces/"+ws.ID+"/sessions/sess_extra", nil)
	r.ServeHTTP(detachW, detachReq)
	if detachW.Code != http.StatusOK {
		t.Fatalf("detach status=%d body=%s", detachW.Code, detachW.Body.String())
	}
	var detached struct {
		SessionIDs []string `json:"sessionIds"`
	}
	if err := json.Unmarshal(detachW.Body.Bytes(), &detached); err != nil {
		t.Fatal(err)
	}
	if len(detached.SessionIDs) != 1 || detached.SessionIDs[0] != sess.ID {
		t.Fatalf("detached=%+v want only %s", detached, sess.ID)
	}

	delW := httptest.NewRecorder()
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/agent-workspaces/"+ws.ID, nil)
	r.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", delW.Code, delW.Body.String())
	}

	list3 := httptest.NewRecorder()
	r.ServeHTTP(list3, httptest.NewRequest(http.MethodGet, "/api/v1/agent-workspaces?limit=50", nil))
	var list3Resp struct {
		Items []any `json:"items"`
	}
	if err := json.Unmarshal(list3.Body.Bytes(), &list3Resp); err != nil {
		t.Fatal(err)
	}
	if len(list3Resp.Items) != 0 {
		t.Fatalf("list after soft close=%+v want empty", list3Resp)
	}

	_ = getWS
}
