// Client-side enhancements for rendered Markdown.
import { renderPluginModules } from "../plugins/loader.js";
import { setupCopyButton } from "../core/clipboard.js";
// Wires markdown tabs behavior.
function setupMarkdownTabs(group) {
    const tabs = [
        ...group.querySelectorAll(":scope > .markdown-tab-list > .markdown-tab"),
    ];
    const panels = [
        ...group.querySelectorAll(":scope > .markdown-tab-panels > .markdown-tab-panel"),
    ];
    if (!tabs.length || tabs.length !== panels.length)
        return;
    function activate(index, focus = false) {
        for (const [candidateIndex, tab] of tabs.entries()) {
            const active = candidateIndex === index;
            tab.classList.toggle("active", active);
            tab.setAttribute("aria-selected", String(active));
            tab.tabIndex = active ? 0 : -1;
            panels[candidateIndex].classList.toggle("markdown-tab-panel-hidden", !active);
        }
        if (focus)
            tabs[index]?.focus();
    }
    for (const [index, tab] of tabs.entries()) {
        tab.addEventListener("click", () => activate(index));
        tab.addEventListener("keydown", (event) => {
            let next = index;
            switch (event.key) {
                case "ArrowRight":
                    next = (index + 1) % tabs.length;
                    break;
                case "ArrowLeft":
                    next = (index - 1 + tabs.length) % tabs.length;
                    break;
                case "Home":
                    next = 0;
                    break;
                case "End":
                    next = tabs.length - 1;
                    break;
                default:
                    return;
            }
            event.preventDefault();
            activate(next, true);
        });
    }
    const initial = Math.max(0, tabs.findIndex((tab) => tab.classList.contains("active")));
    activate(initial);
}
export function setupMarkdownEnhancements(root = document) {
    for (const group of root.querySelectorAll(".markdown-tabs"))
        setupMarkdownTabs(group);
    setupCodeCopyButtons(root);
}
function setupCodeCopyButtons(root = document) {
    for (const pre of root.querySelectorAll(".prose pre")) {
        const code = pre.querySelector(":scope > code");
        if (!code ||
            Boolean(pre.closest("[data-kumbuka-plugin]")) ||
            pre.parentElement?.classList.contains("code-block"))
            continue;
        const wrapper = document.createElement("div");
        wrapper.className = "code-block";
        pre.before(wrapper);
        wrapper.append(pre);
        const button = document.createElement("button");
        button.type = "button";
        button.className = "code-copy-button";
        setupCopyButton(button, () => code.textContent ?? "", "Copy code to clipboard");
        wrapper.append(button);
    }
}
// Initializes markdown.
export async function initMarkdown() {
    setupMarkdownEnhancements(document);
    await renderPluginModules(document);
}
