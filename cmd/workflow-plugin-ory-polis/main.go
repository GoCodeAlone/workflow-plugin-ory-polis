package main

import (
	"github.com/GoCodeAlone/workflow-plugin-ory-polis/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewOryPolisPlugin(), sdk.WithBuildVersion(sdk.ResolveBuildVersion(internal.Version)))
}
