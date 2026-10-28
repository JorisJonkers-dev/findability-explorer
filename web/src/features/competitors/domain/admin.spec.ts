import { describe, expect, it } from "vitest";
import {
  ARCHIVE_ACTION,
  canRemove,
  checkCompanyForm,
  companyProblem,
  leftOut,
  removeAction,
  rescanCandidates,
  rescanProgress,
  rescanSummary,
  splitArchived,
  unscannedCandidates,
} from "./admin";

describe("archiving or removing a company", () => {
  it("archives any company, and deletes only an added one with its scans", () => {
    expect(ARCHIVE_ACTION.button).toBe("Archive");
    expect(canRemove({ source: "research" })).toBe(false);
    expect(canRemove({ source: "added" })).toBe(true);
    expect(removeAction({ scans: 1 })).toEqual({
      button: "Delete with its 1 scan",
      question: "Delete this company with its 1 scan? This cannot be undone.",
    });
    expect(removeAction({ scans: 0 }).button).toBe("Delete with its 0 scans");
  });
});

describe("the company form", () => {
  it("checks the name and the website", () => {
    expect(checkCompanyForm(" Example ", "example.com")).toEqual({
      kind: "ok",
      name: "Example",
      url: "https://example.com",
    });
    expect(checkCompanyForm("", "example.com")).toEqual({
      kind: "invalid",
      field: "name",
      message: "Enter the company’s name.",
    });
    expect(checkCompanyForm("x".repeat(201), "example.com")).toMatchObject({
      field: "name",
      message: "A name has at most 200 characters.",
    });
    expect(checkCompanyForm("Example", "ftp://example.com")).toMatchObject({
      field: "url",
      message: "Only http and https addresses can be scanned.",
    });
  });

  it("says why the server refused", () => {
    expect(
      companyProblem({ type: "about:blank", title: "Forbidden", status: 403 }),
    ).toBe("Only administrators can change the companies.");
    expect(
      companyProblem({
        type: "about:blank",
        title: "Conflict",
        status: 409,
        detail: "Twentebelt already has this website.",
      }),
    ).toBe("Twentebelt already has this website.");
    expect(companyProblem(new Error("boom"))).toBe(
      "Something went wrong. Try again.",
    );
  });
});

describe("a re-scan", () => {
  it("takes every company with data: researched, or scanned before, with a website and not archived", () => {
    const c = (
      id: string,
      source: "research" | "added",
      scans: number,
      websiteUrl?: string,
      archived = false,
    ) => ({ id, source, scans, websiteUrl, archived });
    const companies = [
      c("researched", "research", 0, "https://r.com"),
      c("scanned", "added", 2, "https://s.com"),
      c("never-scanned", "added", 0, "https://n.com"),
      c("no-site", "research", 0),
      c("archived", "research", 3, "https://a.com", true),
      {
        ...c("blocked", "research", 1, "https://b.com"),
        lastScan: { blocked: true },
      },
    ];
    expect(rescanCandidates(companies).map((x) => x.id)).toEqual([
      "researched",
      "scanned",
    ]);
  });

  it("takes the companies with a website no scan reached yet, and says which ones a sweep leaves out", () => {
    const c = (
      id: string,
      scans: number,
      websiteUrl?: string,
      archived = false,
      blocked = false,
    ) => ({
      id,
      source: "research" as const,
      scans,
      websiteUrl,
      archived,
      ...(blocked ? { lastScan: { blocked: true } } : {}),
    });
    const companies = [
      c("new", 0, "https://n.com"),
      c("done", 2, "https://d.com"),
      c("no-site", 0),
      c("no-site-2", 0),
      c("gone", 0, "https://g.com", true),
      c("wall", 1, "https://w.com", false, true),
    ];
    expect(unscannedCandidates(companies).map((x) => x.id)).toEqual(["new"]);
    expect(leftOut(companies)).toBe(
      "Left out of both: 2 companies without a website, 1 whose site blocks the crawler.",
    );
    expect(leftOut([c("no-site", 0)])).toBe(
      "Left out of both: 1 company without a website.",
    );
    expect(leftOut([c("a", 0, "https://a.com")])).toBeNull();
  });

  it("follows the queued scans until every one has finished", () => {
    const scan = (
      id: string,
      status: "queued" | "running" | "done" | "failed",
    ) => ({ id, status });
    const queued = ["a", "b", "c", "d"];
    const midway = rescanProgress(queued, [
      scan("a", "done"),
      scan("b", "running"),
      scan("c", "queued"),
      scan("d", "queued"),
      scan("x", "running"),
    ]);
    expect(midway).toEqual({
      finished: false,
      text: "Re-scanning 4 companies: 1 done, 1 running, 2 waiting.",
      total: 4,
      settled: 1,
      fraction: 0.25,
      running: [scan("b", "running")], // x runs too, but is not one of the re-scan's
      waiting: 2,
    });
    expect(
      rescanProgress(queued, [
        scan("a", "done"),
        scan("b", "failed"),
        scan("c", "done"),
        scan("d", "done"),
      ]),
    ).toMatchObject({
      finished: true,
      text: "Re-scan finished: 3 done, 1 failed.",
      settled: 4,
      fraction: 1,
      running: [],
    });
    // A scan the list no longer holds cannot be followed, so it does not keep the line waiting.
    expect(rescanProgress(["a", "gone"], [scan("a", "done")]).finished).toBe(
      true,
    );
    expect(rescanProgress([], [])).toMatchObject({
      finished: true,
      fraction: 1,
    });
    expect(
      rescanProgress(["a", "b"], [scan("a", "failed"), scan("b", "running")])
        .text,
    ).toBe("Re-scanning 2 companies: 0 done, 1 failed, 1 running, 0 waiting.");
    expect(rescanProgress(["a"], [scan("a", "queued")]).text).toBe(
      "Re-scanning 1 company: 0 done, 0 running, 1 waiting.",
    );
  });

  it("says what was queued and skipped", () => {
    expect(rescanSummary({ queued: [], skipped: [] })).toBe("Queued 0 scans.");
    const one = {
      id: "a",
      url: "https://a.com",
      status: "queued" as const,
      maxPages: 60,
      pagesCrawled: 0,
      createdAt: "2026-10-23T09:30:00Z",
    };
    expect(
      rescanSummary({
        queued: [one],
        skipped: [
          { competitorId: "b", name: "B", reason: "It has no website." },
        ],
      }),
    ).toBe("Queued 1 scan. Skipped 1: B (it has no website).");
  });

  it("keeps archived companies apart", () => {
    expect(
      splitArchived([
        { id: "a", archived: false },
        { id: "b", archived: true },
      ]),
    ).toEqual({
      active: [{ id: "a", archived: false }],
      archived: [{ id: "b", archived: true }],
    });
  });
});
