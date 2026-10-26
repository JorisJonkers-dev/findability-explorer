import type {
  AnalysisSummary,
  Competitor,
  RescanBatch,
} from "../../../infrastructure/api/types.gen";
import { zProblem } from "../../../infrastructure/api/zod.gen";
import { describeError } from "../../../shared/describeError";
import { checkScanUrl, describeScanUrlCheck } from "../../scan/domain/newScan";

export const MAX_NAME_LENGTH = 200;

/** What removing a company does, in the words its confirm button uses. */
export function removeAction(company: Pick<Competitor, "source" | "scans">): {
  button: string;
  question: string;
} {
  if (company.source === "research") {
    return {
      button: "Archive",
      question:
        "Archive this researched company? It leaves the list and the re-scans; its scans stay, and it can be restored.",
    };
  }
  const scans =
    company.scans === 1 ? "1 scan" : `${String(company.scans)} scans`;
  return {
    button: `Delete with its ${scans}`,
    question: `Delete this company with its ${scans}? This cannot be undone.`,
  };
}

/** A company form, checked: the values to send, or what is wrong with them. */
export type CompanyFormCheck =
  | { kind: "ok"; name: string; url: string }
  | { kind: "invalid"; field: "name" | "url"; message: string };

/** Checks a company's name and website before they are sent. */
export function checkCompanyForm(
  name: string,
  website: string,
): CompanyFormCheck {
  const trimmed = name.trim();
  if (trimmed === "")
    return {
      kind: "invalid",
      field: "name",
      message: "Enter the company’s name.",
    };
  if (trimmed.length > MAX_NAME_LENGTH) {
    return {
      kind: "invalid",
      field: "name",
      message: `A name has at most ${String(MAX_NAME_LENGTH)} characters.`,
    };
  }
  const check = checkScanUrl(website);
  const message = describeScanUrlCheck(check);
  if (check.kind !== "ok")
    return { kind: "invalid", field: "url", message: String(message) };
  return { kind: "ok", name: trimmed, url: check.url };
}

/** A refused change to a company: a 409 names the company that has the website already. */
export function companyProblem(error: unknown): string {
  const problem = zProblem.safeParse(error);
  if (problem.success && problem.data.status === 403)
    return "Only administrators can change the companies.";
  return describeError(error);
}

/** What a re-scan did: "Queued 12 scans. Skipped 2: Märtens (a scan of it is still running)." */
export function rescanSummary(batch: RescanBatch): string {
  const queued = `Queued ${String(batch.queued.length)} ${batch.queued.length === 1 ? "scan" : "scans"}.`;
  if (batch.skipped.length === 0) return queued;
  const skipped = batch.skipped
    .map((s) => `${s.name} (${s.reason.replace(/\.$/, "").toLowerCase()})`)
    .join(", ");
  return `${queued} Skipped ${String(batch.skipped.length)}: ${skipped}.`;
}

/** Archived researched companies apart from the rest, so only administrators see them. */
export function splitArchived<T extends Pick<Competitor, "archived">>(
  companies: readonly T[],
): { active: T[]; archived: T[] } {
  return {
    active: companies.filter((c) => !c.archived),
    archived: companies.filter((c) => c.archived),
  };
}

/**
 * The companies a re-scan of all takes, as the server picks them: not archived, with a website,
 * and with data, because the desk research studied them or a scan ran before; not one whose site
 * turned its last scan away.
 */
export function rescanCandidates<
  T extends Pick<Competitor, "source" | "scans" | "websiteUrl" | "archived"> & {
    lastScan?: Pick<AnalysisSummary, "blocked">;
  },
>(companies: readonly T[]): T[] {
  return companies.filter(
    (c) =>
      !c.archived &&
      c.websiteUrl !== undefined &&
      (c.source === "research" || c.scans > 0) &&
      c.lastScan?.blocked !== true,
  );
}

/** The companies scanning the unscanned takes, as the server picks them: not archived, with a website no scan reached yet. */
export function unscannedCandidates<
  T extends Pick<Competitor, "scans" | "websiteUrl" | "archived">,
>(companies: readonly T[]): T[] {
  return companies.filter(
    (c) => !c.archived && c.websiteUrl !== undefined && c.scans === 0,
  );
}

/** Which companies the sweeps leave out and why, in one line, or null when none is. */
export function leftOut(
  companies: readonly (Pick<Competitor, "websiteUrl" | "archived"> & {
    lastScan?: Pick<AnalysisSummary, "blocked">;
  })[],
): string | null {
  const active = companies.filter((c) => !c.archived);
  const noSite = active.filter((c) => c.websiteUrl === undefined).length;
  const blocked = active.filter((c) => c.lastScan?.blocked === true).length;
  const parts = [
    noSite > 0
      ? `${String(noSite)} ${noSite === 1 ? "company" : "companies"} without a website`
      : "",
    blocked > 0 ? `${String(blocked)} whose site turns scanners away` : "",
  ].filter((p) => p !== "");
  return parts.length === 0 ? null : `Left out of both: ${parts.join(", ")}.`;
}

/** Where a re-scan's scans are. */
export type RescanProgress<T> = {
  /** True once no scan of the re-scan is waiting or running. */
  finished: boolean;
  /** The counts in one line. */
  text: string;
  total: number;
  /** How many have finished, done or failed. */
  settled: number;
  /** settled of total, from 0 to 1. */
  fraction: number;
  /** The scans reading pages now, in the order they were queued. */
  running: T[];
  waiting: number;
};

/** Where a re-scan's scans are: counts, the share finished, and the scans running now. */
export function rescanProgress<
  T extends Pick<AnalysisSummary, "id" | "status">,
>(queued: readonly string[], scans: readonly T[]): RescanProgress<T> {
  const byId = new Map(scans.map((s) => [s.id, s]));
  const count = { queued: 0, running: 0, done: 0, failed: 0 };
  const running: T[] = [];
  for (const id of queued) {
    const s = byId.get(id);
    if (s === undefined) continue;
    count[s.status]++;
    if (s.status === "running") running.push(s);
  }
  const settled = count.done + count.failed;
  const total = queued.length;
  const shared = {
    total,
    settled,
    fraction: total === 0 ? 1 : settled / total,
    running,
    waiting: count.queued,
  };
  const failed = count.failed > 0 ? `, ${String(count.failed)} failed` : "";
  if (count.queued + count.running === 0) {
    return {
      ...shared,
      finished: true,
      text: `Re-scan finished: ${String(count.done)} done${failed}.`,
    };
  }
  const companies = total === 1 ? "1 company" : `${String(total)} companies`;
  return {
    ...shared,
    finished: false,
    text: `Re-scanning ${companies}: ${String(count.done)} done${failed}, ${String(count.running)} running, ${String(count.queued)} waiting.`,
  };
}
