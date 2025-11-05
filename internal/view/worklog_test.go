package view

import (
	"strings"
	"testing"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestWorklogList_Render(t *testing.T) {
	tests := []struct {
		name     string
		worklogs []jira.Worklog
		display  DisplayFormat
		wantErr  bool
	}{
		{
			name:     "empty worklogs",
			worklogs: []jira.Worklog{},
			display:  DisplayFormat{Plain: true},
			wantErr:  false,
		},
		{
			name: "single worklog plain mode",
			worklogs: []jira.Worklog{
				{
					ID: "12345",
					Author: jira.User{
						Name: "John Doe",
					},
					Started:          "2024-11-05T10:30:00.000+0000",
					TimeSpent:        "2h 30m",
					TimeSpentSeconds: 9000,
					Created:          "2024-11-05T10:30:00.000+0000",
					Updated:          "2024-11-05T10:30:00.000+0000",
					Comment:          "Test comment",
				},
			},
			display: DisplayFormat{Plain: true},
			wantErr: false,
		},
		{
			name: "multiple worklogs table mode",
			worklogs: []jira.Worklog{
				{
					ID: "12345",
					Author: jira.User{
						Name: "John Doe",
					},
					Started:          "2024-11-05T10:30:00.000+0000",
					TimeSpent:        "2h 30m",
					TimeSpentSeconds: 9000,
					Created:          "2024-11-05T10:30:00.000+0000",
					Updated:          "2024-11-05T10:30:00.000+0000",
				},
				{
					ID: "12346",
					Author: jira.User{
						Name: "Jane Smith",
					},
					Started:          "2024-11-05T14:00:00.000+0000",
					TimeSpent:        "1h 15m",
					TimeSpentSeconds: 4500,
					Created:          "2024-11-05T14:00:00.000+0000",
					Updated:          "2024-11-05T14:00:00.000+0000",
				},
			},
			display: DisplayFormat{Plain: false},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := WorklogList{
				Project:  "TEST",
				Server:   "https://test.atlassian.net",
				Worklogs: tt.worklogs,
				Total:    len(tt.worklogs),
				Display:  tt.display,
			}

			// Just test that it doesn't panic or error
			// Full output testing would require capturing stdout
			err := wl.Render()
			if (err != nil) != tt.wantErr {
				t.Errorf("WorklogList.Render() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatWorklogDate(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		wantFmt  string
		wantFail bool
	}{
		{
			name:    "RFC3339 with milliseconds",
			dateStr: "2024-11-05T10:30:00.000+0000",
			wantFmt: "2024-11-05 10:30",
		},
		{
			name:    "RFC3339",
			dateStr: "2024-11-05T10:30:00Z",
			wantFmt: "2024-11-05 10:30",
		},
		{
			name:     "invalid date",
			dateStr:  "invalid",
			wantFmt:  "invalid", // Should return original string
			wantFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatWorklogDate(tt.dateStr)
			if got != tt.wantFmt {
				t.Errorf("formatWorklogDate() = %v, want %v", got, tt.wantFmt)
			}
		})
	}
}

func TestExtractWorklogComment(t *testing.T) {
	tests := []struct {
		name    string
		comment interface{}
		want    string
	}{
		{
			name:    "nil comment",
			comment: nil,
			want:    "",
		},
		{
			name:    "string comment",
			comment: "Simple text comment",
			want:    "Simple text comment",
		},
		{
			name: "ADF comment with text",
			comment: map[string]interface{}{
				"type":    "doc",
				"version": 1,
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{
								"type": "text",
								"text": "ADF formatted text",
							},
						},
					},
				},
			},
			want: "ADF formatted text",
		},
		{
			name: "ADF comment empty",
			comment: map[string]interface{}{
				"type":    "doc",
				"version": 1,
				"content": []interface{}{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWorklogComment(tt.comment)
			if got != tt.want {
				t.Errorf("extractWorklogComment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			s:      "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "exact length",
			s:      "1234567890",
			maxLen: 10,
			want:   "1234567890",
		},
		{
			name:   "long string",
			s:      "This is a very long string that needs truncation",
			maxLen: 20,
			want:   "This is a very lo...",
		},
		{
			name:   "empty string",
			s:      "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorklogList_Data(t *testing.T) {
	wl := WorklogList{
		Worklogs: []jira.Worklog{
			{
				ID: "12345",
				Author: jira.User{
					Name: "John Doe",
				},
				Started:   "2024-11-05T10:30:00.000+0000",
				TimeSpent: "2h 30m",
				Created:   "2024-11-05T10:30:00.000+0000",
			},
		},
	}

	data := wl.data()

	if len(data) != 1 {
		t.Errorf("Expected 1 row, got %d", len(data))
	}

	if len(data[0]) != 5 {
		t.Errorf("Expected 5 columns, got %d", len(data[0]))
	}

	if data[0][0] != "12345" {
		t.Errorf("Expected ID '12345', got '%s'", data[0][0])
	}

	if data[0][1] != "John Doe" {
		t.Errorf("Expected author 'John Doe', got '%s'", data[0][1])
	}
}

func TestExtractTextFromADF(t *testing.T) {
	content := []interface{}{
		map[string]interface{}{
			"type": "paragraph",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello ",
				},
				map[string]interface{}{
					"type": "text",
					"text": "World",
				},
			},
		},
	}

	var builder strings.Builder
	extractTextFromADF(content, &builder)

	got := builder.String()
	want := "Hello World"

	if got != want {
		t.Errorf("extractTextFromADF() = %v, want %v", got, want)
	}
}
