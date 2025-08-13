// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"database/sql"
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/model"

	"golang.org/x/crypto/bcrypt"
)

// CountUsers returns the total number of users.
func (s *Storage) CountUsers() int {
	var result int
	err := s.db.QueryRow(`SELECT count(*) FROM users`).Scan(&result)
	if err != nil {
		return 0
	}

	return result
}

// SetLastLogin updates the last login date of a user.
func (s *Storage) SetLastLogin(userID int64) error {
	query := `UPDATE users SET last_login_at=datetime('now') WHERE id=?`
	_, err := s.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf(`store: unable to update last login date: %v`, err)
	}

	return nil
}

// UserExists checks if a user exists by using the given username.
func (s *Storage) UserExists(username string) bool {
	var result bool
	s.db.QueryRow(`SELECT true FROM users WHERE username=LOWER(?) LIMIT 1`, username).Scan(&result)
	return result
}

// AnotherUserExists checks if another user exists with the given username.
func (s *Storage) AnotherUserExists(userID int64, username string) bool {
	var result bool
	s.db.QueryRow(`SELECT true FROM users WHERE id != ? AND username=LOWER(?) LIMIT 1`, userID, username).Scan(&result)
	return result
}

// CreateUser creates a new user.
func (s *Storage) CreateUser(userCreationRequest *model.UserCreationRequest) (*model.User, error) {
	var hashedPassword string
	if userCreationRequest.Password != "" {
		var err error
		hashedPassword, err = crypto.HashPassword(userCreationRequest.Password)
		if err != nil {
			return nil, err
		}
	}

	query := `
		INSERT INTO users
			(username, password, is_admin, google_id, openid_connect_id)
		VALUES
			(LOWER(?), ?, ?, ?, ?)
	`

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf(`store: unable to start transaction: %v`, err)
	}

	result, err := tx.Exec(
		query,
		userCreationRequest.Username,
		hashedPassword,
		userCreationRequest.IsAdmin,
		userCreationRequest.GoogleID,
		userCreationRequest.OpenIDConnectID,
	)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf(`store: unable to create user: %v`, err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf(`store: unable to get user ID: %v`, err)
	}

	// Get the created user
	var user model.User
	err = tx.QueryRow(`
		SELECT id, username, is_admin, language, theme, timezone, entry_direction,
		       entries_per_page, keyboard_shortcuts, show_reading_time, entry_swipe,
		       gesture_nav, stylesheet, custom_js, external_font_hosts, google_id,
		       openid_connect_id, display_mode, entry_order, default_reading_speed,
		       cjk_reading_speed, default_home_page, categories_sorting_order,
		       mark_read_on_view, media_playback_rate, block_filter_entry_rules,
		       keep_filter_entry_rules, always_open_external_links, open_external_links_in_new_tab
		FROM users WHERE id = ?`, userID).Scan(
		&user.ID,
		&user.Username,
		&user.IsAdmin,
		&user.Language,
		&user.Theme,
		&user.Timezone,
		&user.EntryDirection,
		&user.EntriesPerPage,
		&user.KeyboardShortcuts,
		&user.ShowReadingTime,
		&user.EntrySwipe,
		&user.GestureNav,
		&user.Stylesheet,
		&user.CustomJS,
		&user.ExternalFontHosts,
		&user.GoogleID,
		&user.OpenIDConnectID,
		&user.DisplayMode,
		&user.EntryOrder,
		&user.DefaultReadingSpeed,
		&user.CJKReadingSpeed,
		&user.DefaultHomePage,
		&user.CategoriesSortingOrder,
		&user.MarkReadOnView,
		&user.MediaPlaybackRate,
		&user.BlockFilterEntryRules,
		&user.KeepFilterEntryRules,
		&user.AlwaysOpenExternalLinks,
		&user.OpenExternalLinksInNewTab,
	)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf(`store: unable to fetch created user: %v`, err)
	}

	_, err = tx.Exec(`INSERT INTO categories (user_id, title) VALUES (?, ?)`, user.ID, "All")
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf(`store: unable to create user default category: %v`, err)
	}

	_, err = tx.Exec(`INSERT INTO integrations (user_id) VALUES (?)`, user.ID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf(`store: unable to create integration row: %v`, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(`store: unable to commit transaction: %v`, err)
	}

	return &user, nil
}

