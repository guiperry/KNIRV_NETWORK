package transformer

import "testing"

func TestHRMDirFollowsTheAppDataRootStageFivePublishesTo(t *testing.T) {
	t.Setenv("KNIRV_HRM_DIR", "")
	t.Setenv("KNIRV_APP_DATA_DIR", "")
	if hrmDir() != DefaultHRMDir {
		t.Fatal(hrmDir())
	}
	t.Setenv("KNIRV_APP_DATA_DIR", "/data/k")
	if got := hrmDir(); got != "/data/k/knirvhasher/data/hrm" {
		t.Fatal(got)
	}
	t.Setenv("KNIRV_HRM_DIR", "/explicit")
	if hrmDir() != "/explicit" {
		t.Fatal(hrmDir())
	}
}
