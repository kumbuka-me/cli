// Theme selection and persistence.
import { requiredElement } from "./dom.js";
import { isRecord, isStringRecord, requireArrayOf } from "./guards.js";
function isColorScheme(value) {
    return value === "light" || value === "dark";
}
function isThemeDefinition(value) {
    return (isRecord(value) &&
        typeof value.title === "string" &&
        value.title.trim() !== "" &&
        isColorScheme(value.color_scheme) &&
        isStringRecord(value.colors));
}
export function parseThemeCatalog(source) {
    let value;
    try {
        value = JSON.parse(source);
    }
    catch (error) {
        throw new Error("Invalid theme catalog JSON.", { cause: error });
    }
    return requireArrayOf(value, isThemeDefinition, "theme catalog");
}
function findTheme(catalog, title) {
    const normalized = title.toLocaleLowerCase();
    return catalog.find((candidate) => candidate.title.toLocaleLowerCase() === normalized);
}
function applyTheme(catalog, select, fallbackTitle, title) {
    const theme = findTheme(catalog, title) ??
        findTheme(catalog, fallbackTitle) ??
        catalog[0];
    if (!theme)
        throw new Error("Theme catalog is empty.");
    for (const [key, value] of Object.entries(theme.colors)) {
        document.documentElement.style.setProperty(`--${key.replaceAll("_", "-")}`, value);
    }
    document.documentElement.style.colorScheme = theme.color_scheme;
    document.documentElement.dataset.theme = theme.title;
    if (select)
        select.value = theme.title;
}
// Initializes theme.
export function initTheme() {
    const source = requiredElement(document, "#kumbuka-themes");
    const catalog = parseThemeCatalog(source.textContent ?? "");
    const select = document.querySelector("[data-theme-select]");
    const activeTheme = document.documentElement.dataset.theme || "Light";
    applyTheme(catalog, select, activeTheme, activeTheme);
    select?.addEventListener("change", () => applyTheme(catalog, select, activeTheme, select.value));
}
