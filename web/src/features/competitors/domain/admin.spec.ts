import { describe, expect, it } from "vitest";
import {
  canRemove,
  checkCompanyForm,
  companyProblem,
  removeAction,
  rescanProgress,
  rescanSummary,
} from "./admin";

describe("archiving or removing a company", () => {
  it("deletes only an added company, with its scans", () => {
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
});
