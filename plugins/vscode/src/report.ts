// Pure helpers for the developer report (no VS Code APIs, so they are unit-tested).
/** Takes personal details out of a report: the home folder and e-mail addresses. */
export function scrubReport(text: string, home: string): string {
  let out = home && home.length > 1 ? text.split(home).join("~") : text;
  out = out.replace(/[\w.+-]+@[\w-]+(\.[\w-]+)+/g, "<email>");
  return out;
}
