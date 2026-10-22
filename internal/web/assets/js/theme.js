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
