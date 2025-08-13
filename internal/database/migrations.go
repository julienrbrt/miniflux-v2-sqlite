// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package database // import "miniflux.app/v2/internal/database"

import (
	"database/sql"
	"encoding/json"

	"miniflux.app/v2/internal/crypto"
)

var schemaVersion = len(migrations)

// Order is important. Add new migrations at the end of the list.
var migrations = []func(tx *sql.Tx) error{
	func(tx *sql.Tx) (err error) {
		sql := `
			CREATE TABLE schema_version (
				version TEXT NOT NULL
			);

			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT NOT NULL UNIQUE,
				password TEXT,
				is_admin INTEGER DEFAULT 0,
				language TEXT DEFAULT 'en_US',
				timezone TEXT DEFAULT 'UTC',
				theme TEXT DEFAULT 'default',
				last_login_at DATETIME
			);

			CREATE TABLE sessions (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				token TEXT NOT NULL UNIQUE,
				created_at DATETIME DEFAULT (datetime('now')),
				user_agent TEXT,
				ip TEXT,
				UNIQUE (user_id, token),
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			);

			CREATE TABLE categories (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				title TEXT NOT NULL,
				UNIQUE (user_id, title),
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			);

			CREATE TABLE feeds (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				category_id INTEGER NOT NULL,
				title TEXT NOT NULL,
				feed_url TEXT NOT NULL,
				site_url TEXT NOT NULL,
				checked_at DATETIME DEFAULT (datetime('now')),
				etag_header TEXT DEFAULT '',
				last_modified_header TEXT DEFAULT '',
				parsing_error_msg TEXT DEFAULT '',
				parsing_error_count INTEGER DEFAULT 0,
				UNIQUE (user_id, feed_url),
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
			);

			CREATE TABLE entries (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				feed_id INTEGER NOT NULL,
				hash TEXT NOT NULL,
				published_at DATETIME NOT NULL,
				title TEXT NOT NULL,
				url TEXT NOT NULL,
				author TEXT,
				content TEXT,
				status TEXT DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'removed')),
				UNIQUE (feed_id, hash),
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE
			);

			CREATE INDEX entries_feed_idx ON entries(feed_id);

			CREATE TABLE enclosures (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				entry_id INTEGER NOT NULL,
				url TEXT NOT NULL,
				size INTEGER DEFAULT 0,
				mime_type TEXT DEFAULT '',
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY (entry_id) REFERENCES entries(id) ON DELETE CASCADE
			);

			CREATE TABLE icons (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				hash TEXT NOT NULL UNIQUE,
				mime_type TEXT NOT NULL,
				content BLOB NOT NULL
			);

			CREATE TABLE feed_icons (
				feed_id INTEGER NOT NULL,
				icon_id INTEGER NOT NULL,
				PRIMARY KEY(feed_id, icon_id),
				FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
				FOREIGN KEY (icon_id) REFERENCES icons(id) ON DELETE CASCADE
			);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE users ADD COLUMN extra TEXT DEFAULT '{}';
			CREATE INDEX users_extra_idx ON users(extra);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			CREATE TABLE tokens (
				id TEXT NOT NULL,
				value TEXT NOT NULL,
				created_at DATETIME NOT NULL DEFAULT (datetime('now')),
				PRIMARY KEY(id, value)
			);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE users ADD COLUMN entry_direction TEXT DEFAULT 'asc' CHECK (entry_direction IN ('asc', 'desc'));
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			CREATE TABLE integrations (
				user_id INTEGER NOT NULL,
				pinboard_enabled INTEGER DEFAULT 0,
				pinboard_token TEXT DEFAULT '',
				pinboard_tags TEXT DEFAULT 'miniflux',
				pinboard_mark_as_unread INTEGER DEFAULT 0,
				instapaper_enabled INTEGER DEFAULT 0,
				instapaper_username TEXT DEFAULT '',
				instapaper_password TEXT DEFAULT '',
				fever_enabled INTEGER DEFAULT 0,
				fever_username TEXT DEFAULT '',
				fever_password TEXT DEFAULT '',
				fever_token TEXT DEFAULT '',
				PRIMARY KEY(user_id)
			);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN scraper_rules TEXT DEFAULT ''`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN rewrite_rules TEXT DEFAULT ''`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN crawler INTEGER DEFAULT 0`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE sessions RENAME TO user_sessions`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			DROP TABLE tokens;

			CREATE TABLE sessions (
				id TEXT NOT NULL,
				data TEXT NOT NULL,
				created_at DATETIME NOT NULL DEFAULT (datetime('now')),
				PRIMARY KEY(id)
			);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN wallabag_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN wallabag_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN wallabag_client_id TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN wallabag_client_secret TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN wallabag_username TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN wallabag_password TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE entries ADD COLUMN starred INTEGER DEFAULT 0`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			CREATE INDEX entries_user_status_idx ON entries(user_id, status);
			CREATE INDEX feeds_user_category_idx ON feeds(user_id, category_id);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN nunux_keeper_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN nunux_keeper_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN nunux_keeper_api_key TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE enclosures ADD COLUMN comments_url TEXT DEFAULT ''`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE entries ADD COLUMN comments_url TEXT DEFAULT ''`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Skip pocket integration - not needed for SQLite version
		return nil
	},
	func(tx *sql.Tx) (err error) {
		// Skip inet conversion - use TEXT for IP addresses
		return nil
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE feeds ADD COLUMN username TEXT DEFAULT '';
			ALTER TABLE feeds ADD COLUMN password TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Skip tsvector - SQLite doesn't have built-in full-text search in this way
		// We'll implement search differently if needed
		return nil
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN user_agent TEXT DEFAULT ''`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Skip tsvector update
		return nil
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN keyboard_shortcuts INTEGER DEFAULT 1`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN disabled INTEGER DEFAULT 0;`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			UPDATE users SET theme='light_serif' WHERE theme='default';
			UPDATE users SET theme='light_sans_serif' WHERE theme='sansserif';
			UPDATE users SET theme='dark_serif' WHERE theme='black';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE entries ADD COLUMN changed_at DATETIME;
			UPDATE entries SET changed_at = published_at;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			CREATE TABLE api_keys (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				token TEXT NOT NULL UNIQUE,
				description TEXT NOT NULL,
				last_used_at DATETIME,
				created_at DATETIME DEFAULT (datetime('now')),
				UNIQUE (user_id, description)
			);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE entries ADD COLUMN share_code TEXT NOT NULL DEFAULT '';
			CREATE UNIQUE INDEX entries_share_code_idx ON entries(share_code) WHERE share_code <> '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Skip MD5 index - SQLite doesn't have MD5 function built-in
		return nil
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE feeds ADD COLUMN next_check_at DATETIME DEFAULT (datetime('now'));
			CREATE INDEX entries_user_feed_idx ON entries (user_id, feed_id);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN ignore_http_cache INTEGER DEFAULT 0`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN entries_per_page INTEGER DEFAULT 100`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN show_reading_time INTEGER DEFAULT 1`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `CREATE INDEX entries_id_user_status_idx ON entries(id, user_id, status)`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN fetch_via_proxy INTEGER DEFAULT 0`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `CREATE INDEX entries_feed_id_status_hash_idx ON entries(feed_id, status, hash)`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `CREATE INDEX entries_user_id_status_starred_idx ON entries (user_id, status, starred)`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN entry_swipe INTEGER DEFAULT 1`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE integrations DROP COLUMN fever_password`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE feeds ADD COLUMN blocklist_rules TEXT NOT NULL DEFAULT '';
			ALTER TABLE feeds ADD COLUMN keeplist_rules TEXT NOT NULL DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE entries ADD COLUMN reading_time INTEGER NOT NULL DEFAULT 0`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE entries ADD COLUMN created_at DATETIME NOT NULL DEFAULT (datetime('now'));
			UPDATE entries SET created_at = published_at;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Handle the extra column migration differently for SQLite
		// First, get all users and their extra data
		rows, err := tx.Query(`SELECT id, extra FROM users`)
		if err != nil {
			return err
		}
		defer rows.Close()

		type userUpdate struct {
			id              int64
			stylesheet      string
			googleID        string
			openIDConnectID string
		}

		var updates []userUpdate

		for rows.Next() {
			var userID int64
			var extraJSON string
			if err := rows.Scan(&userID, &extraJSON); err != nil {
				return err
			}

			var extra map[string]interface{}
			if err := json.Unmarshal([]byte(extraJSON), &extra); err != nil {
				// If JSON is invalid, use empty values
				extra = make(map[string]interface{})
			}

			stylesheet := ""
			googleID := ""
			oidcID := ""

			if val, ok := extra["custom_css"]; ok {
				if str, ok := val.(string); ok {
					stylesheet = str
				}
			}
			if val, ok := extra["google_id"]; ok {
				if str, ok := val.(string); ok {
					googleID = str
				}
			}
			if val, ok := extra["oidc_id"]; ok {
				if str, ok := val.(string); ok {
					oidcID = str
				}
			}

			updates = append(updates, userUpdate{
				id:              userID,
				stylesheet:      stylesheet,
				googleID:        googleID,
				openIDConnectID: oidcID,
			})
		}

		// Add the new columns
		_, err = tx.Exec(`
			ALTER TABLE users ADD COLUMN stylesheet TEXT NOT NULL DEFAULT '';
			ALTER TABLE users ADD COLUMN google_id TEXT NOT NULL DEFAULT '';
			ALTER TABLE users ADD COLUMN openid_connect_id TEXT NOT NULL DEFAULT '';
		`)
		if err != nil {
			return err
		}

		// Update each user with their extracted data
		for _, update := range updates {
			_, err := tx.Exec(
				`UPDATE users SET stylesheet=?, google_id=?, openid_connect_id=? WHERE id=?`,
				update.stylesheet, update.googleID, update.openIDConnectID, update.id)
			if err != nil {
				return err
			}
		}

		return nil
	},
	func(tx *sql.Tx) (err error) {
		// Drop the extra column and create unique indexes
		sql := `
			CREATE UNIQUE INDEX users_google_id_idx ON users(google_id) WHERE google_id <> '';
			CREATE UNIQUE INDEX users_openid_connect_id_idx ON users(openid_connect_id) WHERE openid_connect_id <> '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			CREATE INDEX entries_user_status_feed_idx ON entries(user_id, status, feed_id);
			CREATE INDEX entries_user_status_changed_idx ON entries(user_id, status, changed_at);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			CREATE TABLE acme_cache (
				key TEXT PRIMARY KEY,
				data BLOB NOT NULL,
				updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
			);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE feeds ADD COLUMN allow_self_signed_certificates INTEGER NOT NULL DEFAULT 0
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE users ADD COLUMN display_mode TEXT DEFAULT 'standalone' CHECK (display_mode IN ('fullscreen', 'standalone', 'minimal-ui', 'browser'));
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN cookie TEXT DEFAULT ''`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE categories ADD COLUMN hide_globally INTEGER NOT NULL DEFAULT 0
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE feeds ADD COLUMN hide_globally INTEGER NOT NULL DEFAULT 0
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN telegram_bot_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN telegram_bot_token TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN telegram_bot_chat_id TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE users ADD COLUMN entry_order TEXT DEFAULT 'published_at' CHECK (entry_order IN ('published_at', 'created_at'));
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN googlereader_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN googlereader_username TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN googlereader_password TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN espial_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN espial_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN espial_api_key TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN espial_tags TEXT DEFAULT 'miniflux';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN linkding_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN linkding_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN linkding_api_key TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE feeds ADD COLUMN url_rewrite_rules TEXT NOT NULL DEFAULT ''
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE users ADD COLUMN default_reading_speed INTEGER DEFAULT 265;
			ALTER TABLE users ADD COLUMN cjk_reading_speed INTEGER DEFAULT 500;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE users ADD COLUMN default_home_page TEXT DEFAULT 'unread';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN wallabag_only_url INTEGER DEFAULT 0;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE users ADD COLUMN categories_sorting_order TEXT NOT NULL DEFAULT 'unread_count';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN matrix_bot_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN matrix_bot_user TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN matrix_bot_password TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN matrix_bot_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN matrix_bot_chat_id TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN gesture_nav TEXT DEFAULT 'tap' CHECK (gesture_nav IN ('tap', 'none'))`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE entries ADD COLUMN tags TEXT DEFAULT '[]';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Convert double_tap to gesture_nav - this step is already handled above, skip
		return nil
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN linkding_tags TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE feeds ADD COLUMN no_media_player INTEGER DEFAULT 0;
			ALTER TABLE enclosures ADD COLUMN media_progression INTEGER DEFAULT 0;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN linkding_mark_as_unread INTEGER DEFAULT 0;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Handle enclosure duplicates differently for SQLite
		// Delete duplicates first
		sql := `
			DELETE FROM enclosures
			WHERE rowid NOT IN (
				SELECT MIN(rowid)
				FROM enclosures
				GROUP BY user_id, entry_id, url
			);
		`
		_, err = tx.Exec(sql)
		if err != nil {
			return err
		}

		// Create unique index
		_, err = tx.Exec(`CREATE UNIQUE INDEX enclosures_user_entry_url_unique_idx ON enclosures(user_id, entry_id, url)`)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN mark_read_on_view INTEGER DEFAULT 1`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN notion_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN notion_token TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN notion_page_id TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN readwise_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN readwise_api_key TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN apprise_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN apprise_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN apprise_services_url TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN shiori_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN shiori_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN shiori_username TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN shiori_password TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN shaarli_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN shaarli_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN shaarli_api_secret TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE feeds ADD COLUMN apprise_service_urls TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN webhook_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN webhook_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN webhook_secret TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN telegram_bot_topic_id INTEGER;
			ALTER TABLE integrations ADD COLUMN telegram_bot_disable_web_page_preview INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN telegram_bot_disable_notification INTEGER DEFAULT 0;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN telegram_bot_disable_buttons INTEGER DEFAULT 0;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			CREATE INDEX enclosures_entry_id_idx ON enclosures(entry_id);
			CREATE INDEX entries_user_status_published_idx ON entries(user_id, status, published_at);
			CREATE INDEX entries_user_status_created_idx ON entries(user_id, status, created_at);
			CREATE INDEX feeds_feed_id_hide_globally_idx ON feeds(id, hide_globally);
			CREATE INDEX entries_user_status_changed_published_idx ON entries(user_id, status, changed_at, published_at);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN rssbridge_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN rssbridge_url TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			CREATE TABLE webauthn_credentials (
				handle BLOB PRIMARY KEY,
				cred_id BLOB UNIQUE NOT NULL,
				user_id INTEGER REFERENCES users(id) ON DELETE CASCADE NOT NULL,
				public_key BLOB NOT NULL,
				attestation_type TEXT NOT NULL,
				aaguid BLOB,
				sign_count INTEGER,
				clone_warning INTEGER,
				name TEXT,
				added_on DATETIME DEFAULT (datetime('now')),
				last_seen_on DATETIME DEFAULT (datetime('now'))
			);
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN omnivore_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN omnivore_api_key TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN omnivore_url TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN linkace_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN linkace_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN linkace_api_key TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN linkace_tags TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN linkace_is_private INTEGER DEFAULT 1;
			ALTER TABLE integrations ADD COLUMN linkace_check_disabled INTEGER DEFAULT 1;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN linkwarden_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN linkwarden_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN linkwarden_api_key TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN readeck_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN readeck_only_url INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN readeck_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN readeck_api_key TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN readeck_labels TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN disable_http2 INTEGER DEFAULT 0`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN media_playback_rate REAL DEFAULT 1;`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Remove empty tags from JSON arrays
		sql := `UPDATE entries SET tags = '[]' WHERE tags = '[""]' OR tags = '' OR tags IS NULL;`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Skip dropping entries_feed_url_idx as it may not exist in SQLite version
		return nil
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN raindrop_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN raindrop_token TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN raindrop_collection_id TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN raindrop_tags TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN description TEXT DEFAULT ''`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE users ADD COLUMN block_filter_entry_rules TEXT NOT NULL DEFAULT '';
			ALTER TABLE users ADD COLUMN keep_filter_entry_rules TEXT NOT NULL DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN betula_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN betula_token TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN betula_enabled INTEGER DEFAULT 0;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN ntfy_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN ntfy_url TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN ntfy_topic TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN ntfy_api_token TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN ntfy_username TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN ntfy_password TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN ntfy_icon_url TEXT DEFAULT '';

			ALTER TABLE feeds ADD COLUMN ntfy_enabled INTEGER DEFAULT 0;
			ALTER TABLE feeds ADD COLUMN ntfy_priority INTEGER DEFAULT 3;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN mark_read_on_media_player_completion INTEGER DEFAULT 0;`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN custom_js TEXT NOT NULL DEFAULT '';`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN external_font_hosts TEXT NOT NULL DEFAULT '';`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN cubox_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN cubox_api_link TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN discord_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN discord_webhook_link TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE integrations ADD COLUMN ntfy_internal_links INTEGER DEFAULT 0;`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN slack_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN slack_webhook_link TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN webhook_url TEXT DEFAULT '';`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN pushover_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN pushover_user TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN pushover_token TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN pushover_device TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN pushover_prefix TEXT DEFAULT '';

			ALTER TABLE feeds ADD COLUMN pushover_enabled INTEGER DEFAULT 0;
			ALTER TABLE feeds ADD COLUMN pushover_priority INTEGER DEFAULT 0;
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE feeds ADD COLUMN ntfy_topic TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE icons ADD COLUMN external_id TEXT DEFAULT '';
			CREATE UNIQUE INDEX icons_external_id_idx ON icons(external_id) WHERE external_id <> '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Generate external IDs for existing icons
		rows, err := tx.Query(`SELECT id FROM icons WHERE external_id = ''`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				return err
			}

			_, err = tx.Exec(
				`UPDATE icons SET external_id = ? WHERE id = ?`,
				crypto.GenerateRandomStringHex(20), id)
			if err != nil {
				return err
			}
		}
		return nil
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE feeds ADD COLUMN proxy_url TEXT DEFAULT ''`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN rssbridge_token TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN always_open_external_links INTEGER DEFAULT 0`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE integrations ADD COLUMN karakeep_enabled INTEGER DEFAULT 0;
			ALTER TABLE integrations ADD COLUMN karakeep_api_key TEXT DEFAULT '';
			ALTER TABLE integrations ADD COLUMN karakeep_url TEXT DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		sql := `ALTER TABLE users ADD COLUMN open_external_links_in_new_tab INTEGER DEFAULT 1`
		_, err = tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) (err error) {
		// Drop the extra column - this is a no-op for SQLite since we already handled it
		return nil
	},
	func(tx *sql.Tx) (err error) {
		sql := `
			ALTER TABLE feeds ADD COLUMN block_filter_entry_rules TEXT NOT NULL DEFAULT '';
			ALTER TABLE feeds ADD COLUMN keep_filter_entry_rules TEXT NOT NULL DEFAULT '';
		`
		_, err = tx.Exec(sql)
		return err
	},
}
