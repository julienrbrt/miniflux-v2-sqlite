// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"miniflux.app/v2/internal/model"
)

// HasDuplicateFeverUsername checks if another user have the same Fever username.
func (s *Storage) HasDuplicateFeverUsername(userID int64, feverUsername string) bool {
	query := `SELECT true FROM integrations WHERE user_id != ? AND fever_username=? LIMIT 1`
	var result bool
	s.db.QueryRow(query, userID, feverUsername).Scan(&result)
	return result
}

// HasDuplicateGoogleReaderUsername checks if another user have the same Google Reader username.
func (s *Storage) HasDuplicateGoogleReaderUsername(userID int64, googleReaderUsername string) bool {
	query := `SELECT true FROM integrations WHERE user_id != ? AND googlereader_username=? LIMIT 1`
	var result bool
	s.db.QueryRow(query, userID, googleReaderUsername).Scan(&result)
	return result
}

// UserByFeverToken returns a user by using the Fever API token.
func (s *Storage) UserByFeverToken(token string) (*model.User, error) {
	query := `
		SELECT
			users.id, users.username, users.is_admin, users.timezone
		FROM
			users
		LEFT JOIN
			integrations ON integrations.user_id=users.id
		WHERE
			integrations.fever_enabled=1 AND lower(integrations.fever_token)=lower(?)
	`

	var user model.User
	err := s.db.QueryRow(query, token).Scan(&user.ID, &user.Username, &user.IsAdmin, &user.Timezone)
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("store: unable to fetch user: %v", err)
	default:
		return &user, nil
	}
}

