package view

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestServerInfoRender(t *testing.T) {
	var b bytes.Buffer

	data := &jira.ServerInfo{
		Version:        "9.1.0",
		VersionNumbers: []int{9, 1, 0},
		DeploymentType: "Server",
		BuildNumber:    901000,
		DefaultLocale: struct {
			Locale string `json:"locale"`
		}{Locale: "en_US"},
	}

	serverInfo := NewServerInfo(data, WithServerInfoWriter(&b))
	assert.NoError(t, serverInfo.Render())

	expected := fmt.Sprintf(`SERVER INFO
-----------

Version: 	 %s
Build Number: 	 %d
Deployment Type: %s
Default Locale:  %s
`, data.Version, data.BuildNumber, data.DeploymentType, data.DefaultLocale.Locale)

	assert.Equal(t, expected, b.String())
}
