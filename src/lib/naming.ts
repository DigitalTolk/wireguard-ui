/**
 * Apply a regex pattern + replacement template to an input string.
 *
 * Returns the transformed string when the pattern matches, otherwise an
 * empty string. An empty or invalid pattern also returns an empty string
 * so callers can use the falsy result as a "no-op" signal.
 */
export function applyNamePattern(
  input: string,
  pattern: string,
  replacement: string,
): string {
  if (!pattern) return "";
  let re: RegExp;
  try {
    re = new RegExp(pattern);
  } catch {
    return "";
  }
  if (!re.test(input)) return "";
  return input.replace(re, replacement);
}
