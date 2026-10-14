/*
 * Draws the sighting markers on a Leaflet map.
 *
 * Every map is progressive enhancement: the same records are always in
 * the page as text, so losing the CDN or JavaScript costs the reader
 * nothing but the map itself. Each theme styles its own map through
 * data attributes rather than forking this file.
 */
(function () {
	"use strict";

	// OpenStreetMap's own tiles, which need no API key: CARTO's have
	// wanted one since August 2026. A theme can name its own basemap in
	// data-tiles and data-attribution; the dark ones filter these in CSS.
	var TILES = "https://tile.openstreetmap.org/{z}/{x}/{y}.png";
	var TILES_CREDIT = "© OpenStreetMap contributors";

	var REDUCED_MOTION = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

	function readPoints(id) {
		var el = document.getElementById(id);
		if (!el) {
			return [];
		}
		try {
			return JSON.parse(el.textContent) || [];
		} catch (err) {
			return [];
		}
	}

	function popup(point) {
		var parts = ["<strong>" + escapeHTML(point.title) + "</strong>"];
		parts.push(escapeHTML(point.date) + " &middot; " + escapeHTML(point.place));
		if (point.count) {
			parts.push(escapeHTML(point.count));
		}
		parts.push('<a href="#' + escapeHTML(point.anchor) + '">Jump to record</a>');
		return '<div class="map-popup">' + parts.join("<br>") + "</div>";
	}

	function escapeHTML(value) {
		return String(value == null ? "" : value).replace(/[&<>"']/g, function (c) {
			return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
		});
	}

	function init(el) {
		var points = readPoints(el.dataset.points);
		if (!points.length || typeof window.L === "undefined") {
			el.classList.add("is-unavailable");
			return;
		}

		var map = L.map(el, {
			scrollWheelZoom: false,
			zoomAnimation: !REDUCED_MOTION,
			fadeAnimation: !REDUCED_MOTION,
			worldCopyJump: true,
		});

		L.tileLayer(el.dataset.tiles || TILES, {
			attribution: el.dataset.attribution || TILES_CREDIT,
			maxZoom: 11,
			minZoom: 1,
		}).addTo(map);

		var group = L.featureGroup();
		points.forEach(function (point) {
			L.circleMarker([point.lat, point.lng], {
				radius: Number(el.dataset.markerRadius || 6),
				color: el.dataset.markerStroke,
				weight: 2,
				fillColor: el.dataset.markerFill,
				fillOpacity: 0.75,
				// Leaflet focuses these with the keyboard, so the map is
				// operable without a pointer.
				keyboard: true,
			})
				.bindPopup(popup(point))
				.addTo(group);
		});
		group.addTo(map);

		map.fitBounds(group.getBounds(), { padding: [32, 32], animate: false });
		if (el.dataset.maxZoom && map.getZoom() > Number(el.dataset.maxZoom)) {
			map.setZoom(Number(el.dataset.maxZoom));
		}

		// Wheel zoom stays off until the map is deliberately focused, so
		// the page never traps a scroll on the way past.
		map.on("focus", function () {
			map.scrollWheelZoom.enable();
		});
		map.on("blur", function () {
			map.scrollWheelZoom.disable();
		});
	}

	function start() {
		document.querySelectorAll("[data-sightings-map]").forEach(init);
	}

	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", start);
	} else {
		start();
	}
})();
