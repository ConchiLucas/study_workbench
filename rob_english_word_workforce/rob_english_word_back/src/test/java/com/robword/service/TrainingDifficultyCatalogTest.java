package com.robword.service;

import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class TrainingDifficultyCatalogTest {

    private final TrainingDifficultyCatalog catalog = new TrainingDifficultyCatalog();

    @Test
    void resolvesRankCurrentAsRankBasedDifficulty() {
        TrainingDifficultyCatalog.Difficulty difficulty = catalog.resolve("rank", "rank_current").orElseThrow();

        assertTrue(difficulty.rankBased());
        assertEquals("段位难度", difficulty.label());
        assertTrue(difficulty.libraryNames().isEmpty());
    }

    @Test
    void resolvesExactJuniorChildLibraries() {
        TrainingDifficultyCatalog.Difficulty difficulty = catalog.resolve("junior", "junior_7_1").orElseThrow();

        assertFalse(difficulty.rankBased());
        assertEquals("初中英语 · 7年级上册", difficulty.label());
        assertEquals(List.of("PEPChuZhong7_1"), difficulty.libraryNames());
    }

    @Test
    void resolvesJuniorGroupToAllJuniorLibraries() {
        TrainingDifficultyCatalog.Difficulty difficulty = catalog.resolve("junior", "junior").orElseThrow();

        assertEquals(
                List.of("PEPChuZhong7_1", "PEPChuZhong7_2", "PEPChuZhong8_1", "PEPChuZhong8_2", "PEPChuZhong9_1"),
                difficulty.libraryNames()
        );
    }

    @Test
    void resolvesEveryDifficultyExposedByTheFrontend() {
        List<List<String>> selections = List.of(
                List.of("primary", "primary"),
                List.of("primary", "primary_3_1"),
                List.of("primary", "primary_6_2"),
                List.of("junior", "junior"),
                List.of("junior", "junior_9_1"),
                List.of("senior", "senior"),
                List.of("senior", "senior_11"),
                List.of("college", "college"),
                List.of("college", "college_cet4"),
                List.of("college", "college_cet6"),
                List.of("entrance", "entrance"),
                List.of("entrance", "entrance_kaoyan"),
                List.of("business_abroad", "business_abroad"),
                List.of("business_abroad", "business_bec"),
                List.of("business_abroad", "business_ielts"),
                List.of("business_abroad", "business_toefl"),
                List.of("business_abroad", "business_gmat"),
                List.of("professional", "professional"),
                List.of("professional", "professional_tem4"),
                List.of("professional", "professional_tem8"),
                List.of("advanced_exam", "advanced_exam"),
                List.of("advanced_exam", "advanced_gre"),
                List.of("advanced_exam", "advanced_sat")
        );

        for (List<String> selection : selections) {
            assertTrue(catalog.resolve(selection.get(0), selection.get(1)).isPresent(), selection.toString());
        }
    }

    @Test
    void rejectsGroupAndLevelFromDifferentBranches() {
        assertTrue(catalog.resolve("junior", "senior_1").isEmpty());
    }

    @Test
    void rejectsUnknownAndBlankDifficulty() {
        assertTrue(catalog.resolve("unknown", "unknown_1").isEmpty());
        assertTrue(catalog.resolve(null, "junior_7_1").isEmpty());
        assertTrue(catalog.resolve("junior", "").isEmpty());
    }
}