// GoogleReaderUserCheckPassword validates the Google Reader hashed password.
func (s *Storage) GoogleReaderUserCheckPassword(username, password string) error {
	var hash string

	query := `
		SELECT
			googlereader_password
		FROM
			integrations
		WHERE
			integrations.googlereader_enabled=1 AND integrations.googlereader_username=?
	`

	err := s.db.QueryRow(query, username).Scan(&hash)
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

// GoogleReaderUserGetIntegration returns part of the Google Reader parts of the integration struct.
func (s *Storage) GoogleReaderUserGetIntegration(username string) (*model.Integration, error) {
	var integration model.Integration

	query := `
		SELECT
			user_id,
			googlereader_enabled,
			googlereader_username,
			googlereader_password
		FROM
			integrations
		WHERE
			integrations.googlereader_enabled=1 AND integrations.googlereader_username=?
	`

	err := s.db.QueryRow(query, username).Scan(&integration.UserID, &integration.GoogleReaderEnabled, &integration.GoogleReaderUsername, &integration.GoogleReaderPassword)
	if err == sql.ErrNoRows {
		return &integration, fmt.Errorf(`store: unable to find this user: %s`, username)
	} else if err != nil {
		return &integration, fmt.Errorf(`store: unable to fetch user: %v`, err)
	}

	return &integration, nil
}

// Integration returns user integration settings.
func (s *Storage) Integration(userID int64) (*model.Integration, error) {
	query := `
		SELECT
			user_id,
			pinboard_enabled,
			pinboard_token,
			pinboard_tags,
			pinboard_mark_as_unread,
			instapaper_enabled,
			instapaper_username,
			instapaper_password,
			fever_enabled,
			fever_username,
			fever_token,
			googlereader_enabled,
			googlereader_username,
			googlereader_password,
			wallabag_enabled,
			wallabag_only_url,
			wallabag_url,
			wallabag_client_id,
			wallabag_client_secret,
			wallabag_username,
			wallabag_password,
			notion_enabled,
			notion_token,
			notion_page_id,
			nunux_keeper_enabled,
			nunux_keeper_url,
			nunux_keeper_api_key,
			espial_enabled,
			espial_url,
			espial_api_key,
			espial_tags,
			readwise_enabled,
			readwise_api_key,
			telegram_bot_enabled,
			telegram_bot_token,
			telegram_bot_chat_id,
			telegram_bot_topic_id,
			telegram_bot_disable_web_page_preview,
			telegram_bot_disable_notification,
			telegram_bot_disable_buttons,
			linkace_enabled,
			linkace_url,
			linkace_api_key,
			linkace_tags,
			linkace_is_private,
			linkace_check_disabled,
			linkding_enabled,
			linkding_url,
			linkding_api_key,
			linkding_tags,
			linkding_mark_as_unread,
			linkwarden_enabled,
			linkwarden_url,
			linkwarden_api_key,
			matrix_bot_enabled,
			matrix_bot_user,
			matrix_bot_password,
			matrix_bot_url,
			matrix_bot_chat_id,
			apprise_enabled,
			apprise_url,
			apprise_services_url,
			readeck_enabled,
			readeck_url,
			readeck_api_key,
			readeck_labels,
			readeck_only_url,
			shiori_enabled,
			shiori_url,
			shiori_username,
			shiori_password,
			shaarli_enabled,
			shaarli_url,
			shaarli_api_secret,
			webhook_enabled,
			webhook_url,
			webhook_secret,
			rssbridge_enabled,
			rssbridge_url,
			omnivore_enabled,
			omnivore_api_key,
			omnivore_url,
			raindrop_enabled,
			raindrop_token,
			raindrop_collection_id,
			raindrop_tags,
			betula_enabled,
			betula_url,
			betula_token,
			ntfy_enabled,
			ntfy_topic,
			ntfy_url,
			ntfy_api_token,
			ntfy_username,
			ntfy_password,
			ntfy_icon_url,
			ntfy_internal_links,
			cubox_enabled,
			cubox_api_link,
			discord_enabled,
			discord_webhook_link,
			slack_enabled,
			slack_webhook_link,
			pushover_enabled,
			pushover_user,
			pushover_token,
			pushover_device,
			pushover_prefix,
			rssbridge_token,
			karakeep_enabled,
			karakeep_api_key,
			karakeep_url
		FROM
			integrations
		WHERE
			user_id=?
	`
	var integration model.Integration
	err := s.db.QueryRow(query, userID).Scan(
		&integration.UserID,
		&integration.PinboardEnabled,
		&integration.PinboardToken,
		&integration.PinboardTags,
		&integration.PinboardMarkAsUnread,
		&integration.InstapaperEnabled,
		&integration.InstapaperUsername,
		&integration.InstapaperPassword,
		&integration.FeverEnabled,
		&integration.FeverUsername,
		&integration.FeverToken,
		&integration.GoogleReaderEnabled,
		&integration.GoogleReaderUsername,
		&integration.GoogleReaderPassword,
		&integration.WallabagEnabled,
		&integration.WallabagOnlyURL,
		&integration.WallabagURL,
		&integration.WallabagClientID,
		&integration.WallabagClientSecret,
		&integration.WallabagUsername,
		&integration.WallabagPassword,
		&integration.NotionEnabled,
		&integration.NotionToken,
		&integration.NotionPageID,
		&integration.NunuxKeeperEnabled,
		&integration.NunuxKeeperURL,
		&integration.NunuxKeeperAPIKey,
		&integration.EspialEnabled,
		&integration.EspialURL,
		&integration.EspialAPIKey,
		&integration.EspialTags,
		&integration.ReadwiseEnabled,
		&integration.ReadwiseAPIKey,
		&integration.TelegramBotEnabled,
		&integration.TelegramBotToken,
		&integration.TelegramBotChatID,
		&integration.TelegramBotTopicID,
		&integration.TelegramBotDisableWebPagePreview,
		&integration.TelegramBotDisableNotification,
		&integration.TelegramBotDisableButtons,
		&integration.LinkAceEnabled,
		&integration.LinkAceURL,
		&integration.LinkAceAPIKey,
		&integration.LinkAceTags,
		&integration.LinkAcePrivate,
		&integration.LinkAceCheckDisabled,
		&integration.LinkdingEnabled,
		&integration.LinkdingURL,
		&integration.LinkdingAPIKey,
		&integration.LinkdingTags,
		&integration.LinkdingMarkAsUnread,
		&integration.LinkwardenEnabled,
		&integration.LinkwardenURL,
		&integration.LinkwardenAPIKey,
		&integration.MatrixBotEnabled,
		&integration.MatrixBotUser,
		&integration.MatrixBotPassword,
		&integration.MatrixBotURL,
		&integration.MatrixBotChatID,
		&integration.AppriseEnabled,
		&integration.AppriseURL,
		&integration.AppriseServicesURL,
		&integration.ReadeckEnabled,
		&integration.ReadeckURL,
		&integration.ReadeckAPIKey,
		&integration.ReadeckLabels,
		&integration.ReadeckOnlyURL,
		&integration.ShioriEnabled,
		&integration.ShioriURL,
		&integration.ShioriUsername,
		&integration.ShioriPassword,
		&integration.ShaarliEnabled,
		&integration.ShaarliURL,
		&integration.ShaarliAPISecret,
		&integration.WebhookEnabled,
		&integration.WebhookURL,
		&integration.WebhookSecret,
		&integration.RSSBridgeEnabled,
		&integration.RSSBridgeURL,
		&integration.OmnivoreEnabled,
		&integration.OmnivoreAPIKey,
		&integration.OmnivoreURL,
		&integration.RaindropEnabled,
		&integration.RaindropToken,
		&integration.RaindropCollectionID,
		&integration.RaindropTags,
		&integration.BetulaEnabled,
		&integration.BetulaURL,
		&integration.BetulaToken,
		&integration.NtfyEnabled,
		&integration.NtfyTopic,
		&integration.NtfyURL,
		&integration.NtfyAPIToken,
		&integration.NtfyUsername,
		&integration.NtfyPassword,
		&integration.NtfyIconURL,
		&integration.NtfyInternalLinks,
		&integration.CuboxEnabled,
		&integration.CuboxAPILink,
		&integration.DiscordEnabled,
		&integration.DiscordWebhookLink,
		&integration.SlackEnabled,
		&integration.SlackWebhookLink,
		&integration.PushoverEnabled,
		&integration.PushoverUser,
		&integration.PushoverToken,
		&integration.PushoverDevice,
		&integration.PushoverPrefix,
		&integration.RSSBridgeToken,
		&integration.KarakeepEnabled,
		&integration.KarakeepAPIKey,
		&integration.KarakeepURL,
	)
	switch {
	case err == sql.ErrNoRows:
		return &integration, nil
	case err != nil:
		return &integration, fmt.Errorf(`store: unable to fetch integration row: %v`, err)
	default:
		return &integration, nil
	}
}

// UpdateIntegration saves user integration settings.
func (s *Storage) UpdateIntegration(integration *model.Integration) error {
	query := `
		UPDATE
			integrations
		SET
			pinboard_enabled=?,
			pinboard_token=?,
			pinboard_tags=?,
			pinboard_mark_as_unread=?,
			instapaper_enabled=?,
			instapaper_username=?,
			instapaper_password=?,
			fever_enabled=?,
			fever_username=?,
			fever_token=?,
			wallabag_enabled=?,
			wallabag_only_url=?,
			wallabag_url=?,
			wallabag_client_id=?,
			wallabag_client_secret=?,
			wallabag_username=?,
			wallabag_password=?,
			nunux_keeper_enabled=?,
			nunux_keeper_url=?,
			nunux_keeper_api_key=?,
			googlereader_enabled=?,
			googlereader_username=?,
			googlereader_password=?,
			telegram_bot_enabled=?,
			telegram_bot_token=?,
			telegram_bot_chat_id=?,
			telegram_bot_topic_id=?,
			telegram_bot_disable_web_page_preview=?,
			telegram_bot_disable_notification=?,
			telegram_bot_disable_buttons=?,
			espial_enabled=?,
			espial_url=?,
			espial_api_key=?,
			espial_tags=?,
			linkace_enabled=?,
			linkace_url=?,
			linkace_api_key=?,
			linkace_tags=?,
			linkace_is_private=?,
			linkace_check_disabled=?,
			linkding_enabled=?,
			linkding_url=?,
			linkding_api_key=?,
			linkding_tags=?,
			linkding_mark_as_unread=?,
			matrix_bot_enabled=?,
			matrix_bot_user=?,
			matrix_bot_password=?,
			matrix_bot_url=?,
			matrix_bot_chat_id=?,
			notion_enabled=?,
			notion_token=?,
			notion_page_id=?,
			readwise_enabled=?,
			readwise_api_key=?,
			apprise_enabled=?,
			apprise_url=?,
			apprise_services_url=?,
			readeck_enabled=?,
			readeck_url=?,
			readeck_api_key=?,
			readeck_labels=?,
			readeck_only_url=?,
			shiori_enabled=?,
			shiori_url=?,
			shiori_username=?,
			shiori_password=?,
			shaarli_enabled=?,
			shaarli_url=?,
			shaarli_api_secret=?,
			webhook_enabled=?,
			webhook_url=?,
			webhook_secret=?,
			rssbridge_enabled=?,
			rssbridge_url=?,
			omnivore_enabled=?,
			omnivore_api_key=?,
			omnivore_url=?,
			linkwarden_enabled=?,
			linkwarden_url=?,
			linkwarden_api_key=?,
			raindrop_enabled=?,
			raindrop_token=?,
			raindrop_collection_id=?,
			raindrop_tags=?,
			betula_enabled=?,
			betula_url=?,
			betula_token=?,
			ntfy_enabled=?,
			ntfy_topic=?,
			ntfy_url=?,
			ntfy_api_token=?,
			ntfy_username=?,
			ntfy_password=?,
			ntfy_icon_url=?,
			ntfy_internal_links=?,
			cubox_enabled=?,
			cubox_api_link=?,
			discord_enabled=?,
			discord_webhook_link=?,
			slack_enabled=?,
			slack_webhook_link=?,
			pushover_enabled=?,
			pushover_user=?,
			pushover_token=?,
			pushover_device=?,
			pushover_prefix=?,
			rssbridge_token=?,
			karakeep_enabled=?,
			karakeep_api_key=?,
			karakeep_url=?
		WHERE
			user_id=?
	`
	_, err := s.db.Exec(
		query,
		integration.PinboardEnabled,
		integration.PinboardToken,
		integration.PinboardTags,
		integration.PinboardMarkAsUnread,
		integration.InstapaperEnabled,
		integration.InstapaperUsername,
		integration.InstapaperPassword,
		integration.FeverEnabled,
		integration.FeverUsername,
		integration.FeverToken,
		integration.WallabagEnabled,
		integration.WallabagOnlyURL,
		integration.WallabagURL,
		integration.WallabagClientID,
		integration.WallabagClientSecret,
		integration.WallabagUsername,
		integration.WallabagPassword,
		integration.NunuxKeeperEnabled,
		integration.NunuxKeeperURL,
		integration.NunuxKeeperAPIKey,
		integration.GoogleReaderEnabled,
		integration.GoogleReaderUsername,
		integration.GoogleReaderPassword,
		integration.TelegramBotEnabled,
		integration.TelegramBotToken,
		integration.TelegramBotChatID,
		integration.TelegramBotTopicID,
		integration.TelegramBotDisableWebPagePreview,
		integration.TelegramBotDisableNotification,
		integration.TelegramBotDisableButtons,
		integration.EspialEnabled,
		integration.EspialURL,
		integration.EspialAPIKey,
		integration.EspialTags,
		integration.LinkAceEnabled,
		integration.LinkAceURL,
		integration.LinkAceAPIKey,
		integration.LinkAceTags,
		integration.LinkAcePrivate,
		integration.LinkAceCheckDisabled,
		integration.LinkdingEnabled,
		integration.LinkdingURL,
		integration.LinkdingAPIKey,
		integration.LinkdingTags,
		integration.LinkdingMarkAsUnread,
		integration.MatrixBotEnabled,
		integration.MatrixBotUser,
		integration.MatrixBotPassword,
		integration.MatrixBotURL,
		integration.MatrixBotChatID,
		integration.NotionEnabled,
		integration.NotionToken,
		integration.NotionPageID,
		integration.ReadwiseEnabled,
		integration.ReadwiseAPIKey,
		integration.AppriseEnabled,
		integration.AppriseURL,
		integration.AppriseServicesURL,
		integration.ReadeckEnabled,
		integration.ReadeckURL,
		integration.ReadeckAPIKey,
		integration.ReadeckLabels,
		integration.ReadeckOnlyURL,
		integration.ShioriEnabled,
		integration.ShioriURL,
		integration.ShioriUsername,
		integration.ShioriPassword,
		integration.ShaarliEnabled,
		integration.ShaarliURL,
		integration.ShaarliAPISecret,
		integration.WebhookEnabled,
		integration.WebhookURL,
		integration.WebhookSecret,
		integration.RSSBridgeEnabled,
		integration.RSSBridgeURL,
		integration.OmnivoreEnabled,
		integration.OmnivoreAPIKey,
		integration.OmnivoreURL,
		integration.LinkwardenEnabled,
		integration.LinkwardenURL,
		integration.LinkwardenAPIKey,
		integration.RaindropEnabled,
		integration.RaindropToken,
		integration.RaindropCollectionID,
		integration.RaindropTags,
		integration.BetulaEnabled,
		integration.BetulaURL,
		integration.BetulaToken,
		integration.NtfyEnabled,
		integration.NtfyTopic,
		integration.NtfyURL,
		integration.NtfyAPIToken,
		integration.NtfyUsername,
		integration.NtfyPassword,
		integration.NtfyIconURL,
		integration.NtfyInternalLinks,
		integration.CuboxEnabled,
		integration.CuboxAPILink,
		integration.DiscordEnabled,
		integration.DiscordWebhookLink,
		integration.SlackEnabled,
		integration.SlackWebhookLink,
		integration.PushoverEnabled,
		integration.PushoverUser,
		integration.PushoverToken,
		integration.PushoverDevice,
		integration.PushoverPrefix,
		integration.RSSBridgeToken,
		integration.KarakeepEnabled,
		integration.KarakeepAPIKey,
		integration.KarakeepURL,
		integration.UserID,
	)

	if err != nil {
		return fmt.Errorf(`store: unable to update integration record: %v`, err)
	}

	return nil
}

// HasSaveEntry returns true if the given user can save articles to third-parties.
func (s *Storage) HasSaveEntry(userID int64) (result bool) {
	query := `
		SELECT
			true
		FROM
			integrations
		WHERE
			user_id=?
		AND
			(
				pinboard_enabled=1 OR
				instapaper_enabled=1 OR
				wallabag_enabled=1 OR
				notion_enabled=1 OR
				nunux_keeper_enabled=1 OR
				espial_enabled=1 OR
				readwise_enabled=1 OR
				linkace_enabled=1 OR
				linkding_enabled=1 OR
				linkwarden_enabled=1 OR
				apprise_enabled=1 OR
				shiori_enabled=1 OR
				readeck_enabled=1 OR
				shaarli_enabled=1 OR
				webhook_enabled=1 OR
				omnivore_enabled=1 OR
				karakeep_enabled=1 OR
				raindrop_enabled=1 OR
				betula_enabled=1 OR
				cubox_enabled=1 OR
				discord_enabled=1 OR
				slack_enabled=1
			)
	`
	if err := s.db.QueryRow(query, userID).Scan(&result); err != nil {
		result = false
	}

	return result
}
