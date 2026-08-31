package com.robword.service;

import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class ClozeAnswerComparisonTest {

    @Test
    void preservesBlankIndexesAndNormalizesUnicodeCase() {
        ClozeAnswerComparison comparison = ClozeAnswerComparison.compare(
                List.of("ＲＡＷ", "", "Fracture"),
                List.of("raw", "momentum", "fracture")
        );

        assertFalse(comparison.correct());
        assertEquals(List.of(1), comparison.wrongIndexes());
    }

    @Test
    void marksEveryExpectedBlankWrongWhenAnswerCountDiffers() {
        ClozeAnswerComparison comparison = ClozeAnswerComparison.compare(
                List.of("raw", "momentum", "fracture", "extra"),
                List.of("raw", "momentum", "fracture")
        );

        assertFalse(comparison.correct());
        assertEquals(List.of(0, 1, 2), comparison.wrongIndexes());
    }

    @Test
    void reportsCorrectOnlyWhenEveryIndexedAnswerMatches() {
        ClozeAnswerComparison comparison = ClozeAnswerComparison.compare(
                List.of(" raw ", "Momentum"),
                List.of("raw", "momentum")
        );

        assertTrue(comparison.correct());
        assertEquals(List.of(), comparison.wrongIndexes());
    }
}
