package view

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	worklogFieldID          = "ID"
	worklogFieldAuthor      = "AUTHOR"
	worklogFieldStarted     = "STARTED"
	worklogFieldTimeSpent   = "TIME SPENT"
	worklogFieldCreated     = "CREATED"
	worklogFieldUpdated     = "UPDATED"
	worklogFieldComment     = "COMMENT"
)

// WorklogList is a list view for worklogs.
type WorklogList struct {
	Project  string
	Server   string
	Worklogs []jira.Worklog
	Total    int
	Display  DisplayFormat
}

// Render renders the worklog list view.
func (wl WorklogList) Render() error {
	if wl.Display.Plain {
		return wl.renderPlain(os.Stdout)
	}
	return wl.renderTable()
}

func (wl WorklogList) renderPlain(w io.Writer) error {
	for i, worklog := range wl.Worklogs {
		fmt.Fprintf(w, "Worklog #%d\n", i+1)
		fmt.Fprintf(w, "  ID:          %s\n", worklog.ID)
		fmt.Fprintf(w, "  Author:      %s\n", worklog.Author.Name)
		fmt.Fprintf(w, "  Started:     %s\n", formatWorklogDate(worklog.Started))
		fmt.Fprintf(w, "  Time Spent:  %s (%d seconds)\n", worklog.TimeSpent, worklog.TimeSpentSeconds)
		fmt.Fprintf(w, "  Created:     %s\n", formatWorklogDate(worklog.Created))
		fmt.Fprintf(w, "  Updated:     %s\n", formatWorklogDate(worklog.Updated))
		
		if worklog.Comment != nil {
			comment := extractWorklogComment(worklog.Comment)
			if comment != "" {
				fmt.Fprintf(w, "  Comment:     %s\n", truncateString(comment, 60))
			}
		}
		
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "Total worklogs: %d\n", wl.Total)
	
	return nil
}

func (wl WorklogList) renderTable() error {
	data := wl.data()
	tw := tabwriter.NewWriter(os.Stdout, 0, tabWidth, 1, '\t', 0)

	headers := []string{
		worklogFieldID,
		worklogFieldAuthor,
		worklogFieldStarted,
		worklogFieldTimeSpent,
		worklogFieldCreated,
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	for _, row := range data {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	return tw.Flush()
}

func (wl WorklogList) data() [][]string {
	var data [][]string

	for _, worklog := range wl.Worklogs {
		data = append(data, []string{
			worklog.ID,
			worklog.Author.Name,
			formatWorklogDate(worklog.Started),
			worklog.TimeSpent,
			formatWorklogDate(worklog.Created),
		})
	}

	return data
}

func formatWorklogDate(dateStr string) string {
	formats := []string{
		time.RFC3339,
		jira.RFC3339,
		jira.RFC3339MilliLayout,
		"2006-01-02T15:04:05.000-0700",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("2006-01-02 15:04")
		}
	}

	return dateStr
}

func extractWorklogComment(comment interface{}) string {
	if comment == nil {
		return ""
	}

	switch v := comment.(type) {
	case string:
		return v
	case map[string]interface{}:
		// Handle ADF format
		if content, ok := v["content"].([]interface{}); ok {
			var text strings.Builder
			extractTextFromADF(content, &text)
			return text.String()
		}
	}

	return ""
}

func extractTextFromADF(content []interface{}, builder *strings.Builder) {
	for _, item := range content {
		if node, ok := item.(map[string]interface{}); ok {
			if nodeType, ok := node["type"].(string); ok {
				if nodeType == "text" {
					if text, ok := node["text"].(string); ok {
						builder.WriteString(text)
					}
				}
			}
			if subContent, ok := node["content"].([]interface{}); ok {
				extractTextFromADF(subContent, builder)
			}
		}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
