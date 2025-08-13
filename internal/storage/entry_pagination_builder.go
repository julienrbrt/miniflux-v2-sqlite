// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"miniflux.app/v2/internal/model"
)

// EntryPaginationBuilder is a builder for entry prev/next queries.
type EntryPaginationBuilder struct {
	store      *Storage
	conditions []string
	args       []any
	entryID    int64
	order      string
	direction  string
}

// WithSearchQuery adds basic text search query to the condition.
func (e *EntryPaginationBuilder) WithSearchQuery(query string) {
	if query != "" {
		e.conditions = append(e.conditions, fmt.Sprintf("(e.title LIKE $%d OR e.content LIKE $%d)", len(e.args)+1, len(e.args)+1))
		e.args = append(e.args, "%"+query+"%")
	}
}

// WithStarred adds starred to the condition.
func (e *EntryPaginationBuilder) WithStarred() {
	e.conditions = append(e.conditions, "e.starred = 1")
}

// WithFeedID adds feed_id to the condition.
func (e *EntryPaginationBuilder) WithFeedID(feedID int64) {
	if feedID != 0 {
		e.conditions = append(e.conditions, "e.feed_id = $"+strconv.Itoa(len(e.args)+1))
		e.args = append(e.args, feedID)
	}
}

// WithCategoryID adds category_id to the condition.
func (e *EntryPaginationBuilder) WithCategoryID(categoryID int64) {
	if categoryID != 0 {
		e.conditions = append(e.conditions, "f.category_id = $"+strconv.Itoa(len(e.args)+1))
		e.args = append(e.args, categoryID)
	}
}

// WithStatus adds status to the condition.
func (e *EntryPaginationBuilder) WithStatus(status string) {
	if status != "" {
		e.conditions = append(e.conditions, "e.status = $"+strconv.Itoa(len(e.args)+1))
		e.args = append(e.args, status)
	}
}

func (e *EntryPaginationBuilder) WithTags(tags []string) {
	if len(tags) > 0 {
		for _, tag := range tags {
			e.conditions = append(e.conditions, fmt.Sprintf("e.tags LIKE $%d", len(e.args)+1))
			e.args = append(e.args, "%\""+strings.ToLower(tag)+"\"%")
		}
	}
}

// WithGloballyVisible adds global visibility to the condition.
func (e *EntryPaginationBuilder) WithGloballyVisible() {
	e.conditions = append(e.conditions, "c.hide_globally = 0")
	e.conditions = append(e.conditions, "f.hide_globally = 0")
}

// Entries returns previous and next entries.
func (e *EntryPaginationBuilder) Entries() (*model.Entry, *model.Entry, error) {
	tx, err := e.store.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction for entry pagination: %v", err)
	}

	prevID, nextID, err := e.getPrevNextID(tx)
	if err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	prevEntry, err := e.getEntry(tx, prevID)
	if err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	nextEntry, err := e.getEntry(tx, nextID)
	if err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	tx.Commit()

	if e.direction == "desc" {
		return nextEntry, prevEntry, nil
	}

	return prevEntry, nextEntry, nil
}

func (e *EntryPaginationBuilder) getPrevNextID(tx *sql.Tx) (prevID int64, nextID int64, err error) {
	// SQLite doesn't have window functions in older versions, so we'll use subqueries
	subCondition := strings.Join(e.conditions, " AND ")

	// Get previous entry ID
	prevQuery := fmt.Sprintf(`
		SELECT e.id
		FROM entries AS e
		JOIN feeds AS f ON f.id=e.feed_id
		JOIN categories c ON c.id = f.category_id
		WHERE %s AND (e.%s < (SELECT %s FROM entries WHERE id = ?) OR (e.%s = (SELECT %s FROM entries WHERE id = ?) AND e.id > ?))
		ORDER BY e.%s DESC, e.created_at DESC, e.id ASC
		LIMIT 1
	`, subCondition, e.order, e.order, e.order, e.order, e.order)

	// Get next entry ID
	nextQuery := fmt.Sprintf(`
		SELECT e.id
		FROM entries AS e
		JOIN feeds AS f ON f.id=e.feed_id
		JOIN categories c ON c.id = f.category_id
		WHERE %s AND (e.%s > (SELECT %s FROM entries WHERE id = ?) OR (e.%s = (SELECT %s FROM entries WHERE id = ?) AND e.id < ?))
		ORDER BY e.%s ASC, e.created_at ASC, e.id DESC
		LIMIT 1
	`, subCondition, e.order, e.order, e.order, e.order, e.order)

	args := append(e.args, e.entryID, e.entryID, e.entryID)

	var pID, nID sql.NullInt64

	// Get previous ID
	err = tx.QueryRow(prevQuery, args...).Scan(&pID)
	if err != nil && err != sql.ErrNoRows {
		return 0, 0, fmt.Errorf("entry pagination prev: %v", err)
	}

	// Get next ID
	err = tx.QueryRow(nextQuery, args...).Scan(&nID)
	if err != nil && err != sql.ErrNoRows {
		return 0, 0, fmt.Errorf("entry pagination next: %v", err)
	}

	if pID.Valid {
		prevID = pID.Int64
	}

	if nID.Valid {
		nextID = nID.Int64
	}

	return prevID, nextID, nil
}

func (e *EntryPaginationBuilder) getEntry(tx *sql.Tx, entryID int64) (*model.Entry, error) {
	var entry model.Entry

	err := tx.QueryRow(`SELECT id, title FROM entries WHERE id = ?`, entryID).Scan(
		&entry.ID,
		&entry.Title,
	)

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("fetching sibling entry: %v", err)
	}

	return &entry, nil
}

// NewEntryPaginationBuilder returns a new EntryPaginationBuilder.
func NewEntryPaginationBuilder(store *Storage, userID, entryID int64, order, direction string) *EntryPaginationBuilder {
	return &EntryPaginationBuilder{
		store:      store,
		args:       []any{userID, "removed"},
		conditions: []string{"e.user_id = ?", "e.status <> ?"},
		entryID:    entryID,
		order:      order,
		direction:  direction,
	}
}