// UpdateUser updates a user.
func (s *Storage) UpdateUser(user *model.User) error {
	user.ExternalFontHosts = strings.TrimSpace(user.ExternalFontHosts)

	if user.Password != "" {
		hashedPassword, err := crypto.HashPassword(user.Password)
		if err != nil {
			return err
		}

		query := `
			UPDATE users SET
				username=LOWER(?),
				password=?,
				is_admin=?,
				theme=?,
				language=?,
				timezone=?,
				entry_direction=?,
				entries_per_page=?,
				keyboard_shortcuts=?,
				show_reading_time=?,
				entry_swipe=?,
				gesture_nav=?,
				stylesheet=?,
				custom_js=?,
				external_font_hosts=?,
				google_id=?,
				openid_connect_id=?,
				display_mode=?,
				entry_order=?,
				default_reading_speed=?,
				cjk_reading_speed=?,
				default_home_page=?,
				categories_sorting_order=?,
				mark_read_on_view=?,
				mark_read_on_media_player_completion=?,
				media_playback_rate=?,
				block_filter_entry_rules=?,
				keep_filter_entry_rules=?,
				always_open_external_links=?,
				open_external_links_in_new_tab=?
			WHERE
				id=?
		`

		_, err = s.db.Exec(
			query,
			user.Username,
			hashedPassword,
			user.IsAdmin,
			user.Theme,
			user.Language,
			user.Timezone,
			user.EntryDirection,
			user.EntriesPerPage,
			user.KeyboardShortcuts,
			user.ShowReadingTime,
			user.EntrySwipe,
			user.GestureNav,
			user.Stylesheet,
			user.CustomJS,
			user.ExternalFontHosts,
			user.GoogleID,
			user.OpenIDConnectID,
			user.DisplayMode,
			user.EntryOrder,
			user.DefaultReadingSpeed,
			user.CJKReadingSpeed,
			user.DefaultHomePage,
			user.CategoriesSortingOrder,
			user.MarkReadOnView,
			user.MarkReadOnMediaPlayerCompletion,
			user.MediaPlaybackRate,
			user.BlockFilterEntryRules,
			user.KeepFilterEntryRules,
			user.AlwaysOpenExternalLinks,
			user.OpenExternalLinksInNewTab,
			user.ID,
		)
		if err != nil {
			return fmt.Errorf(`store: unable to update user: %v`, err)
		}
	} else {
		query := `
			UPDATE users SET
				username=LOWER(?),
				is_admin=?,
				theme=?,
				language=?,
				timezone=?,
				entry_direction=?,
				entries_per_page=?,
				keyboard_shortcuts=?,
				show_reading_time=?,
				entry_swipe=?,
				gesture_nav=?,
				stylesheet=?,
				custom_js=?,
				external_font_hosts=?,
				google_id=?,
				openid_connect_id=?,
				display_mode=?,
				entry_order=?,
				default_reading_speed=?,
				cjk_reading_speed=?,
				default_home_page=?,
				categories_sorting_order=?,
				mark_read_on_view=?,
				mark_read_on_media_player_completion=?,
				media_playback_rate=?,
				block_filter_entry_rules=?,
				keep_filter_entry_rules=?,
				always_open_external_links=?,
				open_external_links_in_new_tab=?
			WHERE
				id=?
		`

		_, err := s.db.Exec(
			query,
			user.Username,
			user.IsAdmin,
			user.Theme,
			user.Language,
			user.Timezone,
			user.EntryDirection,
			user.EntriesPerPage,
			user.KeyboardShortcuts,
			user.ShowReadingTime,
			user.EntrySwipe,
			user.GestureNav,
			user.Stylesheet,
			user.CustomJS,
			user.ExternalFontHosts,
			user.GoogleID,
			user.OpenIDConnectID,
			user.DisplayMode,
			user.EntryOrder,
			user.DefaultReadingSpeed,
			user.CJKReadingSpeed,
			user.DefaultHomePage,
			user.CategoriesSortingOrder,
			user.MarkReadOnView,
			user.MarkReadOnMediaPlayerCompletion,
			user.MediaPlaybackRate,
			user.BlockFilterEntryRules,
			user.KeepFilterEntryRules,
			user.AlwaysOpenExternalLinks,
			user.OpenExternalLinksInNewTab,
			user.ID,
		)

		if err != nil {
			return fmt.Errorf(`store: unable to update user: %v`, err)
		}
	}

	return nil
}

