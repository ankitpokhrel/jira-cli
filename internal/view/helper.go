package view

import "time"

func formatDateTime(dt string) string {
	const rfc3339 = "2006-01-02T15:04:05-0700"

	t, err := time.Parse(rfc3339, dt)
	if err != nil {
		return dt
	}

	return t.Format("2006-01-02 15:04:05")
}
