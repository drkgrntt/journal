// Loaded as a plain, non-module <script> at the very top of <head> (see
// base.templ) - unlike assets/js/index.js (type="module", deferred until
// the document is parsed), this runs immediately and blocks parsing, so
// the [data-theme] attribute and the theme-color meta are both set before
// anything paints. Without that, a saved "dark" override would flash the
// light theme for a moment on every load.
//
// Only an explicit choice needs the attribute at all - with nothing saved,
// the plain :root/@media(prefers-color-scheme) rules in theme/colors.css
// already follow the OS preference on their own.
(function () {
	applyThemeColorMeta(currentEffectiveTheme());
	var saved = localStorage.getItem("theme");
	if (saved === "light" || saved === "dark") {
		document.documentElement.setAttribute("data-theme", saved);
	}
})();

// Appearance (palette + font, the paid "Themes" feature; see theme/
// appearance.css). The server renders data-palette/data-font on <html> for a
// user with an active plan, and marks logged-in pages with
// data-appearance-sync. Online, that markup is the truth and is mirrored to
// localStorage here. Offline, the page may be the service worker's cached
// /journal/new copy, which can predate the latest choice, so the mirrored
// value wins. Only the allowlisted tokens are ever stored or applied.
// Never throws: storage can be blocked, and this runs before first paint.
var APPEARANCE_KEY = "appearance";
var PALETTES = ["lavender", "sky", "rose", "sand"];
var FONTS = ["serif"];

function setAppearance(palette, font) {
	var root = document.documentElement;
	if (PALETTES.indexOf(palette) === -1) palette = "";
	if (FONTS.indexOf(font) === -1) font = "";
	if (palette) root.setAttribute("data-palette", palette);
	else root.removeAttribute("data-palette");
	if (font) root.setAttribute("data-font", font);
	else root.removeAttribute("data-font");
	try {
		localStorage.setItem(APPEARANCE_KEY, JSON.stringify({ palette: palette, font: font }));
	} catch (e) {}
	// The installed app's system-bar color comes from the manifest.
	var link = document.getElementById("manifest-link");
	if (link) link.setAttribute("href", "/manifest.webmanifest" + (palette ? "?palette=" + palette : ""));
	// Charts bake their colors in when built.
	document.dispatchEvent(new Event("themechange"));
}

(function () {
	try {
		var root = document.documentElement;
		if (!root.hasAttribute("data-appearance-sync")) return;
		var palette = root.getAttribute("data-palette") || "";
		var font = root.getAttribute("data-font") || "";
		if (navigator.onLine === false) {
			var saved = JSON.parse(localStorage.getItem(APPEARANCE_KEY) || "null");
			if (saved) {
				palette = saved.palette;
				font = saved.font;
			}
			if (PALETTES.indexOf(palette) === -1) palette = "";
			if (FONTS.indexOf(font) === -1) font = "";
			if (palette) root.setAttribute("data-palette", palette);
			else root.removeAttribute("data-palette");
			if (font) root.setAttribute("data-font", font);
			else root.removeAttribute("data-font");
		} else {
			localStorage.setItem(APPEARANCE_KEY, JSON.stringify({ palette: palette, font: font }));
		}
	} catch (e) {}
})();

function currentEffectiveTheme() {
	var saved = localStorage.getItem("theme");
	if (saved === "light" || saved === "dark") {
		return saved;
	}
	return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

// Keeps the address-bar/task-switcher chrome color (mobile Safari/Chrome)
// in sync with the actual theme. A static <meta name="theme-color"
// media="..."> can only ever follow the OS preference, not an explicit
// [data-theme] override, so the single meta tag in base.templ is updated
// from here instead. Values match --surface in theme/colors.css for each
// theme.
function applyThemeColorMeta(theme) {
	var meta = document.getElementById("theme-color-meta");
	if (!meta) return;
	meta.setAttribute("content", theme === "dark" ? "#212526" : "#ffffff");
	applyNativeStatusBar(theme);
}

// In the native shell (mobile/), the status bar icons follow the system theme
// unless told otherwise, so they can disappear against the page. Match them to
// the page's theme. A no-op in a browser.
function applyNativeStatusBar(theme) {
	try {
		var cap = window.Capacitor;
		if (!cap || !cap.isNativePlatform || !cap.isNativePlatform()) return;
		// A page loaded from a server URL has no bundled plugin imports, so the
		// native plugin has to be registered by name first.
		var bar = (cap.Plugins && cap.Plugins.StatusBar) || cap.registerPlugin("StatusBar");
		bar.setStyle({ style: theme === "dark" ? "DARK" : "LIGHT" });
	} catch (e) {}
}

// The header's theme toggle (see header.templ) is a simple two-state
// switch, not a three-way system/light/dark picker - it starts wherever
// the system preference currently resolves to (no saved override yet) and
// flips to an explicit choice from there. Which icon shows is handled
// entirely by CSS (components/header.css, keyed off the same [data-theme]
// attribute/media query as the rest of the palette), not JS, so it can
// never drift out of sync with the actual applied theme.
function toggleTheme() {
	var next = currentEffectiveTheme() === "dark" ? "light" : "dark";
	document.documentElement.setAttribute("data-theme", next);
	localStorage.setItem("theme", next);
	applyThemeColorMeta(next);
	// Charts bake their colors in when built, so they need to be told.
	document.dispatchEvent(new Event("themechange"));
}

// Chart.js's own default tick/legend/grid color is a fixed dark gray, so it
// goes near-invisible on a dark surface. Points the defaults at the active
// theme's palette; called on load (base.templ) and on every theme switch.
// Chart.js re-reads the defaults on its next update(), so live charts follow
// once refreshChartTheme (utils/index.js) updates them.
function applyChartDefaults() {
	if (!window.Chart) return;
	var themeStyle = getComputedStyle(document.documentElement);
	Chart.defaults.color = themeStyle.getPropertyValue("--text-secondary-color").trim();
	Chart.defaults.borderColor = themeStyle.getPropertyValue("--border-color").trim();
}