// UserLanguage returns the language of the given user.
func (s *Storage) UserLanguage(userID int64) (language string) {
	err := s.db.QueryRow(`SELECT language FROM users WHERE id = ?`, userID).Scan(&language)
	if err != nil {
		return "en_US"
	}

	return language
}

// UserByID finds a user by the ID.
func (s *Storage) UserByID(userID int64) (*model.User, error) {
	query := `
		SELECT
			id,
			username,
			is_admin,
			theme,
			language,
			timezone,
			entry_direction,
			entries_per_page,
			keyboard_shortcuts,
			show_reading_time,
			entry_swipe,
			gesture_nav,
			last_login_at,
			stylesheet,
			custom_js,
			external_font_hosts,
			google_id,
			openid_connect_id,
			display_mode,
			entry_order,
			default_reading_speed,
			cjk_reading_speed,
			default_home_page,
			categories_sorting_order,
			mark_read_on_view,
			mark_read_on_media_player_completion,
			media_playback_rate,
			block_filter_entry_rules,
			keep_filter_entry_rules,
			always_open_external_links,
			open_external_links_in_new_tab
		FROM
			users
		WHERE
			id = ?
	`
	return s.fetchUser(query, userID)
}

// UserByUsername finds a user by the username.
func (s *Storage) UserByUsername(username string) (*model.User, error) {
	query := `
		SELECT
			id,
			username,
			is_admin,
			theme,
			language,
			timezone,
			entry_direction,
			entries_per_page,
			keyboard_shortcuts,
			show_reading_time,
			entry_swipe,
			gesture_nav,
			last_login_at,
			stylesheet,
			custom_js,
			external_font_hosts,
			google_id,
			openid_connect_id,
			display_mode,
			entry_order,
			default_reading_speed,
			cjk_reading_speed,
			default_home_page,
			categories_sorting_order,
			mark_read_on_view,
			mark_read_on_media_player_completion,
			media_playback_rate,
			block_filter_entry_rules,
			keep_filter_entry_rules,
			always_open_external_links,
			open_external_links_in_new_tab
		FROM
			users
		WHERE
			username=LOWER(?)
	`
	return s.fetchUser(query, username)
}

// UserByField finds a user by a field value.
func (s *Storage) UserByField(field, value string) (*model.User, error) {
	query := `
		SELECT
			id,
			username,
			is_admin,
			theme,
			language,
			timezone,
			entry_direction,
			entries_per_page,
			keyboard_shortcuts,
			show_reading_time,
			entry_swipe,
			gesture_nav,
			last_login_at,
			stylesheet,
			custom_js,
			external_font_hosts,
			google_id,
			openid_connect_id,
			display_mode,
			entry_order,
			default_reading_speed,
			cjk_reading_speed,
			default_home_page,
			categories_sorting_order,
			mark_read_on_view,
			mark_read_on_media_player_completion,
			media_playback_rate,
			block_filter_entry_rules,
			keep_filter_entry_rules,
			always_open_external_links,
			open_external_links_in_new_tab
		FROM
			users
		WHERE
			%s=?
	`
	return s.fetchUser(fmt.Sprintf(query, field), value)
}

