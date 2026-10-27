package clab

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	clabcore "github.com/srl-labs/containerlab/core"
	clabexec "github.com/srl-labs/containerlab/exec"
)

func TestApplyResultToResponseCopiesFields(t *testing.T) {
	result := &clabcore.ApplyResult{
		DryRun:           true,
		DeployedLab:      true,
		LabName:          "demo",
		AddedNodes:       []string{"n2"},
		DeletedNodes:     []string{"n3"},
		RecreatedNodes:   []string{"xrd1"},
		StartedNodes:     []string{"n4"},
		AddedLinks:       []string{"n1:eth1<-->n2:eth1"},
		DeletedEndpoints: []string{"n3:eth1"},
		RestartedNodes:   []string{"ceos1"},
		NodeChangeReasons: map[string]string{
			"ceos1": "added link",
		},
	}

	response := ApplyResultToResponse(result)

	if !response.DryRun {
		t.Fatal("expected dryRun=true")
	}
	if !response.DeployedLab {
		t.Fatal("expected deployedLab=true")
	}
	if response.LabName != "demo" {
		t.Fatalf("labName = %q, want demo", response.LabName)
	}
	if got := response.AddedNodes[0]; got != "n2" {
		t.Fatalf("addedNodes[0] = %q, want n2", got)
	}
	if got := response.NodeChangeReasons["ceos1"]; got != "added link" {
		t.Fatalf("nodeChangeReasons[ceos1] = %q, want added link", got)
	}

	result.AddedNodes[0] = "mutated"
	result.NodeChangeReasons["ceos1"] = "mutated"
	if got := response.AddedNodes[0]; got != "n2" {
		t.Fatalf("response addedNodes was aliased, got %q", got)
	}
	if got := response.NodeChangeReasons["ceos1"]; got != "added link" {
		t.Fatalf("response nodeChangeReasons was aliased, got %q", got)
	}
}

func TestApplyResultToResponseHandlesNil(t *testing.T) {
	response := ApplyResultToResponse(nil)
	if response.AddedNodes == nil {
		t.Fatal("expected non-nil addedNodes")
	}
	if response.NodeChangeReasons == nil {
		t.Fatal("expected non-nil nodeChangeReasons")
	}
}

func TestApplyResultToResponseHandlesNilFields(t *testing.T) {
	response := ApplyResultToResponse(&clabcore.ApplyResult{})
	if response.AddedNodes == nil {
		t.Fatal("expected non-nil addedNodes")
	}
	if response.DeletedEndpoints == nil {
		t.Fatal("expected non-nil deletedEndpoints")
	}
	if response.NodeChangeReasons == nil {
		t.Fatal("expected non-nil nodeChangeReasons")
	}
}

func TestApplyResultToResponseJSONUsesEmptyArrays(t *testing.T) {
	response := ApplyResultToResponse(&clabcore.ApplyResult{
		AddedNodes: []string{"srl3"},
	})

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	body := string(data)
	if strings.Contains(body, "null") {
		t.Fatalf("expected empty collections to marshal as arrays/maps, got %s", body)
	}

	for _, field := range []string{
		`"deletedNodes":[]`,
		`"recreatedNodes":[]`,
		`"startedNodes":[]`,
		`"addedLinks":[]`,
		`"deletedEndpoints":[]`,
		`"restartedNodes":[]`,
		`"nodeChangeReasons":{}`,
	} {
		if !strings.Contains(body, field) {
			t.Fatalf("expected marshaled response to contain %s, got %s", field, body)
		}
	}
}

func TestExecCollectionToExecResponseKeepsResults(t *testing.T) {
	ec := clabexec.NewExecCollection()
	add := func(cmd []string, rc int, stdout, stderr string) {
		r := clabexec.NewExecResult(clabexec.NewExecCmdFromSlice(cmd))
		r.SetReturnCode(rc)
		r.SetStdOut([]byte(stdout))
		r.SetStdErr([]byte(stderr))
		ec.Add("clab-demo-r1", r)
	}
	add([]string{"sh", "-c", "exit 3"}, 3, "plain text\n", "boom\n")
	add([]string{"vtysh", "-c", "show bgp summary json"}, 0, "{\n  \"ipv4Unicast\": {\"peerCount\": 2}\n}\n", "")
	add([]string{"echo", "5"}, 0, "5\n", "")

	got := ExecCollectionToExecResponse(ec)["clab-demo-r1"]
	if len(got) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(got), got)
	}
	if got[0].ReturnCode != 3 || got[0].Stdout != "plain text\n" || got[0].Stderr != "boom\n" {
		t.Errorf("failing command = %+v", got[0])
	}
	if !slices.Equal(got[0].Cmd, []string{"sh", "-c", "exit 3"}) {
		t.Errorf("cmd = %q", got[0].Cmd)
	}
	if got[1].Stdout != `{"ipv4Unicast":{"peerCount":2}}` {
		t.Errorf("JSON output = %q", got[1].Stdout)
	}
	if got[2].Stdout != "5" {
		t.Errorf("number output = %q", got[2].Stdout)
	}
}
