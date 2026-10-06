// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/ui/view"
)

// shareExpiryDate is when a share link expires; Expired stays true until the
// cleanup job removes the share code.
type shareExpiryDate struct {
	At      time.Time
	Expired bool
}

func (h *handler) sharedEntries(w http.ResponseWriter, r *http.Request) {
	user, err := h.store.UserByID(request.UserID(r))
	if err != nil {
		response.HTMLServerError(w, r, err)
		return
	}

	offset := request.QueryIntParam(r, "offset", 0)

	entries, count, err := h.store.NewEntryQueryBuilder(user.ID).
		WithShareCodeNotEmpty().
		WithSorting(user.EntryOrder, user.EntryDirection).
		WithSorting("id", user.EntryDirection).
		WithOffset(offset).
		WithLimit(user.EntriesPerPage).
		WithoutContent().
		GetEntriesWithCount()
	if err != nil {
		response.HTMLServerError(w, r, err)
		return
	}

	// When each share link expires (SHARE_EXPIRY_DAYS), keyed by entry ID.
	shareExpiry := make(map[int64]*shareExpiryDate)
	if interval := config.Opts.ShareExpiryInterval(); interval >= 0 && len(entries) > 0 {
		entryIDs := make([]int64, 0, len(entries))
		for _, entry := range entries {
			entryIDs = append(entryIDs, entry.ID)
		}
		dates, err := h.store.ShareExpiryDates(user.ID, entryIDs, interval)
		if err != nil {
			response.HTMLServerError(w, r, err)
			return
		}
		now := time.Now()
		for entryID, expiresAt := range dates {
			shareExpiry[entryID] = &shareExpiryDate{At: expiresAt, Expired: !now.Before(expiresAt)}
		}
	}

	view := view.New(h.tpl, r)
	view.Set("entries", entries)
	view.Set("shareExpiry", shareExpiry)
	view.Set("total", count)
	view.Set("pagination", getPagination(h.routePath("/shares"), count, offset, user.EntriesPerPage))
	view.Set("menu", "history")
	view.Set("user", user)
	navMetadata, _ := h.store.GetNavMetadata(user.ID)
	view.Set("countUnread", navMetadata.CountUnread)
	view.Set("countErrorFeeds", navMetadata.CountErrorFeeds)
	view.Set("hasSaveEntry", navMetadata.HasSaveEntry)

	response.HTML(w, r, view.Render("shared_entries"))
}
