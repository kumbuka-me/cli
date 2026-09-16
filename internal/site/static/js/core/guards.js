// Runtime guards for values crossing browser trust boundaries.
export function isRecord(value) {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}
export function isStringRecord(value) {
    if (!isRecord(value))
        return false;
    return Object.values(value).every((item) => typeof item === "string");
}
export function requireArrayOf(value, guard, description) {
    if (!Array.isArray(value) || !value.every(guard))
        throw new Error(`Invalid ${description}.`);
    return value;
}
