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

func (h *handler) createSharedEntry(w http.ResponseWriter, r *http.Request) {
	entryID := request.RouteInt64Param(r, "entryID")
	shareCode, err := h.store.EntryShareCode(request.UserID(r), entryID)
	if err != nil {
		response.HTMLServerError(w, r, err)
		return
	}

	response.HTMLRedirect(w, r, h.routePath("/share/%s", shareCode))
}

func (h *handler) unshareEntry(w http.ResponseWriter, r *http.Request) {
	entryID := request.RouteInt64Param(r, "entryID")
	if err := h.store.UnshareEntry(request.UserID(r), entryID); err != nil {
		response.HTMLServerError(w, r, err)
		return
	}

	response.HTMLRedirect(w, r, h.routePath("/shares"))
}

func (h *handler) sharedEntry(w http.ResponseWriter, r *http.Request) {
	shareCode := request.RouteStringParam(r, "shareCode")
	if shareCode == "" {
		response.HTMLNotFound(w, r)
		return
	}

	// Look the entry up before answering conditional requests, so an expired
	// link cannot keep revalidating as "not modified".
	query := h.store.NewAnonymousQueryBuilder().WithShareCode(shareCode)
	cacheDuration := 72 * time.Hour
	if expiry := config.Opts.ShareExpiryInterval(); expiry >= 0 {
		query = query.SharedAfter(time.Now().Add(-expiry))
		// Let browsers revalidate soon, so they notice the expiry.
		cacheDuration = time.Hour
	}

	entry, err := query.GetEntry()
	if err != nil || entry == nil {
		response.HTMLNotFound(w, r)
		return
	}

	response.NewBuilder(w, r).WithCaching(shareCode, cacheDuration, func(b *response.Builder) {
		view := view.New(h.tpl, r)
		view.Set("entry", entry)

		b.WithHeader("Content-Type", "text/html; charset=utf-8")
		b.WithBodyAsBytes(view.Render("entry"))
		b.Write()
	})
}
