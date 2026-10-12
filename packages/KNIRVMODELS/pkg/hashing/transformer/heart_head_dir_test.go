package transformer

import "testing"

func TestHEADDirFollowsTheAppDataRootStageFivePublishesTo(t *testing.T) {
	t.Setenv("KNIRV_HEAD_DIR", "")
	t.Setenv("KNIRV_APP_DATA_DIR", "")
	if headDir() != DefaultHEADDir {
		t.Fatal(headDir())
	}
	t.Setenv("KNIRV_APP_DATA_DIR", "/data/k")
	if got := headDir(); got != "/data/k/knirvmodels/data/head" {
		t.Fatal(got)
	}
	t.Setenv("KNIRV_HEAD_DIR", "/explicit")
	if headDir() != "/explicit" {
		t.Fatal(headDir())
	}
}
