// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Storage handles all operations related to the database.
type Storage struct {
	db *sql.DB
}

// NewStorage returns a new Storage.
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db}
}

// DatabaseVersion returns the version of the database which is in use.
func (s *Storage) DatabaseVersion() string {
	var dbVersion string
	err := s.db.QueryRow(`SELECT sqlite_version()`).Scan(&dbVersion)
	if err != nil {
		return err.Error()
	}

	return "SQLite " + dbVersion
}

// Ping checks if the database connection works.
func (s *Storage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.db.PingContext(ctx)
}

// DBStats returns database statistics.
func (s *Storage) DBStats() sql.DBStats {
	return s.db.Stats()
}

// DBSize returns how much size the database is using in a pretty way.
func (s *Storage) DBSize() (string, error) {
	var pageCount int64
	var pageSize int64

	err := s.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	if err != nil {
		return "", err
	}

	err = s.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	if err != nil {
		return "", err
	}

	totalBytes := pageCount * pageSize

	// Format size in human readable format
	if totalBytes < 1024 {
		return fmt.Sprintf("%d bytes", totalBytes), nil
	} else if totalBytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(totalBytes)/1024), nil
	} else if totalBytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(totalBytes)/(1024*1024)), nil
	} else {
		return fmt.Sprintf("%.1f GB", float64(totalBytes)/(1024*1024*1024)), nil
	}
}