// AnotherUserWithFieldExists returns true if a user has the value set for the given field.
func (s *Storage) AnotherUserWithFieldExists(userID int64, field, value string) bool {
	var result bool
	s.db.QueryRow(fmt.Sprintf(`SELECT true FROM users WHERE id <> ? AND %s=? LIMIT 1`, field), userID, value).Scan(&result)
	return result
}

// UserByAPIKey returns a User from an API Key.
func (s *Storage) UserByAPIKey(token string) (*model.User, error) {
	query := `
		SELECT
			u.id,
			u.username,
			u.is_admin,
			u.theme,
			u.language,
			u.timezone,
			u.entry_direction,
			u.entries_per_page,
			u.keyboard_shortcuts,
			u.show_reading_time,
			u.entry_swipe,
			u.gesture_nav,
			u.last_login_at,
			u.stylesheet,
			u.custom_js,
			u.external_font_hosts,
			u.google_id,
			u.openid_connect_id,
			u.display_mode,
			u.entry_order,
			u.default_reading_speed,
			u.cjk_reading_speed,
			u.default_home_page,
			u.categories_sorting_order,
			u.mark_read_on_view,
			u.mark_read_on_media_player_completion,
			media_playback_rate,
			u.block_filter_entry_rules,
			u.keep_filter_entry_rules,
			u.always_open_external_links,
			u.open_external_links_in_new_tab
		FROM
			users u
		LEFT JOIN
			api_keys ON api_keys.user_id=u.id
		WHERE
			api_keys.token = ?
	`
	return s.fetchUser(query, token)
}

