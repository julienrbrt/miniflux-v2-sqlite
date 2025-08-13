// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"miniflux.app/v2/internal/model"
)

// AnotherCategoryExists checks if another category exists with the same title.
func (s *Storage) AnotherCategoryExists(userID, categoryID int64, title string) bool {
	var result bool
	query := `SELECT true FROM categories WHERE user_id=? AND id != ? AND lower(title)=lower(?) LIMIT 1`
	s.db.QueryRow(query, userID, categoryID, title).Scan(&result)
	return result
}

// CategoryTitleExists checks if the given category exists into the database.
func (s *Storage) CategoryTitleExists(userID int64, title string) bool {
	var result bool
	query := `SELECT true FROM categories WHERE user_id=? AND lower(title)=lower(?) LIMIT 1`
	s.db.QueryRow(query, userID, title).Scan(&result)
	return result
}

// CategoryIDExists checks if the given category exists into the database.
func (s *Storage) CategoryIDExists(userID, categoryID int64) bool {
	var result bool
	query := `SELECT true FROM categories WHERE user_id=? AND id=? LIMIT 1`
	s.db.QueryRow(query, userID, categoryID).Scan(&result)
	return result
}

// Category returns a category from the database.
func (s *Storage) Category(userID, categoryID int64) (*model.Category, error) {
	var category model.Category

	query := `SELECT id, user_id, title, hide_globally FROM categories WHERE user_id=? AND id=?`
	err := s.db.QueryRow(query, userID, categoryID).Scan(&category.ID, &category.UserID, &category.Title, &category.HideGlobally)

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf(`store: unable to fetch category: %v`, err)
	default:
		return &category, nil
	}
}

// FirstCategory returns the first category for the given user.
func (s *Storage) FirstCategory(userID int64) (*model.Category, error) {
	query := `SELECT id, user_id, title, hide_globally FROM categories WHERE user_id=? ORDER BY title ASC LIMIT 1`

	var category model.Category
	err := s.db.QueryRow(query, userID).Scan(&category.ID, &category.UserID, &category.Title, &category.HideGlobally)

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf(`store: unable to fetch category: %v`, err)
	default:
		return &category, nil
	}
}

// CategoryByTitle finds a category by the title.
func (s *Storage) CategoryByTitle(userID int64, title string) (*model.Category, error) {
	var category model.Category

	query := `SELECT id, user_id, title, hide_globally FROM categories WHERE user_id=? AND title=?`
	err := s.db.QueryRow(query, userID, title).Scan(&category.ID, &category.UserID, &category.Title, &category.HideGlobally)

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf(`store: unable to fetch category: %v`, err)
	default:
		return &category, nil
	}
}

// Categories returns all categories that belongs to the given user.
func (s *Storage) Categories(userID int64) (model.Categories, error) {
	query := `SELECT id, user_id, title, hide_globally FROM categories WHERE user_id=? ORDER BY title ASC`
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch categories: %v`, err)
	}
	defer rows.Close()

	categories := make(model.Categories, 0)
	for rows.Next() {
		var category model.Category
		if err := rows.Scan(&category.ID, &category.UserID, &category.Title, &category.HideGlobally); err != nil {
			return nil, fmt.Errorf(`store: unable to fetch category row: %v`, err)
		}

		categories = append(categories, &category)
	}

	return categories, nil
}

// CategoriesWithFeedCount returns all categories with the number of feeds.
func (s *Storage) CategoriesWithFeedCount(userID int64) (model.Categories, error) {
	user, err := s.UserByID(userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			c.id,
			c.user_id,
			c.title,
			c.hide_globally,
			(SELECT count(*) FROM feeds WHERE feeds.category_id=c.id) AS count,
			(SELECT count(*)
			   FROM feeds
			     JOIN entries ON (feeds.id = entries.feed_id)
			   WHERE feeds.category_id = c.id AND entries.status = ?) AS count_unread
		FROM categories c
		WHERE
			user_id=?
	`

	if user.CategoriesSortingOrder == "alphabetical" {
		query += `
			ORDER BY
				c.title ASC
		`
	} else {
		query += `
			ORDER BY
				count_unread DESC,
				c.title ASC
		`
	}

	rows, err := s.db.Query(query, model.EntryStatusUnread, userID)
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch categories: %v`, err)
	}
	defer rows.Close()

	categories := make(model.Categories, 0)
	for rows.Next() {
		var category model.Category
		if err := rows.Scan(&category.ID, &category.UserID, &category.Title, &category.HideGlobally, &category.FeedCount, &category.TotalUnread); err != nil {
			return nil, fmt.Errorf(`store: unable to fetch category row: %v`, err)
		}

		categories = append(categories, &category)
	}

	return categories, nil
}

// CreateCategory creates a new category.
func (s *Storage) CreateCategory(userID int64, request *model.CategoryCreationRequest) (*model.Category, error) {
	query := `
		INSERT INTO categories
			(user_id, title, hide_globally)
		VALUES
			(?, ?, ?)
	`
	result, err := s.db.Exec(
		query,
		userID,
		request.Title,
		request.HideGlobally,
	)

	if err != nil {
		return nil, fmt.Errorf(`store: unable to create category %q for user ID %d: %v`, request.Title, userID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf(`store: unable to get category ID: %v`, err)
	}

	// Get the created category
	var category model.Category
	err = s.db.QueryRow(`
		SELECT id, user_id, title, hide_globally
		FROM categories WHERE id = ?`, id).Scan(
		&category.ID,
		&category.UserID,
		&category.Title,
		&category.HideGlobally,
	)

	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch created category: %v`, err)
	}

	return &category, nil
}

