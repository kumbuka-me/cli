// DOM helpers for markup that is required by a component template contract.
export function requiredElement(root, selector) {
    const element = root.querySelector(selector);
    if (!element)
        throw new Error(`Missing required element: ${selector}`);
    return element;
}
export function requiredElements(root, selector) {
    const elements = [...root.querySelectorAll(selector)];
    if (!elements.length)
        throw new Error(`Missing required elements: ${selector}`);
    return elements;
}
export function requiredAttribute(element, name) {
    const value = element.getAttribute(name)?.trim();
    if (!value)
        throw new Error(`Missing required attribute: ${name}`);
    return value;
}
