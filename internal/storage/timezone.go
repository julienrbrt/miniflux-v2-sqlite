// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

// Timezones returns all timezones supported by the application.
// Since SQLite doesn't have built-in timezone functions like PostgreSQL,
// we return a predefined list of common timezones.
func (s *Storage) Timezones() (map[string]string, error) {
	timezones := map[string]string{
		"UTC":                 "UTC",
		"America/New_York":    "America/New_York",
		"America/Chicago":     "America/Chicago",
		"America/Denver":      "America/Denver",
		"America/Los_Angeles": "America/Los_Angeles",
		"America/Toronto":     "America/Toronto",
		"America/Vancouver":   "America/Vancouver",
		"America/Sao_Paulo":   "America/Sao_Paulo",
		"Europe/London":       "Europe/London",
		"Europe/Paris":        "Europe/Paris",
		"Europe/Berlin":       "Europe/Berlin",
		"Europe/Rome":         "Europe/Rome",
		"Europe/Madrid":       "Europe/Madrid",
		"Europe/Amsterdam":    "Europe/Amsterdam",
		"Europe/Stockholm":    "Europe/Stockholm",
		"Europe/Helsinki":     "Europe/Helsinki",
		"Europe/Moscow":       "Europe/Moscow",
		"Asia/Tokyo":          "Asia/Tokyo",
		"Asia/Shanghai":       "Asia/Shanghai",
		"Asia/Hong_Kong":      "Asia/Hong_Kong",
		"Asia/Singapore":      "Asia/Singapore",
		"Asia/Seoul":          "Asia/Seoul",
		"Asia/Kolkata":        "Asia/Kolkata",
		"Asia/Dubai":          "Asia/Dubai",
		"Australia/Sydney":    "Australia/Sydney",
		"Australia/Melbourne": "Australia/Melbourne",
		"Pacific/Auckland":    "Pacific/Auckland",
	}

	return timezones, nil
}
