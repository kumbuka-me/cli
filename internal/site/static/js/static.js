// Static-site entry point. Initializes read-only Kumbuka UI features only.
import { initTheme } from "./core/theme.js";
import { initMarkdown } from "./features/markdown.js";
import { initStaticLayout } from "./features/static-layout.js";
import { initStaticPage } from "./features/static-page.js";
import { initStaticSearch } from "./features/static-search.js";
initTheme();
initStaticLayout();
initStaticPage();
initStaticSearch();
await initMarkdown();
