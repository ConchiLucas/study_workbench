import { describe, expect, it } from "vitest";
import {
  DEFAULT_SOLO_DIFFICULTY,
  SOLO_DIFFICULTY_OPTIONS,
  normalizeSoloDifficulty,
} from "../src/lib/soloDifficulty";

describe("solo difficulty", () => {
  it("falls back to junior when the stored preference is rank", () => {
    expect(
      normalizeSoloDifficulty({
        soloDifficultyGroup: "rank",
        soloDifficultyLevel: "rank_current",
      }),
    ).toEqual(DEFAULT_SOLO_DIFFICULTY);
  });

  it("does not expose rank as a solo option", () => {
    expect(SOLO_DIFFICULTY_OPTIONS.some((option) => option.key === "rank")).toBe(false);
  });
});