func (s *Storage) fetchUser(query string, args ...any) (*model.User, error) {
	var user model.User
	err := s.db.QueryRow(query, args...).Scan(
		&user.ID,
		&user.Username,
		&user.IsAdmin,
		&user.Theme,
		&user.Language,
		&user.Timezone,
		&user.EntryDirection,
		&user.EntriesPerPage,
		&user.KeyboardShortcuts,
		&user.ShowReadingTime,
		&user.EntrySwipe,
		&user.GestureNav,
		&user.LastLoginAt,
		&user.Stylesheet,
		&user.CustomJS,
		&user.ExternalFontHosts,
		&user.GoogleID,
		&user.OpenIDConnectID,
		&user.DisplayMode,
		&user.EntryOrder,
		&user.DefaultReadingSpeed,
		&user.CJKReadingSpeed,
		&user.DefaultHomePage,
		&user.CategoriesSortingOrder,
		&user.MarkReadOnView,
		&user.MarkReadOnMediaPlayerCompletion,
		&user.MediaPlaybackRate,
		&user.BlockFilterEntryRules,
		&user.KeepFilterEntryRules,
		&user.AlwaysOpenExternalLinks,
		&user.OpenExternalLinksInNewTab,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch user: %v`, err)
	}

	return &user, nil
}

// RemoveUser deletes a user.
func (s *Storage) RemoveUser(userID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf(`store: unable to start transaction: %v`, err)
	}

	if _, err := tx.Exec(`DELETE FROM users WHERE id=?`, userID); err != nil {
		tx.Rollback()
		return fmt.Errorf(`store: unable to remove user #%d: %v`, userID, err)
	}

	if _, err := tx.Exec(`DELETE FROM integrations WHERE user_id=?`, userID); err != nil {
		tx.Rollback()
		return fmt.Errorf(`store: unable to remove integration settings for user #%d: %v`, userID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(`store: unable to commit transaction: %v`, err)
	}

	return nil
}

// RemoveUserAsync deletes user data without locking the database.
func (s *Storage) RemoveUserAsync(userID int64) {
	go func() {
		if err := s.deleteUserFeeds(userID); err != nil {
			slog.Error("Unable to delete user feeds",
				slog.Int64("user_id", userID),
				slog.Any("error", err),
			)
			return
		}

		s.db.Exec(`DELETE FROM users WHERE id=?`, userID)
		s.db.Exec(`DELETE FROM integrations WHERE user_id=?`, userID)

		slog.Debug("User deleted",
			slog.Int64("user_id", userID),
			slog.Int("goroutines", runtime.NumGoroutine()),
		)
	}()
}

func (s *Storage) deleteUserFeeds(userID int64) error {
	rows, err := s.db.Query(`SELECT id FROM feeds WHERE user_id=?`, userID)
	if err != nil {
		return fmt.Errorf(`store: unable to get user feeds: %v`, err)
	}
	defer rows.Close()

	for rows.Next() {
		var feedID int64
		rows.Scan(&feedID)

		slog.Debug("Deleting feed",
			slog.Int64("user_id", userID),
			slog.Int64("feed_id", feedID),
			slog.Int("goroutines", runtime.NumGoroutine()),
		)

		if err := s.RemoveFeed(userID, feedID); err != nil {
			return err
		}
	}

	return nil
}

// Users returns all users.
func (s *Storage) Users() (model.Users, error) {
	query := `
		SELECT
			id,
			username,
			is_admin,
			theme,
			language,
			timezone,
			entry_direction,
			entries_per_page,
			keyboard_shortcuts,
			show_reading_time,
			entry_swipe,
			gesture_nav,
			last_login_at,
			stylesheet,
			custom_js,
			external_font_hosts,
			google_id,
			openid_connect_id,
			display_mode,
			entry_order,
			default_reading_speed,
			cjk_reading_speed,
			default_home_page,
			categories_sorting_order,
			mark_read_on_view,
			mark_read_on_media_player_completion,
			media_playback_rate,
			block_filter_entry_rules,
			keep_filter_entry_rules,
			always_open_external_links,
			open_external_links_in_new_tab
		FROM
			users
		ORDER BY username ASC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch users: %v`, err)
	}
	defer rows.Close()

	var users model.Users
	for rows.Next() {
		var user model.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.IsAdmin,
			&user.Theme,
			&user.Language,
			&user.Timezone,
			&user.EntryDirection,
			&user.EntriesPerPage,
			&user.KeyboardShortcuts,
			&user.ShowReadingTime,
			&user.EntrySwipe,
			&user.GestureNav,
			&user.LastLoginAt,
			&user.Stylesheet,
			&user.CustomJS,
			&user.ExternalFontHosts,
			&user.GoogleID,
			&user.OpenIDConnectID,
			&user.DisplayMode,
			&user.EntryOrder,
			&user.DefaultReadingSpeed,
			&user.CJKReadingSpeed,
			&user.DefaultHomePage,
			&user.CategoriesSortingOrder,
			&user.MarkReadOnView,
			&user.MarkReadOnMediaPlayerCompletion,
			&user.MediaPlaybackRate,
			&user.BlockFilterEntryRules,
			&user.KeepFilterEntryRules,
			&user.AlwaysOpenExternalLinks,
			&user.OpenExternalLinksInNewTab,
		)

		if err != nil {
			return nil, fmt.Errorf(`store: unable to fetch users row: %v`, err)
		}

		users = append(users, &user)
	}

	return users, nil
}

// CheckPassword validate the hashed password.
func (s *Storage) CheckPassword(username, password string) error {
	var hash string
	username = strings.ToLower(username)

	err := s.db.QueryRow("SELECT password FROM users WHERE username=?", username).Scan(&hash)
	if err == sql.ErrNoRows {
		return fmt.Errorf(`store: unable to find this user: %s`, username)
	} else if err != nil {
		return fmt.Errorf(`store: unable to fetch user: %v`, err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf(`store: invalid password for "%s" (%v)`, username, err)
	}

	return nil
}

// HasPassword returns true if the given user has a password defined.
func (s *Storage) HasPassword(userID int64) (bool, error) {
	var result bool
	query := `SELECT true FROM users WHERE id=? AND password <> '' LIMIT 1`

	err := s.db.QueryRow(query, userID).Scan(&result)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf(`store: unable to execute query: %v`, err)
	}

	if result {
		return true, nil
	}
	return false, nil
}