// UpdateCategory updates an existing category.
func (s *Storage) UpdateCategory(category *model.Category) error {
	query := `UPDATE categories SET title=?, hide_globally=? WHERE id=? AND user_id=?`
	_, err := s.db.Exec(
		query,
		category.Title,
		category.HideGlobally,
		category.ID,
		category.UserID,
	)

	if err != nil {
		return fmt.Errorf(`store: unable to update category: %v`, err)
	}

	return nil
}

// RemoveCategory deletes a category.
func (s *Storage) RemoveCategory(userID, categoryID int64) error {
	query := `DELETE FROM categories WHERE id = ? AND user_id = ?`
	result, err := s.db.Exec(query, categoryID, userID)
	if err != nil {
		return fmt.Errorf(`store: unable to remove this category: %v`, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(`store: unable to remove this category: %v`, err)
	}

	if count == 0 {
		return errors.New(`store: no category has been removed`)
	}

	return nil
}

// delete the given categories, replacing those categories with the user's first
// category on affected feeds
func (s *Storage) RemoveAndReplaceCategoriesByName(userid int64, titles []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return errors.New("store: unable to begin transaction")
	}

	var count int
	query := "SELECT count(*) FROM categories WHERE user_id = $1 and title != ANY($2)"
	// For SQLite, we need to use IN clause with placeholders
	placeholders := make([]string, len(titles))
	args := make([]interface{}, len(titles)+1)
	args[0] = userid
	for i, title := range titles {
		placeholders[i] = "?"
		args[i+1] = title
	}
	query = fmt.Sprintf("SELECT count(*) FROM categories WHERE user_id = ? and title NOT IN (%s)", strings.Join(placeholders, ","))
	err = tx.QueryRow(query, args...).Scan(&count)
	if err != nil {
		tx.Rollback()
		return errors.New("store: unable to retrieve category count")
	}
	if count < 1 {
		tx.Rollback()
		return errors.New("store: at least 1 category must remain after deletion")
	}

	// Get category IDs to delete
	placeholders = make([]string, len(titles))
	args = make([]interface{}, len(titles)+1)
	args[0] = userid
	for i, title := range titles {
		placeholders[i] = "?"
		args[i+1] = title
	}

	// Get the first remaining category ID
	var firstCategoryID int64
	query = fmt.Sprintf("SELECT id FROM categories WHERE user_id = ? AND title NOT IN (%s) ORDER BY title ASC LIMIT 1", strings.Join(placeholders, ","))
	err = tx.QueryRow(query, args...).Scan(&firstCategoryID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("store: unable to find replacement category: %v", err)
	}

	// Update feeds to use the first remaining category
	placeholders = make([]string, len(titles))
	args = make([]interface{}, len(titles)+2)
	args[0] = firstCategoryID
	args[1] = userid
	for i, title := range titles {
		placeholders[i] = "?"
		args[i+2] = title
	}
	query = fmt.Sprintf("UPDATE feeds SET category_id = ? WHERE user_id = ? AND category_id IN (SELECT id FROM categories WHERE user_id = ? AND title IN (%s))", strings.Join(placeholders, ","))
	args = append([]interface{}{firstCategoryID, userid, userid}, args[2:]...)
	_, err = tx.Exec(query, args...)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("store: unable to replace categories: %v", err)
	}

	placeholders = make([]string, len(titles))
	args = make([]interface{}, len(titles)+1)
	args[0] = userid
	for i, title := range titles {
		placeholders[i] = "?"
		args[i+1] = title
	}
	query = fmt.Sprintf("DELETE FROM categories WHERE user_id = ? AND title IN (%s)", strings.Join(placeholders, ","))
	_, err = tx.Exec(query, args...)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("store: unable to delete categories: %v", err)
	}
	tx.Commit()
	return nil
}
