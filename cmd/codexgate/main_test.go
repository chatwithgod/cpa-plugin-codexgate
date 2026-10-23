package main

import (
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"testing"
)

func TestNativeABIVersion(t *testing.T) {
	if nativeABIVersion != pluginabi.ABIVersion {
		t.Fatal("C ABI bridge and pinned CPA SDK disagree")
	}
}
