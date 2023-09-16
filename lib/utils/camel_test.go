package utils

import "testing"

func TestToCamel(t *testing.T) {
	if ToCamel("target_uuid") != "TargetUUID" {
		t.Errorf("ToCamel should replace Uuid with UUID: %s", ToCamel("target_uuid"))
	}

	if ToCamel("target_type") != "TargetType" {
		t.Errorf("ToCamel should convert to camel case: %s", ToCamel("target_type"))
	}

}
