package utils

import "testing"

func TestGetJSONKeys(t *testing.T) {
	var jsonStr = `
	{
		"Name": "test",
		"TableName": "test",
		"TemplateID": "test",
		"TemplateInfo": "test",
		"Limit": 0
}`
	keys, err := GetJSONKeys(jsonStr)
	if err != nil {
		t.Fatalf("GetJSONKeys failed: %v", err)
		return
	}
	if len(keys) != 5 {
		t.Fatalf("expected 5 keys, got %d", len(keys))
		return
	}
	if keys[0] != "Name" {
		t.Errorf("expected key 0 to be Name, got %s", keys[0])

		return
	}
	if keys[1] != "TableName" {
		t.Errorf("expected key 1 to be TableName, got %s", keys[1])

		return
	}
	if keys[2] != "TemplateID" {
		t.Errorf("expected key 2 to be TemplateID, got %s", keys[2])

		return
	}
	if keys[3] != "TemplateInfo" {
		t.Errorf("expected key 3 to be TemplateInfo, got %s", keys[3])

		return
	}
	if keys[4] != "Limit" {
		t.Errorf("expected key 4 to be Limit, got %s", keys[4])

		return
	}
}
